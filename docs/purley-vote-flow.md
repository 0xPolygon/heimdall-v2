# Purley vote and event flow

This document covers the signed vote transport and bridge event confirmation
activated at the Purley hardfork. The checkpoint buffer, stake nonce admission,
checkpoint ACK vote and governance cancellation changes of the same hardfork are
not described here.

## Invariants

1. A vote's transport validity depends on its signed bytes, height, block hash,
   chain, round, and validator key. From the first proposal that carries Purley
   votes, neither checkpoint state nor Bor availability decides whether a signed
   envelope can be transported. Checkpoint continuity and L1 validation remain in
   the side and post handlers.
2. A committing opinion cannot be replaced by an empty placeholder. The signed
   RP extension commits to the NR bytes; neither independently signed NR data
   from another height nor stripped payloads may change the opinion being tallied.
3. Late valid votes are accepted during CometBFT's existing commit timeout. They
   use the previous validator set and the same stateless envelope checks as early
   votes. A block precommit quorum is not necessarily a side-tx YES quorum.
4. The bridge relays a validator's staking events one nonce at a time. A task
   whose nonce is ahead of heimdall's polls on the nonce gap; a task whose nonce
   heimdall already used ends, since the side handler can never apply it.
5. Any other valid decoded bridge task completes only when the event's persistent
   effect is observed. An OK CheckTx response, elapsed time, or an API outage
   cannot complete the task. Both confirmation and transient-failure retries
   bypass Machinery's finite ordinary retry budget.

## Implementation boundaries

Purley is disabled (`0`) on mainnet, Mumbai and Amoy, and starts at `1` on devnets.
The height remains a coordinated rollout decision.

At vote height H, the Purley envelope adds the NR hash and enforces an RP ceiling
of 6 KiB and an NR ceiling of 500 bytes. Comet authenticates signatures before
calling the app, also for late precommits, except for copies LastCommit already
holds. The app uses no SDK state or RPC for Purley envelope validation. All valid
delayed opinions remain available.

From the same vote height the RP extension is compact: it omits `height`, which
the extension signature binds together with the round, and carries only the
first 8 bytes of the voted block hash. Its milestone proposition carries an
8-byte parent hash prefix and omits `latest_block_hash` when the head is the
proposition tail, and the NR placeholder shrinks to a `0x00` marker, the 8-byte
height and the chain id, so the envelope's NR floor is the compact one. The
envelope compares the prefix with the voted block in VerifyVoteExtension, and
with the prefix the first committing vote carries in a proposal. The legacy
placeholder filter only handles votes signed before H, so it never sees the
compact format.

A proposal at H+1 transports the original signed Purley votes. PrepareProposal,
ProcessProposal and PreBlocker share the commit authenticator; the PreBlocker
skips the signature checks ProcessProposal already made. A proposer whose
LastCommit fails authentication, which only a block-synced commit can cause,
still proposes, because CometBFT panics on a PrepareProposal error, and validators
reject the proposal. Completeness is
checked against the block's canonical commit, power comes from the historical
validator set, and both signatures are verified before deriving a local view.
If a milestone proposition is unusable under the current params, only that
proposition is omitted from the local view; its side votes and the transported
bytes remain unchanged. Derived bytes are never used as signature evidence.

The first activation proposal (height H) still carries legacy votes from H-1
and takes the legacy path: PrepareProposal filters them, and ProcessProposal runs
the existing vote extension checks plus `ValidateNonRpVoteExtensions`, including
its checkpoint-state and Bor-dependent validation, under the existing failure
policy. Signed envelope transport starts at H+1, so pre-H votes never need the
new field. Pre-H binaries and upgraded binaries reject the new field on pre-H
wire payloads, including explicit empty fields.

## Accepted tradeoffs

- Consecutive events for one validator take separate approval cycles. Other
  validators' events can still be processed concurrently. Future nonces stay out
  of the mempool, avoiding speculative nonce tracking and free future-nonce fill.
- Side approval remains strictly greater than two-thirds of total canonical
  power. Absent votes are not converted into approval. Omission or network delay
  can defer an effect; bridge confirmation retries provide recovery when honest
  proposals and external services resume. This is not a multi-block voting or
  absolute censorship-resistance protocol.
