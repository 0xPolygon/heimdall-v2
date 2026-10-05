# Native peer reputation and connection enforcement

This draft enables local peer scoring and connection enforcement by default in
Heimdall. Risk 100 disconnects an active peer and rejects connections from the same
authenticated node ID while its live score remains 100. It does not throttle
requests, reserve slots, or alter consensus validation. Existing CometBFT protections
continue to apply. There is no sidecar, overlay, listener, IPC, extra handshake,
proof of work, additional signature verification, or Bor dependency.

## Enable and inspect

Normal `heimdalld start` enables scoring and connection enforcement on the native
peer network. Use `--peer-reputation-enforce=false` for observation only or
`--peer-reputation=false` to disable both. The flag is not a config.toml option.
Query-only (`--grpc-only`) and external-CometBFT (`--with-comet=false`) modes skip
observation by default because there is no native peer network in this process;
explicitly setting either reputation flag while scoring is enabled in those modes
returns an error.
Enable the existing CometBFT Prometheus endpoint using `[instrumentation]
prometheus = true` to scrape the metrics. No new endpoint is created. The observer
is installed after configuration parsing and before node creation. Scoring starts
without requiring the metrics endpoint to be enabled.

The CometBFT dependency is pinned to the [companion fork commit](https://github.com/0xPolygon/cometbft/pull/46) implementing
`p2p/observation.Observer` and `ConnectionPolicy`. Merge/release that companion before promoting this
Heimdall draft. No Cosmos SDK changes are required. That fork base also carries
its existing persistent-peer redial fix since v0.3.8-polygon.

## Module placement and data flow

| Module | Responsibility |
| --- | --- |
| `cmd/heimdalld/cmd/peer_reputation.go` | Startup flag, metrics registration, injection into the parsed CometBFT P2P config |
| `peerpolicy/evidence.go` | Classify existing native message types and select experimental observation profiles |
| `peerpolicy/tracker.go` | Bounded per-node windows, repeat history, score, connection eligibility and hypothetical throttle band |
| `peerpolicy/metrics.go` | Fixed-cardinality Prometheus telemetry |
| CometBFT `p2p/observation` | Bounded observation and connection-policy interfaces; application-owned state |
| CometBFT `p2p/peer.go` | Observe decode/receive/queue outcomes; check policy before sends and after events; close denied connections through native error handling |
| CometBFT `p2p/switch.go` | Check policy after authentication and before peer/reactor admission in both directions |
| CometBFT blocksync/statesync reactors | Report existing basic protocol validation failures before their existing disconnect path |

```text
Authenticated native peer (CometBFT node ID)
               |
    Switch.filterPeer -> AllowPeer(ID) -------- denied -> reject connection
               |                         ^
               v                         |
    existing framing / protobuf decode   |       Heimdall peerpolicy
               |                         |      bounded score ledger
    Observe(Received or InvalidEncoding) ---------> risk + reason counters
               |                         ^                  |
    AllowPeer(ID) -----------------------+                  v
               |                                     existing metrics
    native reactor validation
               | invalid -> ObserveInvalid -> native disconnect
               v
    existing block/chunk loading
               |
    AllowPeer(ID) -> existing send queue -> Observe(Queued) -> AllowPeer(ID)

    Activity denial -> close connection -> native removal / reconnect handling
    Reconnect       -> authenticate ID -> consult the same unexpired evidence
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
  honest node may cross the experimental profiles. In enforcement mode repeated
  excess can disconnect it; calibrate in observation mode before production rollout.
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
  private, unconditional and ordinary peers use the same score and connection gate.
  Being persistent does not bypass policy; native redial/backoff remains in place.
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
Only the risk-100 band has an enforced action: disconnect/reject the connection.
The `throttle` band remains hypothetical; it does not reduce request allowances.
The legacy `wouldAction: jail` label describes a score band, not an independent jail.
`Snapshot.mode` is `enforce` or `observe`; `connectionAllowed` reports eligibility.
There is no additional timer or jail map. Admission opens as soon as live evidence
expires enough to put the score below 100. Checks do not refresh that evidence;
reconnection still depends on native dialing/backoff and other existing filters.

A received envelope that reaches 100 is not dispatched to its reactor. A queued
response has already consumed serving work and may already be sent when it pushes
the score to 100; disconnecting cannot recover that cost. Checks use short local
critical sections and fixed-size windows. They add no validation round trip, but
are not claimed to have zero latency. Existing packet/queue bounds still apply.
Score admission occurs after authentication, so it does not protect handshake work.
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
claim production-safe thresholds. They require sentry calibration before release. Volume uses saturating arithmetic.

## Bounds and identity

At most 1,024 peer records are retained in an LRU. Each has six fixed buckets and
128 object-history entries in a bounded FIFO ring. Object history expires after
six ticks and can be evicted sooner. Rotating object keys can evade repetition
history but not total volume. Rotating node IDs can evade per-ID history; eventual
node-wide admission limits remain essential.

The key is the node ID authenticated by native CometBFT transport. A same-ID
reconnect reuses its record until expiry or eviction. State is local and in memory;
restart clears it. Eviction or restart can therefore reopen a connection before
the old evidence would have expired. It is not shared across processes or connected Bors. No libp2p
identity, separate ledger, or identity-binding handshake is introduced.

## Metrics and rollout gate

- `heimdall_peer_reputation_reason_events_total{reason}`: individual classified events.
- `heimdall_peer_reputation_reason_windows_total{reason}`: distinct peer windows per reason.
- `heimdall_peer_reputation_bytes_total{family}`: classified encoded envelope bytes.
- `heimdall_peer_reputation_would_actions_total{action}`: upward score-band transitions, not counts of actual disconnects.
- `heimdall_peer_reputation_risk`: event-sampled histogram, not a peer population distribution.
- `heimdall_peer_reputation_evictions_total{mode="observe|enforce"}`: bounded-state evictions.

Labels come from fixed enumerations. No peer ID, IP, height, hash or payload is a
metric label. `Tracker.Snapshot` supports local callers/tests; this draft does not
expose it through a new RPC endpoint. Per-peer UI/slot allocation is not implemented.

Start on selected sentries with `--peer-reputation-enforce=false` and compare normal
sync/reconnect periods with resource pressure. Measure CPU, lock contention, serving latency, sync throughput, consensus
round/commit latency and false-positive reason windows. Tune profiles before releasing default enforcement. A unit hot-path benchmark is
not a network benchmark.

## Existing mechanism mapping

| Existing CometBFT mechanism | This draft | Future enforcement requirement |
| --- | --- | --- |
| `Switch.StopPeerForError` and reactor disconnects | Kept; score-driven closes enter the existing removal path | Attribute causes without counting an action as a second incident |
| Blocksync `BlockPool.bannedPeers` and existing 60-second expiry | Kept; not copied into a new jail map | Separate misconduct from scheduling outcomes before consolidating expiry |
| Blocksync timeout/rate handling and requester reassignment | Kept and neutral to this correctness score | Preserve sync liveness and distinguish transient failures |
| Statesync snapshot/format/peer blacklists and ABCI rejection | Kept; not imported as proof of invalid data | Retain snapshot-scoped exclusions separately from peer misconduct |
| PEX address selection/backoff and persistent-peer reconnect | Kept | Do not turn scheduling backoff into a jail |
| Existing send queues, packet limits and network rate limits | Kept | Add admission budgets before expensive loading and serialization |

Connection enforcement does not replace serving controls. Follow-up work must provide
node-wide byte/work/concurrency limits, per-peer admission, and explicitly reserved
operator-designated validator headroom within node-wide caps. Persistent/private
status alone must not imply unlimited service or consume the reserved pool.
Existing blocksync bans and snapshot exclusions keep their own protocol-specific
semantics. This change neither imports them into the score nor creates another
misconduct timer; policy rejection itself never adds evidence.
