# Security Review Rules

## Threat Model -- External vs Self-Inflicted

When classifying severity, always consider **who can trigger the bug**:

| Attacker | Example | Severity Multiplier |
|---|---|---|
| **External user/tx submitter** | Crafted transaction crashes all validators | **Highest** -- anyone can attack, no permissions needed |
| **Malicious validator/proposer** | Proposal with crafted VEs triggers panic in ProcessProposal on all honest nodes | **Critical** -- 1 of ~100 validators can halt the chain |
| **Malicious RPC provider** | Fabricated L1 responses cause wrong side-tx votes | **High** -- trusted dependency, but external |
| **Adjacent service** | RabbitMQ message injection forges bridge events | **High** -- requires network access |
| **Malicious peer** | Eclipse attack, gossip flooding | **High** -- any P2P participant |
| **Node misconfiguration** | Wrong chain ID, bad RPC URL | **Lower** -- operator error, self-inflicted |
| **Code bug (no external trigger)** | Race condition, memory leak | **Lower** -- affects only the buggy node |

**Key principle**: A bug that a single external actor (proposer, validator, user, peer) can trigger to crash/corrupt ALL honest nodes is always CRITICAL, regardless of how unlikely the scenario seems. A bug that only affects the node running the bad code is lower severity.

When reviewing, ask: **"Can someone else make my node do this?"** If yes, severity goes up.

## Pre-Commit Security Checklist

Before approving or completing any code change, verify:

- No hardcoded secrets, private keys, mnemonics, or RPC endpoints
- No logging of sensitive data (private keys, validator key material, keyring contents)
- All external inputs validated before use (L1 receipts, event logs, RPC responses)
- Nonce/sequence checks present for replay protection
- Error messages do not leak internal state or validator identity
- New dependencies audited for known CVEs (`govulncheck ./...`)
- Tests pass with `-race` flag to detect data races

## Secret Management

- Secrets come from environment variables or keyring, never from source code
- Config files with secrets must be in `.gitignore`
- Never log private keys, mnemonics, keyring passphrases, or signing material at any log level
- Checkpoint signatures and vote extension payloads may be logged at Debug level only

## Go Security Patterns

- Use `crypto/rand` for all randomness, never `math/rand` in any security-relevant code
- Never use the `unsafe` package in consensus or crypto paths
- Bound all loops and slices that process external data (vote extensions, event logs, validator lists) -- unbounded input causes OOM/DoS
- Use `context.WithTimeout` for all RPC calls to L1 and Bor
- Check `err != nil` immediately after every call -- do not defer error handling
- Use `math.Int` (from `cosmossdk.io/math`) for all arithmetic involving token amounts or voting power -- `sdk.Int` is deprecated in Cosmos SDK v0.50, and native `int64`/`uint64` overflow silently
- Guard against log injection: external data in log messages can contain newlines that forge log entries -- use structured logging (zerolog fields), not string interpolation
- Protect against nil pointer dereference in all paths that handle RPC responses, proto messages, or interface values -- panics in ABCI handlers crash the node

## Dependency Security

- Forked dependencies (`cosmos-sdk`, `cometbft`, `bor`) must pin to exact commit hashes via `replace` directives in `go.mod`
- Run `go mod verify` to ensure module checksums match
- Run `govulncheck ./...` to check for known vulnerabilities
- Review `go.sum` changes in every PR -- unexpected checksum changes indicate supply chain risk
- Verify that `replace` directives point to the expected 0xPolygon fork repositories, not third-party forks

## Security Response Protocol

When a security issue is found during review:

1. **STOP** -- do not continue with the change
2. Classify severity using both impact AND attacker model:
   - **CRITICAL**: externally triggerable consensus break or fund loss (malicious proposer/validator/user can halt chain or steal funds)
   - **HIGH**: externally triggerable DoS or validator manipulation (malicious peer/RPC/queue can degrade network)
   - **MEDIUM**: self-inflicted consensus risk or externally triggerable info leak
   - **LOW**: self-inflicted degradation, hardening opportunity
3. For CRITICAL/HIGH: flag immediately, do not merge, recommend fix before any other work
4. Check the entire codebase for similar patterns
5. If secrets were exposed: rotate immediately, check git history with `git log -p --all -S 'SECRET_VALUE'`
