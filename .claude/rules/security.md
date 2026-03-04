# Security Review Rules

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
2. Classify severity: CRITICAL (consensus break, fund loss), HIGH (validator manipulation, DoS), MEDIUM (info leak, degraded security), LOW (hardening opportunity)
3. For CRITICAL/HIGH: flag immediately, do not merge, recommend fix before any other work
4. Check the entire codebase for similar patterns
5. If secrets were exposed: rotate immediately, check git history with `git log -p --all -S 'SECRET_VALUE'`
