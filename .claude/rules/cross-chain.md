---
paths:
  - "bridge/**/*.go"
  - "helper/**/*.go"
  - "contracts/**/*.go"
  - "x/stake/**/*.go"
  - "x/topup/**/*.go"
  - "x/clerk/**/*.go"
  - "x/chainmanager/**/*.go"
---

# Cross-Chain & Bridge Security Review

This code handles L1<->L2 communication. Bugs here can lead to unauthorized validator joins, stolen funds, forged checkpoints, or state sync corruption.

## External Attack Vectors

- **L1 transaction crafter** (anyone with ETH): deploys a contract that emits events with matching topic hashes but from a wrong address, or crafts transactions with multiple events where log index extraction grabs the wrong one. If Heimdall doesn't verify `log.Address` and log index correctly, fake validator joins, stake updates, or state syncs are accepted.
- **L1 reorg exploiter**: submits a staking event, waits for Heimdall to process it on a non-finalized block, then reorgs L1. If the bridge uses `latest` instead of `finalized`, the event is processed but never happened.
- **Replay attacker**: replays a previously valid staking event (join, exit, signer change) if nonce checks are missing or flawed. Cost: one L1 transaction.
- **Malicious validator (as tx submitter)**: crafts Heimdall messages with manipulated fields (wrong nonce, fake pubkey, altered amount) that pass `ValidateBasic()` but exploit weak side-handler validation.

## L1 Receipt Validation

- ALWAYS verify transaction receipts against finalized L1 blocks (`rpc.FinalizedBlockNumber`), never pending/latest
- Validate the receipt's contract address matches the expected contract from ChainManager params
- Check `receipt.Status == 1` (success) before trusting event logs -- reverted txs still emit events in some edge cases
- Verify log indices -- a single transaction can emit multiple events; extracting the wrong log index is a critical vulnerability
- Ensure tx confirmation count meets the required threshold before processing
- Validate `receipt.BlockNumber` is not zero (zero indicates the receipt was not mined)

## Event Log Verification

- Decode event logs using the correct ABI -- mismatched ABIs silently produce wrong values with no error
- Verify `log.Topics[0]` matches the expected event signature hash before decoding
- Verify ALL fields from the event, not just a subset (e.g., for ValidatorJoin: signer, pubkey, activation epoch, amount, nonce, validator ID)
- Cross-check decoded values against expected ranges:
  - Amounts: must be > 0 and within sane bounds (not exceeding total supply)
  - Epochs: activation epoch >= current epoch for joins; deactivation epoch > current epoch for exits
  - Addresses: must not be zero address (`0x0000...0000`)
  - Nonces: must be exactly `validator.Nonce + 1`
- Never trust event data alone for state transitions -- cross-validate with contract view calls where possible

## Staking Security

- Nonce must equal `validator.Nonce + 1` for all staking operations (join, update, signer change, exit) -- prevents replay and reordering attacks
- Verify secp256k1 public key format: 33-byte compressed, first byte must be 0x02 or 0x03
- Signer address must match `crypto.PubkeyToAddress(pubkey)` derived from the event log -- never trust a signer address provided separately from its public key
- ValidatorJoin: verify the validator ID doesn't already exist in the active set
- ValidatorExit: deactivation epoch must be strictly > current epoch
- SignerUpdate: verify both old and new signer, ensure the old signer matches current on-chain state, fee token transfer must occur
- StakeUpdate: verify updated amount doesn't drop below minimum stake
- Slashing: validate slash amount against the minimum stake requirement, verify the slashing event comes from the SlashManager contract specifically

## Checkpoint Security

- Checkpoint continuity: new checkpoint start block must equal previous checkpoint end block + 1 -- any gap or overlap corrupts the checkpoint chain
- Root hash must match the value computed from Bor chain's `GetRootHash` for the exact same block range
- Account root hash must match the value from Bor's state at the checkpoint end block
- Proposer must be the expected proposer from the validator set rotation (round-robin based on voting power)
- Checkpoint ack: cross-validate header block number, proposer, start/end, root hash against L1 RootChain contract
- Checkpoint no-ack: verify the timeout period has actually elapsed since the last checkpoint

## State Sync (Clerk Module)

- State sync events must be processed in order by event nonce/ID -- out-of-order processing corrupts L2 state
- Verify the StateSender contract address matches ChainManager params
- Validate state sync data size bounds -- unbounded data causes OOM in consensus
- Event record ID must be monotonically increasing -- reject duplicates and gaps

## TopUp Fee Validation

- Verify the fee amount is > 0 and the fee token address matches the expected POL/MATIC token from ChainManager
- Validate the recipient validator exists and is active
- TopUp events from L1 must reference a valid, confirmed receipt

## Bridge Event Processing

- Listeners must filter events by exact contract address -- accepting events from wrong contracts is critical
- Processors must validate event data before creating Heimdall transactions
- RabbitMQ message processing must be idempotent (safe to replay on failure)
- Handle chain reorgs: using `finalized` block subscription prevents reorg issues, but verify this is enforced throughout -- any path using `latest` is vulnerable
- Rate limit bridge transaction creation to prevent spam during high L1 activity

## Red Flags -- Reject Immediately

- Removing or weakening finality requirements for L1 receipt validation
- Trusting event data without receipt status check
- Skipping nonce validation on any staking operation
- Processing events from unverified contract addresses
- Using `latest` instead of `finalized` block for L1 queries in consensus-critical paths
- Any change that makes bridge processing non-idempotent
- State sync events processed out of order
- Zero-address accepted as valid signer or validator
