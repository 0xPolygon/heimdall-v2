# Topup module

## Table of Contents

* [Overview](#overview)
* [Interact with the Node](#interact-with-the-node)
  * [Tx Commands](#tx-commands)
  * [CLI Query Commands](#cli-query-commands)
  * [GRPC Endpoints](#grpc-endpoints)
  * [REST Endpoints](#rest-endpoints)

## Overview

Heimdall Topup is an amount which will be used to pay fees on Heimdall chain.

There are two ways to top up your account:

1. When new validator joins, they can mention a `topup` amount as top-up in addition to the staked amount, which will be
   moved as balance on Heimdall chain to pay fees on Heimdall.
2. A user can directly call the top-up function on the staking smart contract on Ethereum to increase top-up balance on
   Heimdall.

## Messages

### MsgTopupTx

`MsgTopupTx` is responsible for minting balance to an address on Heimdall based on Ethereum chain's `TopUpEvent` on
staking manager contract.

Handler for this transaction processes top-up and increases the balance only once for any given `msg.TxHash`
and `msg.LogIndex`. It throws an error if trying to process the top-up more than once.

Here is the structure for the top-up transaction message:

```protobuf
message MsgTopupTx {
   string proposer = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
   string user = 2 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
   string fee = 3 [
      (gogoproto.customtype) = "cosmossdk.io/math.Int",
      (gogoproto.nullable) = false
   ];
   bytes tx_hash = 4;
   uint64 log_index = 5;
   uint64 block_number = 6;
}
```

### MsgWithdrawFeeTx

`MsgWithdrawFeeTx` is responsible for withdrawing balance from Heimdall to Ethereum chain. A Validator can
withdraw any amount from Heimdall.

Handler processes the withdrawal by deducting the balance from the given validator and prepares the state to send the next
checkpoint. The next possible checkpoint will contain the withdrawal related state for the specific validator.

Handler gets validator information based on `ValidatorAddress` and processes the withdrawal.

```protobuf
message MsgWithdrawFeeTx {
   string proposer = 1;
   string amount = 3 [
      (gogoproto.customtype) = "cosmossdk.io/math.Int",
      (gogoproto.nullable) = false
   ];
}
```

## Interact with the Node

### Tx Commands

#### Topup fee

```bash
./build/heimdalld tx topup handle-topup-tx [proposer] [user] [fee] [tx_hash] [log_index] [block_number]
```

#### Withdraw fee

```bash
./build/heimdalld tx topup withdraw-fee [proposer] [amount]
```

### CLI Query Commands

One can run the following query commands from the topup module:

* `topup-sequence` - Query the sequence of a topup tx
* `is-old-tx` - Check if a tx is old (already submitted)
* `dividend-account` - Query a dividend account by its address
* `dividend-account-root` - Query dividend account root hash
* `account-proof` - Query account proof
* `verify-account-proof` - Verify account proof

```bash
./build/heimdalld query topup topup-sequence [tx_hash] [log_index]
```

```bash
./build/heimdalld query topup is-old-tx [tx_hash] [log_index]
```

```bash
./build/heimdalld query topup dividend-account [address]
```

```bash
./build/heimdalld query topup dividend-account-root
```

```bash
./build/heimdalld query topup account-proof [address]
```

```bash
./build/heimdalld query topup verify-account-proof [address] [proof]
```

### GRPC Endpoints

The endpoints and the params are defined in the [topup/query.proto](/proto/heimdallv2/topup/query.proto) file. Please refer them for more information about the optional params.

```bash
grpcurl -plaintext -d '{"tx_hash": <>, "log_index": <>}' localhost:9090 heimdallv2.topup.Query/GetTopupTxSequence
```

```bash
grpcurl -plaintext -d '{"tx_hash": <>, "log_index": <>}' localhost:9090 heimdallv2.topup.Query/IsTopupTxOld
```

```bash
grpcurl -plaintext -d '{"address": <>}' localhost:9090 heimdallv2.topup.Query/GetDividendAccountByAddress
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.topup.Query/GetDividendAccountRootHash
```

```bash
grpcurl -plaintext -d '{"address": <>}' localhost:9090 heimdallv2.topup.Query/GetAccountProofByAddress
```

```bash
grpcurl -plaintext -d '{"address": <>, "proof": <>}' localhost:9090 heimdallv2.topup.Query/VerifyAccountProofByAddress
```

### REST APIs

The endpoints and the params are defined in the [topup/query.proto](/proto/heimdallv2/topup/query.proto) file. Please refer them for more information about the optional params.

```bash
curl curl localhost:1317/topup/sequence?tx_hash=<tx-hash>&log_index=<log-index>
```

```bash
curl curl localhost:1317/topup/isoldtx?tx_hash=<tx-hash>&log_index=<log-index>
```

```bash
curl curl localhost:1317/topup/dividend-account/{address}
```

```bash
curl curl localhost:1317/topup/dividend-account-root
```

```bash
curl curl localhost:1317/topup/account-proof/{address}
```

```bash
curl curl localhost:1317/topup/account-proof/{address}/verify
```
