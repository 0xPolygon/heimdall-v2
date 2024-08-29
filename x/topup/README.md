# Topup

Heimdall Topup is an amount which will be used to pay fees on Heimdall chain.

There are two ways to topup your account:

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

```go
message MsgTopupTx {
string proposer = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
string user = 2 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
string fee = 3 [
(gogoproto.customtype) = "cosmossdk.io/math.Int",
(gogoproto.nullable) = false
];
heimdallv2.types.TxHash tx_hash = 4 [ (gogoproto.nullable) = false ];
uint64 log_index = 5;
uint64 block_number = 6;
}
```

### MsgWithdrawFeeTx

`MsgWithdrawFeeTx` is responsible for withdrawing balance from Heimdall to Ethereum chain. A Validator can
withdraw any amount from Heimdall.

Handler processes the withdraw by deducting the balance from the given validator and prepares the state to send the next
checkpoint. The next possible checkpoint will contain the withdraw related state for the specific validator.

Handler gets validator information based on `ValidatorAddress` and processes the withdraw.

```go
message MsgWithdrawFeeTx {
string proposer = 1;
string amount = 3 [
(gogoproto.customtype) = "cosmossdk.io/math.Int",
(gogoproto.nullable) = false
];
}
```

## CLI Commands

[//]: # (TODO HV2: fill this section once the cli commands are tested)

### Topup fee

```bash
```

### Withdraw fee

```bash
```

To check reflected topup on account run following command

```bash
heimdalld query auth account <validator-address> --trust-node
```

## REST APIs

[//]: # (TODO HV2: fill this section once the REST APIs are tested)

### Topup fee

```bash
curl -X POST ...
```

### Withdraw fee

```bash
curl -X POST ...
```
