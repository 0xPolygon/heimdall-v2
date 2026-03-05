---
paths:
  - "helper/config.go"
  - "helper/query.go"
  - "bridge/broadcaster/**/*.go"
  - "bridge/listener/**/*.go"
  - "bridge/service/**/*.go"
  - "bridge/queue/**/*.go"
  - "cmd/**/*.go"
  - "packaging/templates/**"
---

# P2P, Networking & RPC Security Review

Heimdall delegates P2P consensus to CometBFT, but configures seeds, peers, RPC endpoints, and runs bridge listeners/broadcasters that connect to L1, Bor, and RabbitMQ. Compromised networking allows eclipse attacks, transaction censorship, or forged chain data.

## External Attack Vectors

- **Eclipse attacker** (any P2P participant): monopolizes a validator's peer connections by flooding with sybil peers. The isolated validator sees only attacker-controlled blocks/txs, can be tricked into signing conflicting proposals or missing votes. Low peer thresholds and enabled PEX make this easier.
- **MITM on RPC** (network-level): if RPC connections to L1/Bor use plaintext HTTP, a network-level attacker can intercept and modify responses -- feeding fake finalized blocks, wrong root hashes, or fabricated receipts. Affects all bridge operations.
- **RabbitMQ message injector** (adjacent service/host): if AMQP uses default `guest:guest` credentials or no TLS, any process on the same network can inject fabricated bridge events (fake state syncs, forged staking events) or consume/drop legitimate ones.
- **Gossip flooder** (any peer): exploits aggressive gossip settings (e.g., 25ms `PeerGossipSleepDuration`) to amplify bandwidth consumption, degrading consensus performance for the target validator.
- **SubGraph endpoint attacker** (MITM or compromised endpoint): feeds fabricated or empty event data to self-heal logic, causing the bridge to skip events or process fake ones during recovery.

## CometBFT P2P Configuration

- Seed nodes and persistent peers must come from verified config for known networks (mainnet, amoy) -- never accept seeds from unvalidated runtime input
- `PeerGossipSleepDuration` and `PeerQueryMaj23SleepDuration` affect consensus timing -- lowering these increases bandwidth; raising them degrades liveness. Changes require network-wide coordination.
- `AddrBookStrict = false` disables address verification -- acceptable for testnets only, never mainnet
- Verify `MinPeerThreshold` / `WarnPeerThreshold` are set to sane values -- low peer counts make the node vulnerable to eclipse attacks (attacker controls all peers)
- P2P config changes in `packaging/templates/` propagate to all new deployments -- review with same rigor as code changes
- Peer Exchange (PEX) reactor: if enabled, peers can inject malicious peer addresses. For validator nodes, prefer `pex = false` with explicit persistent peers. Sentry nodes can use PEX.
- Validator nodes should not expose their P2P port publicly -- use sentry node architecture where sentry nodes shield validators from direct network access
- **Reference the repo's own config templates** for current defaults and param tuning: `packaging/templates/config/mainnet/config.toml` and `packaging/templates/config/amoy/config.toml`. CometBFT params set programmatically in `cmd/heimdalld/cmd/commands.go` (`initCometBFTConfig`) may differ from template defaults.

## RPC Endpoint Security

- All RPC URLs (CometBFT, Ethereum, Bor) must come from config, never hardcoded in source
- Use TLS (`https://` / `wss://`) for remote RPC endpoints. Local endpoints (`localhost`, `127.0.0.1`) may use plaintext but must bind to loopback only, not `0.0.0.0`.
- Every RPC call must use `context.WithTimeout` -- hanging connections block consensus or bridge processing. Timeout should be shorter than CometBFT's ABCI timeout (~10s).
- Handle RPC errors explicitly: never silently use zero values from failed calls -- this is the most common source of consensus divergence between validators
- Validate L1/Bor chain IDs on initial connection and periodically -- prevents connecting to wrong network (testnet vs mainnet)
- Consider RPC provider reliability: if using third-party providers (Infura, Alchemy), a provider outage affects all validators using it, which can stall consensus. Recommend multiple fallback providers.

## Bridge Listeners

- Listeners must subscribe to **finalized** blocks on L1 (`rpc.FinalizedBlockNumber`), never pending/latest -- this is the primary reorg protection
- Filter events by exact contract address from ChainManager params -- accepting events from wrong contracts enables arbitrary state injection
- Self-heal mechanisms (`rootchain_selfheal.go`) must validate recovered events with the same strictness as live events -- weaker validation in self-heal is a backdoor
- SubGraph queries for self-healing: validate the subgraph endpoint is the official Polygon subgraph, verify response schema, cross-validate results against L1 when possible
- Listener polling intervals affect event detection latency -- too slow misses events, too fast wastes resources and may trigger RPC rate limits

## Bridge Broadcaster

- `BroadcastToHeimdall()` signs Cosmos SDK txs using the keyring -- verify keyring backend is `file` or `os` in production, never `test`
- `BroadcastToBorChain()` sends raw Ethereum txs -- nonce management must be serialized (mutex or channel) to prevent nonce collisions in concurrent sends
- Set explicit gas limits on all broadcast transactions -- `EstimateGas()` from untrusted RPCs can return manipulated values
- Log tx hashes for audit trail, but never log private keys, keyring passwords, or raw signing bytes
- Implement retry with backoff for failed broadcasts, but cap retries to prevent infinite loops on permanently rejected txs

## RabbitMQ / Message Queue

- Queue connections must use authentication and TLS in production -- unauthenticated AMQP allows message injection (attacker can forge bridge events)
- Message processing must be idempotent -- RabbitMQ guarantees at-least-once delivery, so duplicate messages are expected
- Validate message content before processing -- treat queue messages as untrusted input, verify expected schema
- Monitor queue depth -- unbounded growth indicates stuck processing or DoS
- Set message TTL to prevent indefinite accumulation of stale events

## CLI Commands (`cmd/`)

- Most modules have **autocli enabled** -- auto-generated CLI commands must correctly reflect the manually implemented CLI commands. If a module adds custom CLI commands, ensure they don't conflict with or shadow autocli-generated ones.
- CLI commands that interact with contracts (`stake.go`: StakeFor, ApproveTokens) must validate all user inputs before calling the contract layer
- Never log or echo private keys, even in error messages or debug output
- `--home` flag controls key storage location -- validate path permissions (not world-readable, mode 0700 for keyring directory)
- `testnet` command generates node keys and persistent peer lists -- verify generated keys have proper entropy and peer IDs are derived correctly

## Red Flags

- Removing TLS requirements for remote RPC endpoints
- Accepting seeds/peers from unvalidated runtime input
- RPC calls without `context.WithTimeout`
- Bridge listeners using `latest` instead of `finalized` block subscription
- Queue connections without authentication in production configs
- Self-heal code with weaker validation than live event processing
- Validator P2P port exposed to public internet without sentry nodes
- `pex = true` on validator nodes in mainnet config
- Keyring backend set to `test` in production
