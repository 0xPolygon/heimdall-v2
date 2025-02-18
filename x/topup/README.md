[//]: # (TODO HV2: https://polygon.atlassian.net/browse/POS-2757)

# Topup module

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

## CLI Commands
[//]: # (TODO HV2: populate and check the commands below)

### Topup fee

```bash
```

### Withdraw fee

```bash
./build/heimdalld tx topup withdraw-fee <proposer> <amount≥
```

To check reflected topup on account run following command

```bash
./build/heimdalld query auth account <validator-address>
```

## REST APIs

### Is topup processed

```bash
curl -X GET "localhost:1317/clerk/isoldtx?tx-hash=<transaction-hash>&log-index=<log-index>"
```

### Get topup transaction sequence

```bash
curl -X GET "localhost:1317/topup/sequence?tx-hash=<transaction-hash>&log-index=<log-index>"
```

### Get dividend account by address

```bash
curl -X GET "localhost:1317/topup/dividend-account/<address>"
```

### Get dividend account root hash
    
```bash
curl -X GET "localhost:1317/topup/dividend-account-root"
```

### Get account proof by address

```bash
curl -X GET "localhost:1317/topup/account-proof/<address>"
```

### Verify account proof

```bash
curl -X GET "localhost:1317/topup/account-proof/<address>/verify"
```

### Topup fee

```bash
curl -X POST ...
```

### Withdraw fee

```bash
curl -X POST ...
```
