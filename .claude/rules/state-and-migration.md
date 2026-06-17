---
paths:
  - "migration/**/*.go"
  - "types/**/*.go"
  - "common/**/*.go"
  - "proto/**/*.proto"
  - "x/*/types/**/*.go"
  - "x/*/module.go"
  - "app/app.go"
---

# State, Types & Migration Security Review

Changes to shared types, state migration, and proto definitions can silently corrupt chain state or break consensus across validators running different versions.

## External Attack Vectors

- **Malicious tx submitter** (anyone): crafts messages with fields that exploit `ValidateBasic()` gaps -- zero-value addresses, maximum-length strings, negative amounts that wrap unsigned, or proto messages with unexpected `oneof` variants. If these reach keepers and corrupt state, all nodes are affected.
- **Governance attacker** (validator coalition): proposes param changes that set critical values to zero (span duration, checkpoint interval, min stake) or extremes, triggering division-by-zero panics or infinite loops in ABCI handlers. Params without bounds validation are exploitable.
- **Upgrade-time attacker**: if a migration has a non-deterministic bug, the attacker waits for the upgrade height and submits transactions that exercise the buggy path, causing some validators to produce different post-migration state than others -- splitting the network.

## State Migration

- Migrations must be deterministic -- all validators must produce identical post-migration state from identical pre-migration state
- Never delete or rename store keys without a migration path from the old key -- orphaned data silently persists and new keys start empty
- Test migrations against real chain state exports (not just genesis) -- production state has edge cases genesis doesn't
- Verify migration order: module migrations run in the order set by `SetOrderMigrations()` in `app.go`. If not explicitly set, defaults to module registration order -- implicit ordering is fragile.
- Off-by-one errors in block height migration triggers can skip or double-apply migrations -- verify the exact upgrade height
- Migrations that fail midway leave the store in a partially migrated state -- consider writing a migration version marker before and after to detect incomplete migrations
- State migrations run during `InitChainer` on upgrade -- they block the chain until complete. Estimate runtime for large state sets.

## Store Keys and Module Accounts

- Adding new store keys requires registration in `app.go` (`KVStoreKeys`) -- missing keys cause panics at runtime, not compile time
- Each module must use a unique store key prefix -- KV prefix collisions between modules silently overwrite each other's data
- Module accounts (e.g., for fee collection, staking pool) hold real tokens. Verify:
  - Only the owning module can mint/burn from its module account
  - Module account permissions (`Minter`, `Burner`, `Staking`) match intended behavior
  - No code path allows unauthorized transfers from module accounts

## Proto Definitions

- Never change field numbers in existing proto messages -- this silently breaks decoding of stored state and network messages
- Never remove fields -- mark as `reserved` with both the field number and name
- Adding new fields is safe only if all consumers handle the zero value correctly (empty string, 0, nil)
- `oneof` fields: adding a new variant is safe, but changing or removing existing variants breaks decoding
- Proto changes require `make proto-all` and verification that generated Go code compiles and tests pass
- Enum values must never be reordered or renumbered -- add new values at the end only
- `google.protobuf.Any` fields: verify the type URL is validated on unmarshal to prevent type confusion attacks

## Shared Types (`types/`, `x/*/types/`)

- Changes to message types affect wire encoding -- all validators must agree on serialization. A mismatch causes the chain to fork.
- Validate all new message fields in `ValidateBasic()` -- this is the first line of defense before messages reach keepers
- `ValidateBasic()` must be pure (no state access, no external calls) and deterministic
- Events and error types must not leak sensitive validator information (private key material, internal IP addresses)
- Genesis import/export must be round-trip safe: `export -> import` produces identical state. Test this explicitly.
- Custom Amino registrations (if any remain) must match Protobuf definitions -- Cosmos SDK v0.50 uses Protobuf for state storage, but Amino may still be used for legacy signing in some paths. Encoding mismatches between Amino and Protobuf are consensus-breaking.

## Common Utilities (`common/`)

- Cache invalidation bugs can serve stale data to consensus-critical code paths -- if a cache is used in a keeper, verify invalidation happens at the correct block boundaries (epoch, span, checkpoint interval)
- Hex encoding/decoding must handle edge cases: odd-length strings, `0x` prefix vs no prefix, empty input, mixed case
- Tracing context propagation must not affect determinism -- trace IDs must not be included in state hashes or consensus messages
- String utility functions used on external data must handle malformed UTF-8 and control characters

## Governance Parameter Changes

- Some module params can be changed via governance proposals at runtime -- verify that all param values are bounds-checked
- Critical params (checkpoint interval, span duration, minimum stake, voting power thresholds) must have min/max bounds to prevent governance attacks
- Param changes that affect consensus must be applied at epoch/span boundaries, not mid-block

## Red Flags

- Changing proto field numbers or removing fields without `reserved`
- Migrations without state verification (compare pre/post state roots)
- Shared type changes without updating `ValidateBasic()`
- Cache changes without considering consensus determinism
- New store keys not registered in `app.go`
- Module account permission changes without thorough review
- Missing round-trip test for genesis import/export after type changes
- Governance-changeable params without bounds validation
