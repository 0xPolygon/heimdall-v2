# Bor endpoint failover

Heimdall talks to Bor over HTTP JSON-RPC and (optionally) gRPC. Both transports
support **multiple priority-ordered endpoints with automatic failover and
switchback**, so a single Bor outage doesn't stall Heimdall's Bor-dependent
reads (span/producer lookups, root hashes, milestone votes, block headers).

The logic is a transport-agnostic state machine in
[`x/bor/failover`](../x/bor/failover), shared by the HTTP client
([`helper/bor_failover_http.go`](../helper/bor_failover_http.go)) and the gRPC
client ([`x/bor/grpc/failover.go`](../x/bor/grpc/failover.go)).

## Configuration

`bor_rpc_url` and `bor_grpc_url` accept a **comma-separated, priority-ordered**
list. Index 0 is the primary; the rest are fallbacks in descending priority.

```toml
# Failover across three Bor RPC endpoints (first = primary).
bor_rpc_url = "http://localhost:8545,https://bor-b.internal:8545,https://bor-c.internal:8545"

bor_grpc_flag  = "true"
# Remote gRPC endpoints need an explicit http:// or https:// scheme; a bare
# host:port is accepted only for localhost.
bor_grpc_url   = "localhost:3131,https://bor-b.internal:3131"
bor_grpc_token = "" # Bearer token for gRPC auth; put gRPC credentials here, not in the URL.

# Per-attempt timeout (also the unit the in-call budget is built from).
# Clamped to MaxBorRPCTimeout (3s) at config load so the in-call budget stays
# under CometBFT's ~10s ABCI window.
bor_rpc_timeout = "1s"
```

A **single endpoint keeps the previous behavior exactly**: a plain dial, no
background prober, no failover machinery.

## Block producer restriction

Bor endpoint failover is rejected at startup for Heimdall nodes whose local
signer maps to a validator ID in the protected block-producer set. The
protected set is built from the configured producer list (`producer_votes`,
defaulting by network, e.g. mainnet `91,92,93,94`), the current span's selected
producer(s), and the current VEBLOP producer candidate set.

This restriction is intentionally narrower than "all validators": active
validators outside the protected producer set may use Bor failover. Block
producers must use a single local/self Bor endpoint so their milestone votes do
not mask local Bor downtime and keep them eligible for span rotation when they
cannot actually produce blocks.

HTTP endpoint URLs may carry credentials (userinfo or `?apikey=...`); these are
masked in the HTTP failover logs — userinfo password and query values, though a
path-embedded token cannot be detected generically. gRPC authentication uses the
separate `bor_grpc_token` (a Bearer token), not the URL; the gRPC client logs its
configured address as-is, so do not embed secrets in `bor_grpc_url`.

## Identity validation

A fallback is **never served traffic until a probe confirms its chain identity**
matches the expected chain:

- HTTP validates the **chain ID** (`eth_chainId`).
- gRPC validates the **genesis block hash** (the gRPC API has no chain-id call)
  and, when HTTP Bor is available, a recent-block **HTTP/gRPC hash parity**
  check. This keeps a same-chain but wrong-version gRPC fallback from becoming a
  candidate when its recent header hashes diverge from HTTP.

This is the guard against a misconfigured fallback that points at the wrong Bor
network feeding bad data into consensus-critical reads.

The **primary owns the expected identity whenever it is reachable**:

- Normally the primary establishes it (captured best-effort at startup, then on
  its first probe).
- If the primary has **never** been reachable, a reachable fallback may
  *provisionally* anchor the expectation — but only after the primary has failed
  `primaryAnchorFailureThreshold` (2) consecutive probes — so failover still
  engages when the primary is down at boot. Probes run concurrently, so whichever
  reachable fallback validates first wins the anchor; this is not
  priority-ordered.
