# Native peer reputation observation

This draft adds opt-in local peer scoring to Heimdall. It does not throttle,
reserve slots, disconnect, jail, or alter consensus. Existing CometBFT protections
continue unchanged. There is no sidecar, overlay, listener, IPC, extra handshake,
proof of work, additional signature verification, or Bor dependency.

## Enable and inspect

Run `heimdalld start --peer-reputation` with in-process CometBFT. The flag defaults
to false and is not a new config.toml option. It is rejected with `--with-comet=false`
or `--grpc-only`. Enable the existing CometBFT Prometheus endpoint using
`[instrumentation] prometheus = true` to scrape the metrics. No new endpoint is
created. Disabling the flag restores the default nil observer path on restart.
The observer is installed after configuration parsing and before node creation.

The CometBFT dependency is pinned to the [companion fork commit](https://github.com/0xPolygon/cometbft/pull/46) implementing
`p2p/observation.Observer`. Merge/release that companion before promoting this
Heimdall draft. No Cosmos SDK changes are required. That fork base also carries
its existing persistent-peer redial fix since v0.3.8-polygon.

## Module placement and data flow

| Module | Responsibility |
| --- | --- |
| `cmd/heimdalld/cmd/peer_reputation.go` | Startup flag, metrics registration, injection into the parsed CometBFT P2P config |
| `peerpolicy/evidence.go` | Classify existing native message types and select experimental observation profiles |
| `peerpolicy/tracker.go` | Bounded per-node windows, repeat history, score and hypothetical action |
| `peerpolicy/metrics.go` | Fixed-cardinality Prometheus telemetry |
| CometBFT `p2p/observation` | Small borrowed-event interface, without a policy or admission result |
| CometBFT `p2p/peer.go` | Existing decode results, received envelope size, and successful local queueing |
| CometBFT blocksync/statesync reactors | Report existing basic protocol validation failures before their existing disconnect path |

```text
Authenticated native peer (CometBFT node ID)
               |
    existing framing and protobuf decode
               |                 decode failure ------------------+
               v                                                  |
    Received event: type + encoded bytes                          |
               |                                                  |
    unchanged native reactor validation ---- invalid message -----+
               |                                                  |
    unchanged block/chunk loading and send queue                   |
               |                                                  v
    Queued event only on successful enqueue ------------> Heimdall peerpolicy
                                                       bounded windows per ID
                                                               |
                                                       risk + reason counters
                                                               |
                                                       existing metrics endpoint
```

Callbacks run synchronously with bounded local work. They do not retain decoded
messages or call back into the networking stack. Message bytes are measured from
already encoded buffers; there is no second marshal, payload hashing or database
read for scoring. Consensus messages are not scored for traffic volume.

## Evidence and neutrality

- Protobuf decode/unwrap failures and blocksync/statesync basic message or block
  conversion failures contribute correctness evidence. They reuse native checks.
- Block/status requests, snapshot discovery requests and chunk requests count by
  message and encoded bytes before serving. Varied heights still count.
- Inbound mempool `Txs` count every transaction and the whole encoded envelope.
  Changing transaction contents cannot evade the volume profile. Already cached,
  previously committed and application-rejected transactions are not labelled
  cryptographically invalid. No ABCI CheckTx result is used as misconduct proof.
- Inbound block/snapshot/chunk responses count as resource volume. A slow or syncing
  honest node may cross the experimental profiles; observation measures this and
  is not authorization to punish it.
- Outbound block responses and non-missing chunk responses count only after the
  existing send operation succeeds. This means *accepted by the local queue*, not
  bytes flushed to the network or acknowledged by the remote peer. Failed queueing,
  missing blocks/chunks and local load errors produce no serving credit or repeat
  event. There is no request-to-response correlation proof.
- The first response and one retry per object are tolerated. Further queued copies
  count repeated bytes. Block objects are identified by finalized height; chunks by
  height, snapshot format and index. These are native object keys, not new hashes.
- The local `MaxSnapshotChunks` policy rejection remains neutral for correctness.
  Its existing CometBFT rejection still applies. Its incoming volume can still count.
- Silence and requesting data without contributing gossip are neutral. Persistent,
  private and ordinary peers are observed alike; no class loses an existing allowance.
  CometBFT mempool gossip carries transactions, so Bor hash-announcement rules are
  not transplanted into this protocol.

This first draft does not attribute deferred commit verification failures, consensus
vote/proposal validation, ABCI snapshot rejection or network timeouts to a peer score.
A downstream failure may implicate several suppliers. Those hooks need provenance
before enforcing correctness penalties. Protocol validation itself is unchanged.

## Score calculation

A peer record contains six ten-second monotonic buckets. Each bucket retains a bit
for each observed reason. Correctness reasons have weight **60**. Traffic-volume
and repeated-serving reasons have weight **20**.

`risk = min(100, sum(strongest_reason_weight_per_live_bucket))`

For example, repeated serving above the profile in one bucket gives 20; three such
buckets give 60. A malformed packet and a reactor report in the same bucket give
60 together. Two correctness buckets give 100. Taking the strongest reason per
bucket conservatively avoids double counting across layers while undercounting
independent incidents. This is an experimental local model, not the GossipSub
formula or a calibrated probability of malicious behavior.

Evidence ages out in approximately 50–60 seconds depending on bucket alignment.
Risk below 40 maps to hypothetical `none`; 40–99 to `throttle`; 100 to `jail`.
**These names are observations only. There is no new timer, jail map or action.**
Counters expose each reason even when another reason determines the bucket score.

## Experimental profiles

Each row applies per peer per ten-second bucket. Crossing either column records
one resource reason in that bucket, irrespective of how many later events cross it.

| Traffic family | Items | Encoded bytes |
| --- | ---: | ---: |
| Block/status requests | 640 | 1 MiB |
| Snapshot discovery requests | 64 | 1 MiB |
| Chunk requests | 640 | 1 MiB |
| Inbound mempool transactions | 32,768 transactions | 64 MiB |
| Inbound block/snapshot/chunk responses | 10,240 messages | 160 MiB |
| Queued block responses | 10,240 messages | 160 MiB |
| Queued non-missing chunk responses | 10,240 messages | 160 MiB |

Repeated queued bytes above **32 MiB** in a bucket also produce resource evidence
after the initial copy and retry. Profiles are hardcoded and intentionally do not
claim production-safe enforcement thresholds. Volume uses saturating arithmetic.

## Bounds and identity

At most 1,024 peer records are retained in an LRU. Each has six fixed buckets and
128 object-history entries in a bounded FIFO ring. Object history expires after
six ticks and can be evicted sooner. Rotating object keys can evade repetition
history but not total volume. Rotating node IDs can evade per-ID history; eventual
node-wide admission limits remain essential.

The key is the node ID authenticated by native CometBFT transport. A same-ID
reconnect reuses its record until expiry or eviction. State is local and in memory;
restart clears it. It is not shared across processes or connected Bors. No libp2p
identity, separate ledger, or identity-binding handshake is introduced.

## Metrics and rollout gate

Import `docs/peer-reputation-dashboard.json` into Grafana and select the existing
Prometheus datasource. The template is not automatically installed.

- `heimdall_peer_reputation_reason_events_total{reason}`: individual classified events.
- `heimdall_peer_reputation_reason_windows_total{reason}`: distinct peer windows per reason.
- `heimdall_peer_reputation_bytes_total{family}`: classified encoded envelope bytes.
- `heimdall_peer_reputation_would_actions_total{action}`: upward hypothetical transitions.
- `heimdall_peer_reputation_risk`: event-sampled histogram, not a peer population distribution.
- `heimdall_peer_reputation_evictions_total{mode="observe"}`: bounded-state evictions.

Labels come from fixed enumerations. No peer ID, IP, height, hash or payload is a
metric label. `Tracker.Snapshot` supports local callers/tests; this draft does not
expose it through a new RPC endpoint. Per-peer UI/slot allocation is not implemented.

Start on selected sentries and compare normal sync/reconnect periods with resource
pressure. Measure CPU, lock contention, serving latency, sync throughput, consensus
round/commit latency and false-positive reason windows. Tune profiles before a
separate enforcement change. A unit hot-path benchmark is not a network benchmark.

## Existing mechanism mapping

| Existing CometBFT mechanism | This draft | Future enforcement requirement |
| --- | --- | --- |
| `Switch.StopPeerForError` and reactor disconnects | Kept; no score-driven second disconnect | Attribute causes without counting an action as a second incident |
| Blocksync `BlockPool.bannedPeers` and existing 60-second expiry | Kept; not copied into a new jail map | Separate misconduct from scheduling outcomes before consolidating expiry |
| Blocksync timeout/rate handling and requester reassignment | Kept and neutral to this correctness score | Preserve sync liveness and distinguish transient failures |
| Statesync snapshot/format/peer blacklists and ABCI rejection | Kept; not imported as proof of invalid data | Retain snapshot-scoped exclusions separately from peer misconduct |
| PEX address selection/backoff and persistent-peer reconnect | Kept | Do not turn scheduling backoff into a jail |
| Existing send queues, packet limits and network rate limits | Kept | Add admission budgets before expensive loading and serialization |

Observation does not replace serving controls. Follow-up enforcement must provide
node-wide byte/work/concurrency limits, per-peer admission, and explicitly reserved
operator-designated validator headroom within node-wide caps. Persistent/private
status alone must not imply unlimited service or consume the reserved pool. It must
also define one authoritative misconduct policy and expiry before enabling actions.
