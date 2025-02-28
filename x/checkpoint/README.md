# Checkpoint Module

## Table of Contents

* [Overview](#overview)
* [Interact with the Node](#interact-with-the-node)
  * [Tx Commands](#tx-commands)
  * [CLI Query Commands](#cli-query-commands)
  * [GRPC Endpoints](#grpc-endpoints)
  * [REST Endpoints](#rest-endpoints)

## Overview

Heimdall selects the next proposer using Peppermint’s leader selection algorithm.  
The multi-stage checkpoint process is crucial due to potential failures when submitting checkpoints on the Ethereum chain caused by factors like gas limit, network traffic, or high gas fees.
Each checkpoint has a validator as the proposer.  
The outcome of a checkpoint on the Ethereum chain (success or failure) triggers an ack (acknowledgment) or no-ack (no acknowledgment) transaction,  
altering the proposer for the next checkpoint on Heimdall. 

![Checkpoint Flow.png](checkpoint_flow.png)

### Messages

#### MsgCheckpoint

`MsgCheckpoint` defines a message for creating a checkpoint on the Ethereum chain.

```protobuf
message MsgCheckpoint {
option (cosmos.msg.v1.signer) = "proposer";
option (amino.name) = "checkpoint/MsgCheckpoint";

option (gogoproto.equal) = true;
option (gogoproto.goproto_getters) = true;

string proposer = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];

uint64 start_block = 2 [ (amino.dont_omitempty) = true ];

uint64 end_block = 3 [ (amino.dont_omitempty) = true ];

bytes root_hash = 4 [ (amino.dont_omitempty) = true ];

bytes account_root_hash = 5 [ (amino.dont_omitempty) = true ];

string bor_chain_id = 6 [ (amino.dont_omitempty) = true ];
}
```

#### MsgCpAck

`MsgCpAck` defines a message for creating the ack tx of a submitted checkpoint.

```protobuf
message MsgCpAck {
option (cosmos.msg.v1.signer) = "from";
option (amino.name) = "checkpoint/MsgCpAck";

option (gogoproto.equal) = false;
option (gogoproto.goproto_getters) = true;

string from = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];

uint64 number = 2 [ (amino.dont_omitempty) = true ];

string proposer = 3 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];

uint64 start_block = 4 [ (amino.dont_omitempty) = true ];

uint64 end_block = 5 [ (amino.dont_omitempty) = true ];

bytes root_hash = 6 [ (amino.dont_omitempty) = true ];

bytes tx_hash = 7 [ (amino.dont_omitempty) = true ];

uint64 log_index = 8 [ (amino.dont_omitempty) = true ];
}
```

#### MsgCheckpointNoAck

`MsgCheckpointNoAck` defines a message for creating the no-ack tx of a checkpoint.

```protobuf
message MsgCheckpointNoAck {
option (cosmos.msg.v1.signer) = "from";

option (amino.name) = "checkpoint/MsgCheckpointNoAck";

option (gogoproto.equal) = false;
option (gogoproto.goproto_getters) = true;

string from = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];
}
```

## Interact with the Node

### Tx Commands

#### Send checkpoint
```bash
./build/heimdalld tx checkpoint send-checkpoint --proposer=<proposer-address> --start-block=<start-block-number> --end-block=<end-block-number> --root-hash=<root-hash> --account-root=<account-root> --bor-chain-id=<bor-chain-id> --chain-id=<chain-id> --auto-configure=true/false
```

#### Send checkpoint ack

```bash
./build/heimdalld tx checkpoint send-ack --tx-hash=<checkpoint-tx-hash> --log-index=<log-index> --header=<header> --proposer=<proposer-address> --chain-id=<heimdall-chainid>
```

#### Send checkpoint no-ack

```bash
./build/heimdalld tx checkpoint checkpoint-no-ack --from <from>
```

## CLI Query Commands

One can run the following query commands from the checkpoint module:

* `get-params` - Get checkpoint params
* `get-overview` - Get checkpoint overview
* `get-ack-count` - Get checkpoint ack count
* `get-checkpoint` - Get checkpoint based on its number
* `get-checkpoint-latest` - Get the latest checkpoint
* `get-checkpoint-buffer` - Get the checkpoint buffer
* `get-last-no-ack` - Get the last no ack
* `get-next-checkpoint` - Get the next checkpoint
* `get-current-proposer` - Get the current proposer
* `get-proposers` - Get the proposers
* `get-checkpoint-list` - Get the list of checkpoints

```bash
./build/heimdalld query checkpoint get-params
```

```bash
./build/heimdalld query checkpoint get-overview
```

```bash
./build/heimdalld query checkpoint get-ack-count
```

```bash
./build/heimdalld query checkpoint get-checkpoint
```

```bash
./build/heimdalld query checkpoint get-checkpoint-latest
```

```bash
./build/heimdalld query checkpoint get-checkpoint-buffer
```

```bash
./build/heimdalld query checkpoint get-last-no-ack
```

```bash
./build/heimdalld query checkpoint get-next-checkpoint
```

```bash
./build/heimdalld query checkpoint get-current-proposer
```

```bash
./build/heimdalld query checkpoint get-proposers
```

```bash
./build/heimdalld query checkpoint get-checkpoint-list
```

## GRPC Endpoints

The endpoints and the params are defined in the [checkpoint/query.proto](/proto/heimdallv2/checkpoint/query.proto) file. Please refer them for more information about the optional params.

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetCheckpointParams
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetCheckpointOverview
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetAckCount
```

```bash
grpcurl -plaintext -d '{"number": <>}' localhost:9090 heimdallv2.checkpoint.Query/GetCheckpoint
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetCheckpointLatest
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetCheckpointBuffer
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetLastNoAck
```

```bash
grpcurl -plaintext -d '{"bor_chain_id": <>}' localhost:9090 heimdallv2.checkpoint.Query/GetNextCheckpoint
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetCurrentProposer
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetProposers
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.checkpoint.Query/GetCheckpointList
```

```bash
grpcurl -plaintext -d '{"tx_hash": <>}' localhost:9090 heimdallv2.checkpoint.QueryGetCheckpointSignatures
```

## REST Endpoints

The endpoints and the params are defined in the [checkpoint/query.proto](/proto/heimdallv2/checkpoint/query.proto) file. Please refer them for more information about the optional params.


```bash
curl localhost:1317/checkpoints/params
```


```bash
curl localhost:1317/checkpoints/overview
```


```bash
curl localhost:1317/checkpoints/count
```


```bash
curl localhost:1317/checkpoints/latest
```


```bash
curl localhost:1317/checkpoints/buffer
```


```bash
curl localhost:1317/checkpoints/last-no-ack
```


```bash
curl localhost:1317/checkpoints/prepare-next/{bor-chain-id}
```


```bash
curl localhost:1317/checkpoint/proposers/current
```


```bash
curl localhost:1317/checkpoint/proposers/{times}
```

```bash
curl localhost:1317/checkpoints/list
```

```bash
curl localhost:1317/checkpoint/signatures/{tx_hash}
```

```bash
curl localhost:1317/checkpoints/{number}
```