- Once the primary answers, it **reclaims** the expectation, becomes active
  immediately, and **demotes every other endpoint** (they must re-validate
  against the reclaimed identity before they can be used again). The real
  primary is therefore never rejected, and any fallback that had validated
  against a provisional identity is dropped from the candidate set at once.

The only window in which a wrong-network fallback can serve is one where the
primary has *never* been reachable. Once the primary is reachable even once,
that window closes.

## Failover rules

The active endpoint always receives the request first (even if it is currently
flagged unhealthy — the flag governs candidacy and proactive switching, not
whether the current active is tried).

### In-call cascade

On a **retriable** failure of the active endpoint, the call cascades to the next
validated endpoint and promotes the first that succeeds.

- **HTTP** cascades on a transport error, a per-attempt timeout, or an HTTP
  **5xx**. It does **not** cascade on a 4xx — that is returned as-is (a fallback
  would answer identically). Application-level JSON-RPC errors arrive as HTTP 200
  and are not failover triggers. If every endpoint returns 5xx, the last real
  response is preserved and returned.
- **gRPC** cascades only on transport-style status codes: `Unavailable`,
  `DeadlineExceeded`, `ResourceExhausted`. Logical errors (NotFound / validation,
  surfacing as `Unknown`) are returned as-is, never retried.

Candidate selection:

- Only **currently-healthy** endpoints (identity-validated and not recently
  failed) are candidates.
- Ordered **cooled first, then uncooled, each in priority (index) order** — a
  cooled higher-priority fallback is preferred over an uncooled one within a
  single call.
- A failed endpoint is marked unhealthy (dropped from candidates until a probe
  re-confirms it). The first candidate that succeeds is **promoted to active**,
  so subsequent calls start there until switchback.
- The cascade stops early if the **caller context is done** (deadline or
  cancellation).

### Time budget

- Per-attempt timeout is `bor_rpc_timeout`, clamped at config load to
  `MaxBorRPCTimeout` (`3s`).
- The caller-side budget for one Bor call is:

  ```
  budget = bor_rpc_timeout × clamp( bor_grpc_flag ? max(httpCount, grpcCount) : httpCount , 1, maxBudgetedEndpoints )
  ```

  with `maxBudgetedEndpoints = 3`. This is a **time budget, not a hard attempt
  counter**: the deadline stops the cascade, so fast-failing endpoints beyond the
  cap may still be tried within the budget, while slow ones beyond it are reached
  on later calls (by then the prober has typically switched active away from a
  dead endpoint). The cap keeps one call's worst case under CometBFT's ~10s ABCI
  budget; because `bor_rpc_timeout` is itself clamped to `MaxBorRPCTimeout`
  (`maxBorChainCallBudget / maxBudgetedEndpoints` = `3s`), the worst-case budget
  is bounded at `9s` regardless of the configured value, so a slow Bor can't
  stall a milestone/checkpoint vote extension into a missed vote. When
  `bor_grpc_flag = true` the budget is sized by the larger of the
  HTTP and gRPC counts (the HTTP client used by the broadcaster and the gRPC
  client used by side handlers share it); when gRPC is disabled it is the HTTP
  count alone.

`SendTransaction` (the bridge broadcaster) also rides the HTTP failover client; a
cascade re-broadcasts the same signed transaction, which is idempotent (same tx
hash), so failover on writes is safe.

### Proactive switch (background prober)

Independently of any call, the background prober switches the active endpoint off
an **unhealthy** active before the next call has to discover the failure. It
moves to the best healthy alternative: highest priority, preferring a cooled
endpoint but using an **uncooled** one in an emergency. If nothing healthy
exists, it stays put (it never routes to an unvalidated endpoint).

## Switchback rules

- **Reclaim (immediate):** when the primary recovers and reclaims chain identity,
  it becomes active at once, bypassing the threshold and cooldown (and demoting
  the other endpoints). This is the only immediate switchback.
