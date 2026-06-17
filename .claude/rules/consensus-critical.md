---
paths:
  - "app/**/*.go"
  - "sidetxs/**/*.go"
  - "x/**/keeper/*.go"
  - "x/milestone/abci/**/*.go"
  - "x/*/module.go"
---

# Consensus-Critical Code Review

This code directly affects consensus. Bugs here can halt the chain, cause forks, or enable fund theft. Review with extreme caution.

## External Attack Vectors

These are the actors who can trigger consensus bugs **remotely** -- making any exploitable issue CRITICAL:

- **Malicious proposer** (1 of ~100 validators): crafts proposals with malformed VEs, invalid tx ordering, or oversized payloads. If this triggers a panic in `ProcessProposal` on ALL honest validators, the chain halts. The proposer sacrifices their own block but stops the network.
- **Malicious validator**: sends crafted vote extensions (malformed proto, oversized NonRpVE, duplicate votes, smuggled unknown fields). If `VerifyVoteExtension` panics or produces different results on different honest nodes, consensus breaks.
- **External tx submitter** (anyone): submits transactions designed to exploit side-tx handlers, ante decorators, or keeper logic. If a crafted message causes non-deterministic state writes across validators, app hash diverges.
- **Colluding minority** (<1/3 validators): submits conflicting vote extensions or milestones to exploit threshold edge cases (e.g., non-deterministic map iteration when multiple values meet 1/3 threshold).

**Key question for every finding: "Can a malicious proposer/validator/user trigger this on honest nodes?"** If yes, it is CRITICAL regardless of how unlikely. Self-inflicted bugs (misconfiguration, code bug affecting only the buggy node) are lower severity.

**Every module keeper change is consensus-critical.** Divergences in keeper state propagate to the app layer. If two nodes run different states, they produce the infamous **app hash mismatch error**, halting heimdall block production and impacting bor (finality, checkpoints, network liveness). Even if all nodes run the same version, a change in state management applied to previous blocks triggers app hash mismatch -- this is why **hard forks are required** for state-divergence changes (e.g., post-handler modifications).

## Determinism Requirements

