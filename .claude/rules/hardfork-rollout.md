---
paths:
  - "helper/config.go"
  - "app/**/*.go"
  - "sidetxs/**/*.go"
  - "x/**/keeper/*.go"
  - "x/milestone/abci/**/*.go"
  - "x/*/module.go"
  - "x/*/types/**/*.go"
  - "migration/**/*.go"
  - "types/**/*.go"
  - "proto/**/*.proto"
  - "common/**/*.go"
---
# Hardfork Rollout Review

Heimdall hardforks and height-gated behavior changes are consensus-critical.
They affect validator agreement, app hash, Bor coordination, checkpointing,
milestones, spans, side transactions, or store migrations. Review them as
rollout wiring changes, not isolated code edits.

## Trigger Conditions

Apply this rule whenever a change:

- Adds or changes a hardfork height, named network height, or config getter in
  `helper/config.go`.
- Gates ABCI behavior, side/post handlers, keeper writes, milestone logic,
  checkpoint logic, span selection, validator-set updates, or vote extension
  behavior by height.
- Changes module `ConsensusVersion`, migrations, store keys, proto state,
  genesis import/export, or replay behavior for existing chain state.
- Introduces Bor/Heimdall coordination logic where a Heimdall height maps to a
  Bor block, span, sprint, milestone, or checkpoint boundary.

## Required Wiring

For every new or changed fork, verify all of these surfaces before approving:

- Mainnet, Amoy, and local/devnet values in `helper/config.go` are present,
  named consistently, and intentionally different where needed.
- Getters, setters, config structs, CLI/config loading, and tests all read the
  same height. Avoid one path using a zero value while another path has the
  intended network value.
- ABCI handlers are symmetric: proposer-side filtering in `PrepareProposal`
  must be validated by all validators in `ProcessProposal`; vote-extension
  generation in `ExtendVote` must be matched by deterministic checks in
  `VerifyVoteExtension`; deterministic writes belong in `PreBlocker` or
  module keepers.
- Height checks use the correct chain domain. Be explicit about Heimdall block
  height, Bor block number, `height`, `height-1`, initial height, span start,
  sprint start, milestone end, and checkpoint boundaries.
- Module `ConsensusVersion`, migrations, store key registration, migration
  order, proto generated code, and genesis export/import are updated together
  when the fork changes stored state.
- Existing chain state can replay through the fork. A fresh genesis-only test
  is not enough for migrations or post-handler state changes.

## Determinism Requirements

- Deterministic paths must not use external RPC calls, wall clock time, random
  data, goroutines, channel races, or unsorted map iteration.
- `PrepareProposal` may be proposer-specific, but its output must be accepted
  or rejected deterministically by `ProcessProposal` on every validator.
- `VerifyVoteExtension`, `ProcessProposal`, `PreBlocker`, post-handlers, and
  keeper writes must make identical decisions from identical inputs on every
  validator.
- Post-handler state changes that would alter the result of replaying old
  blocks require explicit fork gating. Without the gate, upgraded validators
  can produce a different app hash from the same historical block.

## Rollout Checks

- [ ] Does `git diff` show a new fork name, hardcoded height, or new
      height-gated branch? If yes, list each network height and each runtime
      path that loads it.
- [ ] Are mainnet, Amoy, and devnet/local values wired through the same config
      path, or is any network silently left at zero?
- [ ] Are boundary tests present for `H-1`, `H`, and `H+1`, including any
      Bor-block-to-Heimdall-height conversion?
- [ ] If ABCI logic changed, do `PrepareProposal`, `ProcessProposal`,
      `ExtendVote`, `VerifyVoteExtension`, and `PreBlocker` remain mutually
      consistent?
- [ ] If stored state changed, are `ConsensusVersion`, migrations, store keys,
      proto definitions, generated code, and genesis import/export covered?
- [ ] Is there a replay, migration, or app-hash-style test using pre-fork state
      rather than only fresh genesis state?
- [ ] During mixed-version rollout, is pre-fork behavior unchanged until the
      activation height, and is rollback behavior understood before activation?