- **Ordinary same-network recovery (gated):** the prober reverts to a recovered
  higher-priority endpoint only after it has accrued `consecutiveThreshold` (3)
  consecutive successful probes **and** stayed healthy for the
  `promotionCooldown` (60s). It reverts to the **highest-priority** endpoint
  (scanning from index 0 up to the active index) that qualifies.

The asymmetry is deliberate: **fail away fast, switch back slow.** Moving down to
a healthy fallback is cheap; moving back up to a higher-priority endpoint waits
out the cooldown to avoid flapping. Switching *off* a dead active (proactive
switch) may use an uncooled endpoint in an emergency; reverting *up* to a merely
recovered endpoint requires it to be cooled.

Because reclaim demotes all other endpoints, a correct same-chain fallback that
was demoted is briefly unavailable for in-call failover (~`consecutiveThreshold`
probes ≈ 30s) until it re-validates — deliberately chosen over ever serving a
possibly-wrong-network fallback.

## Timing constants

Sources vary: `bor_rpc_timeout` is operator config (clamped to `MaxBorRPCTimeout`
at load); `MaxBorRPCTimeout` and `maxBudgetedEndpoints` are constants in
[`helper/bor_failover_http.go`](../helper/bor_failover_http.go); the remaining
prober defaults are hardcoded in
[`x/bor/failover/health.go`](../x/bor/failover/health.go) and are not
operator-tunable (`SetTuning` exists for tests only).

| Constant | Default | Meaning |
| --- | --- | --- |
| `bor_rpc_timeout` (config) | `1s` | Per-attempt timeout (clamped to `MaxBorRPCTimeout`); unit of the in-call budget |
| `MaxBorRPCTimeout` | `3s` | Upper bound enforced on `bor_rpc_timeout` (`maxBorChainCallBudget / maxBudgetedEndpoints`) |
| `maxBudgetedEndpoints` | `3` | Cap on the in-call time budget multiplier |
| `DefaultCheckInterval` | `10s` | Background prober cycle period |
| `DefaultConsecutiveThreshold` | `3` | Consecutive good probes for a fallback to become healthy |
| `DefaultPromotionCooldown` | `60s` | Continuous-health duration before revert-to-higher-priority |
| `DefaultProbeTimeout` | `3s` | Per-probe timeout (separate from `bor_rpc_timeout`) |
| `primaryAnchorFailureThreshold` | `2` | Consecutive primary probe failures before a fallback may provisionally anchor identity |

Each prober cycle runs `probeAll → maybePromote → maybeProactiveSwitch`.

## Expected latencies (defaults)

| Event | Latency |
| --- | --- |
| Reactive fail-over off a failing active | within one call (budget ≥ 2 endpoints) |
| Proactive switch off a dead active | ≤ ~1 cycle once a healthy alternative exists; otherwise after fallback validation (~30s) |
| A recovered fallback becomes a usable candidate | ~3 good probes ≈ ~30s |
| Switchback to a recovered primary (ordinary) | healthy (~30s) + cooldown (60s) ≈ 60–90s |
| Switchback to the primary on identity reclaim | immediate (~1 probe interval) |

## Scope and safety

These are **per-node read-routing** rules feeding Heimdall's side handlers
(`ExtendVote`), never consensus state. The worst case for a node briefly routed
to a lagging but same-chain fallback is its own NO/UNSPECIFIED vote on a side
transaction — not a chain split. A wrong-network or unvalidated endpoint is never
a candidate and never active, except the narrow provisional-anchor window
described above, which the primary's reclaim ends.

## Metrics

Labeled by transport (`http` / `grpc`), see
[`metrics/borfailover.go`](../metrics/borfailover.go):

- `*_bor_failover_switches_total` — in-call cascade switches
- `*_bor_failover_proactive_switches_total` — background-prober switches (incl. revert-to-primary)
- `*_bor_failover_active_index` — index of the currently active endpoint (0 = primary)
- `*_bor_failover_healthy_endpoints` — number of endpoints currently healthy
