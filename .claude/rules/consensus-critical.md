---
paths:
  - "app/**/*.go"
  - "sidetxs/**/*.go"
  - "x/checkpoint/**/*.go"
  - "x/milestone/**/*.go"
  - "x/bor/**/*.go"
---

# Consensus-Critical Code Review

This code directly affects consensus. Bugs here can halt the chain, cause forks, or enable fund theft. Review with extreme caution.

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
- **`VerifyVoteExtension`**: per-validator, MUST be deterministic for vote validity. All validators must agree on whether a vote extension is valid. No external network calls.
- **`ProcessProposal`**: deterministic validation. All validators must accept or reject the same proposal identically. No external network calls.
- **`PreBlocker`**: deterministic execution. Tallies votes, runs post-handlers, applies approved side-txs. No external network calls. State writes happen here.
- **`PrepareProposal`**: proposer-only, may filter/reorder txs. Not required to be deterministic with other validators, but output must pass `ProcessProposal`.

Confusing these guarantees is the #1 source of consensus bugs.

## Vote Extension Security

- Validate ALL fields of incoming vote extensions: block height, block hash, proto encoding
- Reject vote extensions with unknown proto fields -- CometBFT's proto unmarshaling ignores unknown fields by default, which can be used to smuggle data
- Check for duplicate validator votes in the same round
- Use canonical voting power from the validator set at height H-1 (penultimate block), NEVER trust power values from `ExtendedCommitInfo`
- Verify vote extension signatures against the canonical public key set at H-1
- Enforce 2/3+ voting power threshold for side-tx approval (use `> 2/3` not `>= 2/3` -- match CometBFT's convention)
- Enforce the single-side-msg-per-tx invariant to prevent vote hash collisions
- `VoteExtensionsEnableHeight` in CometBFT config determines when VEs activate -- code must handle blocks before this height where VEs are absent

## PrepareProposal / ProcessProposal

- Both handlers must enforce identical validation rules -- any divergence causes consensus failure
- PrepareProposal filters invalid vote extensions; ProcessProposal rejects the entire proposal if any invalid VE is included
- VEBLOP conditions (MsgVoteProducers, MsgSetProducerDowntime) must be checked consistently in both handlers
- PrepareProposal chooses transaction order; ProcessProposal must validate any valid ordering (don't assume order)

## PreBlocker Security

- Tallying logic must use canonical validator set from H-1, never trust caller-provided voting power
- 2/3 majority threshold for milestone acceptance, 1/3 for pending status -- verify exact threshold math (off-by-one in voting power comparison is a consensus vulnerability)
- Checkpoint signature aggregation must validate each signature individually
- Post-handlers for approved side-txs execute state changes -- they MUST be deterministic and MUST NOT make external calls
- Post-handlers should be safe to crash-recover (node restarts mid-block replay the entire block)

## Panic Safety

- Panics in any ABCI handler crash the node and can halt the chain if triggered for all validators
- Guard against nil pointer dereference: proto message fields, RPC responses, interface values
- Guard against index out of range: vote extension arrays, validator lists, event logs
- Guard against division by zero: voting power calculations, span duration math
- Use `defer func() { recover() }()` only as a last resort -- prefer explicit nil/bounds checks

## Computation Bounds

- ABCI handlers (PrepareProposal, ProcessProposal, PreBlocker) have no Cosmos SDK gas metering
- Unbounded computation causes CometBFT timeouts and consensus stalls
- Bound iteration over vote extensions, side-tx responses, and validator sets
- Maximum 50 side-tx responses per vote extension -- validate this bound before iterating

## Side Transaction Invariants

- No duplicate tx hashes in side-tx responses
- Valid vote types only (YES, NO, UNSPECIFIED)
- `SideTxDecorator` must enforce at most one side-tx message per transaction
- Side handlers (in `ExtendVote`) produce per-validator opinions -- disagreement is normal
- Post-handlers (in `PreBlocker`) execute only for txs with 2/3+ approval -- must be deterministic

## Red Flags -- Reject Immediately

- Any change that skips vote extension validation
- Removing or weakening the 2/3 voting power threshold
- Adding non-deterministic operations in `ProcessProposal`, `PreBlocker`, or post-handlers
- Adding external network calls in `ProcessProposal`, `VerifyVoteExtension`, or `PreBlocker`
- Trusting unverified data from `ExtendedCommitInfo` or `RequestPrepareProposal`
- Modifying state directly in ABCI handlers instead of through keepers
- Changes to tallying logic without corresponding test updates
- Unguarded array/slice indexing on external data in ABCI handlers
