---
paths:
  - "helper/call.go"
  - "helper/tx.go"
  - "helper/receipt.go"
  - "helper/unpack.go"
  - "helper/mocks/**/*.go"
  - "contracts/**/*.go"
  - "x/bor/grpc/**/*.go"
  - "x/*/keeper/keeper.go"
  - "x/*/keeper/grpc_query.go"
  - "bridge/processor/**/*.go"
  - "cmd/heimdalld/cmd/stake.go"
---

# Smart Contract Interaction Security Review

All Ethereum/Bor contract interactions flow through `helper.IContractCaller`. This is the trust boundary between Heimdall consensus and external chains. Bugs here can forge validator joins, steal funds, or corrupt checkpoints.

## External Attack Vectors

- **Malicious/compromised RPC provider**: returns fabricated block headers, fake receipts, wrong chain IDs, manipulated gas estimates, or zero-value responses. If `IContractCaller` doesn't validate responses, all validators using that provider vote based on false data. A coordinated RPC attack on a popular provider (Infura, Alchemy) can affect a majority of validators simultaneously.
- **L1 contract upgrader** (Polygon governance): if L1 contracts (RootChain, StakeManager) are upgraded via proxy, ABI bindings become stale. Mismatched ABIs decode silently with wrong values -- no error, just garbage data flowing into consensus.
- **Front-running attacker** (L1 MEV): observes Heimdall's `SendCheckpoint` or `StakeFor` transaction in the L1 mempool and front-runs with a same-nonce higher-gas replacement, or sandwich-attacks to manipulate gas pricing.
- **Bor chain attacker**: if Bor's gRPC endpoint is compromised or a malicious Bor node is connected, it can feed false root hashes, block headers, or milestone data to Heimdall's side handlers.

## IContractCaller Interface (`helper/call.go`)

- This is the single most security-critical non-consensus file in the codebase
- Every method must validate its return values -- nil checks on all pointer returns, length checks on byte slices
- `GetConfirmedTxReceipt()` must enforce finality (finalized block, not latest) -- this is the root of all L1 trust
- `GetHeaderInfo()`, `GetLastChildBlock()` read from RootChain -- verify the contract instance was created with the address from ChainManager params, not a hardcoded address
- `GetValidatorInfo()` reads from StakingInfo -- cross-check returned data against expected ranges (non-zero amount, valid epoch)
- `GetRootHash()` and `GetVoteOnHash()` call Bor RPC -- validate response length (root hash = 32 bytes) and non-nil
- `CurrentAccountStateRoot()` reads merkle root -- used for checkpoint validation, single-bit difference means invalid checkpoint
- ABI instances are loaded at init time from compiled bindings in `contracts/` -- integrity depends on the bindings matching deployed contracts

## Determinism in Contract Calls: Side Handlers vs Post Handlers

This is a critical architectural distinction:

- **Side handlers** (run in `ExtendVote`): CAN make RPC calls to L1/Bor. Each validator calls independently. RPC failures, timeouts, or different results across validators are expected -- they produce different vote extension opinions (YES/NO/UNSPECIFIED). This is by design.
- **Post handlers** (run in `PreBlocker`): MUST NOT make RPC calls. They execute only for side-txs that received 2/3+ approval. They must be fully deterministic using only the data already in the Cosmos SDK state store. Any external call here breaks consensus.

If a change adds an `IContractCaller` call, verify it's in a side handler path, never a post handler path.

## Transaction Construction (`helper/tx.go`)

- `GenerateAuthObj()` creates EIP-1559 `TransactOpts` -- verify gas tip cap and fee cap have upper bounds to prevent overpaying. Don't blindly trust `ethclient.SuggestGasPrice()` from remote RPCs -- a malicious RPC can suggest extreme gas prices.
- `SendCheckpoint()` submits to RootChain -- irreversible L1 write, validate ALL inputs before the call. A bad checkpoint on L1 requires governance intervention to fix.
- `StakeFor()` and `ApproveTokens()` move real funds -- double-validate amounts and addresses, verify the spender is the expected StakeManager contract
- Nonce management: query pending nonce before sending, use mutex if concurrent tx submission is possible. Same-nonce txs with higher gas replace previous ones (front-running vector).
- Set explicit gas limits -- `EstimateGas()` from a malicious RPC can return 0 or max uint64

## Contract Upgrade Risk

- L1 contracts (RootChain, StakeManager, StakingInfo) may be upgradeable proxies. If the implementation changes, ABI bindings become stale.
- ChainManager stores contract addresses and can update them via governance -- this is the correct upgrade path
- If ChainManager params change, verify the new addresses point to contracts with compatible ABIs
- After any L1 contract upgrade, regenerate Go bindings with `abigen` and verify they compile

## ABI Encoding/Decoding (`helper/unpack.go`, `contracts/`)

- ABI mismatch silently produces wrong values with no error -- this is the most insidious class of bug
- `UnpackLog()` decodes event logs -- always verify `log.Topics[0]` (event signature) matches the expected ABI event before decoding
- `UnpackSigAndVotes()` decodes checkpoint signatures -- validate array lengths match expected signer count, validate each signature format (65 bytes: r[32] + s[32] + v[1])
- Generated bindings in `contracts/` must match deployed contract ABIs exactly -- track deployed contract versions
- When decoding events from receipts, always verify `log.Address` matches the expected contract

## Bor gRPC Client (`x/bor/grpc/`)

- Direct gRPC client to Bor bypasses the `IContractCaller` abstraction -- apply identical security standards
- `HeaderByNumber`, `BlockByNumber`: validate response is non-nil, block number matches request, chain ID matches expected Bor chain
- `GetRootHash`: response must be exactly 32 bytes, non-zero
- `TransactionReceipt`: verify receipt status and that the block is finalized on Bor
- All gRPC calls need `context.WithTimeout` -- default CometBFT ABCI timeout is 10s, so gRPC timeout must be shorter
- gRPC connection failures must return explicit errors, never nil responses that pass downstream validation

## Module Keepers Using `IContractCaller`

- Every keeper that holds `IContractCaller` is a potential attack surface
- Side handlers call `contractCaller` during `ExtendVote` -- RPC errors here produce a NO vote, which is safe
- gRPC query handlers (`grpc_query.go`) expose contract data to external clients -- sanitize responses, don't leak internal state
- Mock implementations (`helper/mocks/`) must cover error cases (RPC timeout, nil response, wrong chain ID) -- tests that only mock happy paths miss critical vulnerabilities

## Bridge Processors (`bridge/processor/`)

- Processors create their own `ContractCaller` instances -- same security rules apply as `helper/call.go`
- `checkpoint.go` calls `SendCheckpoint()` -- the most critical write operation in the system. Must verify checkpoint isn't already submitted on L1 before sending.
- All processors must validate event data before constructing Heimdall transactions
- Check tx status against Heimdall before re-submitting to prevent duplicate transactions

## Red Flags

- Any change to `helper/call.go` or `helper/tx.go` without corresponding test updates
- Removing or weakening receipt finality checks
- Hardcoding contract addresses instead of reading from ChainManager params
- ABI changes without regenerating Go bindings
- New contract calls without `context.WithTimeout`
- Transaction construction with unbounded gas or unchecked amounts
- Bor gRPC calls without nil/error checking on responses
- Adding `IContractCaller` usage in a post-handler or `PreBlocker` path