- State-sync, topup and checkpoint-ACK polling backs off to at most 384 seconds;
  staking polling is described below. Missing valid events remain queued
  indefinitely, including during
  API/broadcast outages; retries past the self-heal depth are counted in
  `event_retries_past_self_heal_depth_total` for alerting.
  Each bridge keeps one live task per L1 event: a self-heal re-enqueue of an event
  another task still owns ends on its first run. After an accepted broadcast the
  task keeps polling but resends the event only once per interval of the
  self-heal pass that used to resend it (by default 30 min for checkpoint ACKs,
  3 h for state syncs, staking events and topups): without the tx indexer,
  which the shipped configs disable, a dropped tx can't be told apart from a
  committed one whose side tx failed. An event that never applies therefore
  costs what self-heal cost before, and a dropped tx is resent within that
  interval instead of after the 24 h self-heal depth. Ownership and the last broadcast live in memory, so a
  restart resends at once.
  Queue storage and occasional repeated fees are the cost of retaining work.
  Malformed JSON/ABI tasks retain ordinary bounded failure handling. Operators
  must retain the queue; this does not recover tasks lost before this upgrade.
- Stake tasks poll every 15 s per pending nonce (capped at 150 s) instead of
  backing off, and every 15 s once broadcast.
- Mempool duplicate detection is a bounded best-effort scan of 100 transactions.
  A pending transaction suppresses a broadcast only when every event field
  matches, ignoring the relayer address. Idempotent on-chain event checks remain
  authoritative.
- Full authenticated transport needs room for the entire bounded certificate.
  At 104 validators, the 6 KiB RP limit plus NR, signatures and metadata fits
  below 1 MiB, leaving room in the existing 2 MiB block configuration even with
  the maximum evidence allowance. Increasing the validator count or reducing
  block capacity requires checking this bound. It is no longer safe to solve
  capacity problems by stripping an honest validator's opinion.

## Validation

The regression suite covers original envelopes, stripped payloads, changed NR
bytes, both signatures, canonical power, duplicates, signed height/hash changes,
late approvals, milestone isolation, fork wire compatibility, the activation
boundary, mempool duplicate matching, stake nonce polling and durable retry
behavior.
Full application tests, affected-package race tests, Comet consensus/state/types
race tests, pinned lint, protobuf generation, and the Docker build pass. A six-block
pre-fork replay produces the same app hashes as the original base. The govulncheck
scan reports the same reachable advisories as the base; these are not resolved by
this branch. The mutation quality gate remains open: the complete-suite run kills
306/379 mutations (80.7% overall; 86.4% T1 logic), below the required threshold.

### Local end-to-end run: 2026-10-28

Kurtosis completed successfully with four validators, one RPC node, Bor v2.10.3,
and an Anvil L1. All Heimdall nodes used the locally built image
`heimdall-v2:vote-flow-local` (image digest
`sha256:99a5aa96e6f9661154044bb52af1a0b375f68df9ff2843be479d19c8841fab22`).
The fork activates at height 1 on this devnet. The image was built from the
combined Purley stack, not from this branch alone.

- POL, ERC20 and ERC721 deposits increased Heimdall records and Bor's
  `lastStateId` from 8 to 11. The ERC721 owner's L2 balance increased by one.
- Two consecutive L1 restakes for validator 1 advanced its stored nonce
  2 → 3 → 4 and voting power 10025 → 10026 → 10027.
- A real four-validator certificate contains all four signed envelopes and
  matching RP-to-NR hash bindings. Pausing validator 4 for twelve blocks yields
  three signed committing envelopes and continued consensus; it then resumes.
- Pausing validators 3 and 4 removes quorum: Heimdall remains at height 624
  while an L1 deposit emits event 12. Restoring both validators resumes consensus,
  records event 12 on Heimdall, and advances Bor's `lastStateId` to 12. The record's
  L1 transaction hash matches the deposit receipt.
- After recovery, all five nodes agree on finalized Heimdall block/app hashes
  and Bor block hashes. Checkpoint ACKs advance; an acknowledged checkpoint's
  root and range exactly match the L1 contract.

This run exercises live transport, normal cross-chain processing and temporary
outages. It does not reproduce malicious proposer payloads or a voted-down event;
those scenarios remain regression-test coverage. Anvil does not exercise Ethereum
PoS finality. The earlier geth/lighthouse setup reached L2 startup but timed out
during contract setup, so it is not counted as a passing end-to-end run.