- All validators MUST produce identical results from identical inputs
- Never use maps for iteration order (use sorted slices) -- Go randomizes map iteration intentionally
- Never use `time.Now()` -- use `ctx.BlockTime()` for all timestamps
- Never use goroutines or channels in deterministic consensus paths (`ProcessProposal`, `PreBlocker`, post-handlers)
- Never use floating point arithmetic -- use `math.LegacyDec` (Cosmos SDK's decimal type)
- Random number generation must use deterministic seeds (span seed from L1, not local randomness)
- `fmt.Sprintf("%v", map)` produces non-deterministic output -- never use map-derived strings in state or hashing

## ABCI++ Handler Semantics (CometBFT v0.38 / Cosmos SDK v0.50)

Understand which handlers are deterministic vs non-deterministic:

- **`ExtendVote`**: per-validator, MAY make external RPC calls (L1, Bor), MAY be non-deterministic. This is where side-handlers run and produce validator-specific opinions.
- **`VerifyVoteExtension`**: per-validator, MUST be deterministic for vote validity. All validators must agree on whether a vote extension is valid. No external network calls. Should mirror the validation logic of `ExtendVote` -- every check in `ExtendVote` should have a corresponding verification in `VerifyVoteExtension`.
- **`ProcessProposal`**: deterministic validation. All validators must accept or reject the same proposal identically. No external network calls. **All validation checks from `PrepareProposal` must also be executed in `ProcessProposal`** -- if the proposer validates something, all validators must validate it too.
- **`PreBlocker`**: deterministic execution. State writes happen here at two levels:
  - **App-level PreBlocker** (`app/abci.go`): tallies votes, runs post-handlers for approved side-txs, processes milestone propositions, handles checkpoint signatures
  - **Module-level PreBlockers**: invoked by the app-level PreBlocker at the end. The **stake module** performs validator set updates here. Changes to module PreBlockers are as critical as app-level ones.
- **`PrepareProposal`**: proposer-only, may filter/reorder txs. Not required to be deterministic with other validators, but output must pass `ProcessProposal`. **Malformed VEs must be filtered out here** (not halt consensus), but explicitly rejected during ProcessProposal.

**Every error returned by an ABCI method triggers a panic in CometBFT, which can halt the chain.** Errors must be justified by strong backing reasons (e.g., proposal is provably invalid). When in doubt, log and continue rather than return an error.

## Vote Extension Security

- Validate ALL fields of incoming vote extensions: block height, block hash, proto encoding
- Reject vote extensions with unknown proto fields -- CometBFT's proto unmarshaling ignores unknown fields by default, which can be used to smuggle data
- Check for duplicate validator votes in the same round
- Use canonical voting power from the validator set at height H-1 (penultimate block), NEVER trust power values from `ExtendedCommitInfo`
- Verify vote extension signatures against the canonical public key set at H-1
- Enforce 2/3+ voting power threshold for side-tx approval (use `> 2/3` not `>= 2/3` -- match CometBFT's convention)
- Enforce the single-side-msg-per-tx invariant to prevent vote hash collisions
- **`VoteExtensionsEnableHeight`**: heimdall-v2 launched with VEs always enabled (final v1 block + 1). This value MUST NEVER be changed -- modifying it would be catastrophic for consensus.
- Enforce explicit size bounds on VEs and NonRpVEs -- filter/reject oversized and undersized extensions. Size params are defined in module params and enforced in PrepareProposal/ProcessProposal.

## PrepareProposal / ProcessProposal

- **All checks in PrepareProposal must be mirrored in ProcessProposal.** If the proposer validates something, all validators must validate it identically.
- PrepareProposal **filters** invalid/malformed vote extensions (silently drops them to avoid halting consensus). ProcessProposal **rejects** the entire proposal if invalid VEs are included (the proposer should have filtered them).
- VEBLOP conditions (MsgVoteProducers, MsgSetProducerDowntime) must be checked consistently in both handlers
- PrepareProposal chooses transaction order; ProcessProposal must validate any valid ordering (don't assume order)

## Milestone Module ABCI (`x/milestone/abci/`)

The milestone module has its own ABCI implementation requiring particular attention:

- Milestone propositions affect **bor finality** -- incorrect milestones compromise the entire finality guarantee of the Polygon PoS stack
- `GenMilestoneProposition` makes external RPC calls to Bor (allowed in ExtendVote)
- `GetMajorityMilestoneProposition` runs in PreBlocker (deterministic path) -- map iterations MUST be sorted
- Milestone acceptance (2/3 majority) vs pending status (1/3) have different thresholds with different safety properties
- Stalled milestones trigger span rotation -- incorrect milestone logic cascades into producer selection failures

## PreBlocker Security

- State writes happen at **both** app level and module level:
  - App-level: vote tallying, side-tx post-handlers, milestone processing, checkpoint signatures
  - Module-level: stake module performs **validator set updates** via `ApplyAndReturnValidatorSetUpdates()`, chainmanager migrates contract addresses at specific heights
- Tallying logic must use canonical validator set from H-1, never trust caller-provided voting power
- 2/3 majority threshold for milestone acceptance, 1/3 for pending status -- verify exact threshold math (off-by-one in voting power comparison is a consensus vulnerability)
- Checkpoint signature aggregation must validate each signature individually
- Post-handlers for approved side-txs execute state changes -- they MUST be deterministic and MUST NOT make external calls
- Post-handlers should be safe to crash-recover (node restarts mid-block replay the entire block)
- **Any change to post-handler state logic applied to previous blocks requires a hard fork** -- you cannot change how past blocks are processed without causing app hash mismatch

## Panic Safety

- Panics in any ABCI handler crash the node and can halt the chain if triggered for all validators
- **Any error returned from an ABCI method triggers a CometBFT panic** -- only return errors when you have strong justification (provably invalid proposal, corrupted state). Prefer logging + graceful degradation over error returns.
- Guard against nil pointer dereference: proto message fields, RPC responses, interface values
- Guard against index out of range: vote extension arrays, validator lists, event logs
- Guard against division by zero: voting power calculations, span duration math
- Use `defer func() { recover() }()` only as a last resort -- prefer explicit nil/bounds checks

## Computation Bounds

- ABCI handlers (PrepareProposal, ProcessProposal, PreBlocker) have no Cosmos SDK gas metering
- Unbounded computation causes CometBFT timeouts and consensus stalls
- Bound iteration over vote extensions, side-tx responses, and validator sets
- Maximum 50 side-tx responses per vote extension -- validate this bound before iterating
- Enforce explicit size bounds on VEs and NonRpVEs: filter/reject oversized and undersized extensions at PrepareProposal, reject at ProcessProposal

## Side Transaction Invariants

- No duplicate tx hashes in side-tx responses
- Valid vote types: YES and NO are actively used. UNSPECIFIED exists in the proto definition but is not actively used in code -- treat it as an abstain/no-op, do not assume it carries semantic meaning.
- `SideTxDecorator` must enforce at most one side-tx message per transaction
- Side handlers (in `ExtendVote`) produce per-validator opinions -- disagreement is normal
- Post-handlers (in `PreBlocker`) execute only for txs with 2/3+ approval -- must be deterministic
- **`side_msg_server.go` in every module is especially critical** -- these define the side and post handlers that directly write state. Any change here can cause app hash divergence.

## Red Flags -- Reject Immediately

- Any change that skips vote extension validation
- Removing or weakening the 2/3 voting power threshold
- Adding non-deterministic operations in `ProcessProposal`, `PreBlocker`, or post-handlers
- Adding external network calls in `ProcessProposal`, `VerifyVoteExtension`, or `PreBlocker`
- Trusting unverified data from `ExtendedCommitInfo` or `RequestPrepareProposal`
- Modifying state directly in ABCI handlers instead of through keepers
- Changes to tallying logic without corresponding test updates
- Unguarded array/slice indexing on external data in ABCI handlers
- Changes to post-handler state logic without planning a hard fork
- Modifying `VoteExtensionsEnableHeight`
- Any ABCI error return without strong justification (it triggers a CometBFT panic)
