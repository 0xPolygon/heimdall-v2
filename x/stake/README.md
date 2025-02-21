# Stake module

## Table of Contents

* [Overview](#overview)
* [Interact with the Node](#interact-with-the-node)
  * [Tx Commands](#tx-commands)
  * [CLI Query Commands](#cli-query-commands)
  * [GRPC Endpoints](#grpc-endpoints)
  * [REST Endpoints](#rest-endpoints)

## Overview

This module manages validators related transactions and state for Heimdall.  
validators stake their tokens on the Ethereum chain and send the transactions on Heimdall using necessary parameters to acknowledge the Ethereum stake change.  
Once the majority of the validators agree on the change on the stake, this module saves the validator information on Heimdall state.  

![Stake Flow.png](stake_flow.png)

## Messages

### MsgValidatorJoin

`MsgValidatorJoin` defines a message for a node to join the network as validator.

Here is the structure for the transaction message:

```protobuf
message MsgValidatorJoin {
option (amino.name) = "heimdallv2/MsgValidatorJoin";

option (gogoproto.equal) = false;
option (gogoproto.goproto_getters) = true;

string from = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];
uint64 val_id = 2 [ (amino.dont_omitempty) = true ];
uint64 activation_epoch = 3 [ (amino.dont_omitempty) = true ];
string amount = 4 [
(gogoproto.nullable) = false,
(gogoproto.customtype) = "cosmossdk.io/math.Int",
(amino.dont_omitempty) = true
];
google.protobuf.Any signer_pub_key = 5
[ (cosmos_proto.accepts_interface) = "cosmos.crypto.PubKey" ];
bytes tx_hash = 6 [ (amino.dont_omitempty) = true ];
uint64 log_index = 7 [ (amino.dont_omitempty) = true ];
uint64 block_number = 8 [ (amino.dont_omitempty) = true ];
uint64 nonce = 9 [ (amino.dont_omitempty) = true ];
}
```

### MsgStakeUpdate

`MsgStakeUpdate` defines a message for a validator to perform a stake update on Ethereum network.

```protobuf
message MsgStakeUpdate {
option (amino.name) = "heimdallv2/MsgStakeUpdate";

option (gogoproto.equal) = false;
option (gogoproto.goproto_getters) = true;

string from = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];
uint64 val_id = 2 [ (amino.dont_omitempty) = true ];
string new_amount = 3 [
(gogoproto.nullable) = false,
(amino.dont_omitempty) = true,
(gogoproto.customtype) = "cosmossdk.io/math.Int"
];
bytes tx_hash = 4 [ (amino.dont_omitempty) = true ];
uint64 log_index = 5 [ (amino.dont_omitempty) = true ];
uint64 block_number = 6 [ (amino.dont_omitempty) = true ];
uint64 nonce = 7 [ (amino.dont_omitempty) = true ];
}
```

### MsgSignerUpdate

`MsgSignerUpdate` defines a message for updating the signer of the existing validator.

```protobuf
message MsgSignerUpdate {
  option (amino.name) = "heimdallv2/MsgSignerUpdate";

  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = true;

  string from = 1 [
    (amino.dont_omitempty) = true,
    (cosmos_proto.scalar) = "cosmos.AddressString"
  ];
  uint64 val_id = 2 [ (amino.dont_omitempty) = true ];
  google.protobuf.Any new_signer_pub_key = 3
      [ (cosmos_proto.accepts_interface) = "cosmos.crypto.PubKey" ];
  bytes tx_hash = 4 [ (amino.dont_omitempty) = true ];
  uint64 log_index = 5 [ (amino.dont_omitempty) = true ];
  uint64 block_number = 6 [ (amino.dont_omitempty) = true ];
  uint64 nonce = 7 [ (amino.dont_omitempty) = true ];
}
```

### MsgValidatorExit

`MsgValidatorExit` defines a message for a validator to exit the network.

```protobuf
message MsgValidatorExit {
option (amino.name) = "heimdallv2/MsgValidatorExit";

option (gogoproto.equal) = false;
option (gogoproto.goproto_getters) = true;

string from = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];
uint64 val_id = 2 [ (amino.dont_omitempty) = true ];
uint64 deactivation_epoch = 3 [ (amino.dont_omitempty) = true ];
bytes tx_hash = 4 [ (amino.dont_omitempty) = true ];
uint64 log_index = 5 [ (amino.dont_omitempty) = true ];
uint64 block_number = 6 [ (amino.dont_omitempty) = true ];
uint64 nonce = 7 [ (amino.dont_omitempty) = true ];
}
```

## Interact with the Node

### Tx Commands

#### Validator Join
```bash
./build/heimdalld tx stake validator-join --proposer {proposer address} --signer-pubkey {signer pubkey with 04 prefix} --tx-hash {tx hash} --block-number {L1 block number} --staked-amount {total stake amount} --activation-epoch {activation epoch} --home="{path to home}"
```

#### Signer Update
```bash
./build/heimdalld tx stake signer-update --proposer {proposer address} --id {val id} --new-pubkey {new pubkey with 04 prefix} --tx-hash {tx hash}  --log-index {log index} --block-number {L1 block number} --nonce {nonce} --home="{path to home}"
```

#### Stake Update
```bash
./build/heimdalld tx stake stake-update [valAddress] [valId] [amount] [txHash] [logIndex] [blockNumber] [nonce]
```

#### Validator Exit
```bash
./build/heimdalld tx stake validator-exit [valAddress] [valId] [deactivationEpoch] [txHash] [logIndex] [blockNumber] [nonce]
```

### CLI Query Commands

One can run the following query commands from the stake module:

* `current-validator-set` - Query all validators which are currently active in validator set
* `signer` - Query validator info for given validator address
* `validator` - Query validator info for a given validator id
* `validator-status` - Query validator status for given validator address
* `total-power` - Query total power of the validator set
* `is-old-tx` - Check if a tx is old (already submitted)

```bash
./build/heimdalld query stake current-validator-set
```

```bash
./build/heimdalld query stake signer [val_address]
```

```bash
./build/heimdalld query stake validator [id]
```

```bash
./build/heimdalld query stake validator-status [val_address]
```

```bash
./build/heimdalld query stake total-power
```

```bash
./build/heimdalld query stake is-old-tx [txHash] [logIndex]
```

### GRPC Endpoints

The endpoints and the params are defined in the [stake/query.proto](/proto/heimdallv2/stake/query.proto) file. Please refer them for more information about the optional params.

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.stake.Query/GetCurrentValidatorSet
```

```bash
grpcurl -plaintext -d '{"val_address": <>}' localhost:9090 heimdallv2.stake.Query/GetSignerByAddress
```

```bash
grpcurl -plaintext -d '{"id": <>}' localhost:9090 heimdallv2.stake.Query/GetValidatorById
```

```bash
grpcurl -plaintext -d '{"val_address": <>}' localhost:9090 heimdallv2.stake.Query/GetValidatorStatusByAddress
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.stake.Query/GetTotalPower
```

```bash
grpcurl -plaintext -d '{"tx_hash": <>, "log_index": <>}' localhost:9090 heimdallv2.stake.Query/IsStakeTxOld
```

```bash
grpcurl -plaintest -d '{"times": <>}' localhost:9090 heimdallv2.stake.Query/GetProposersByTimes
```

## REST APIs

The endpoints and the params are defined in the [stake/query.proto](/proto/heimdallv2/stake/query.proto) file. Please refer them for more information about the optional params.

```bash
curl localhost:1317/stake/validator-set
```

```bash
curl localhost:1317/stake/signer/{val_address}
```

```bash
curl localhost:1317/stake/validator/{id}
```


```bash
curl localhost:1317/stake/validator-status/{val_address}
```

```bash
curl localhost:1317/stake/total-power
```

```bash
curl localhost:1317/stake/is-old-tx?tx_hash=<tx-hash>&log_index=<log-index>
```

```bash
curl localhost:1317/stake/proposers/{times}
```
