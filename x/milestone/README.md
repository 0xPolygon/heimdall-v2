# Milestone module

## Table of Contents

* [Overview](#overview)
* [Interact with the Node](#interact-with-the-node)
  * [Tx Commands](#tx-commands)
  * [CLI Query Commands](#cli-query-commands)
  * [GRPC Endpoints](#grpc-endpoints)
  * [REST Endpoints](#rest-endpoints)

## Overview

This module enables deterministic finality by leveraging Polygon PoS’s dual client architecture.  
This is done using a hybrid system that utilizes Peppermint layer consensus,  
along with an additional fork choice rule within the execution layer.
With the introduction of milestones, finality is deterministic even before a checkpoint is submitted to L1.  
After a certain number of blocks (minimum 12), a milestone is proposed and voted by Heimdall.  
Once 2/3+ of the network agrees, the milestone is finalized, and all transactions up to that milestone are considered final, with no chance of reorganization.

## Messages

### MsgMilestone

`MsgMilestone` defines a message for submitting a milestone
```protobuf
message MsgMilestone {
option (cosmos.msg.v1.signer) = "proposer";
option (amino.name) = "heimdallv2/MsgMilestone";
option (cosmos.msg.v1.signer) = "proposer";

option (gogoproto.equal) = true;
option (gogoproto.goproto_getters) = true;

string proposer = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];

uint64 start_block = 2 [ (amino.dont_omitempty) = true ];

uint64 end_block = 3 [ (amino.dont_omitempty) = true ];

bytes hash = 4 [ (amino.dont_omitempty) = true ];

string bor_chain_id = 5 [ (amino.dont_omitempty) = true ];

string milestone_id = 6 [ (amino.dont_omitempty) = true ];
}
```

### MsgMilestone

`MsgMilestone` defines a message for submitting a milestone
```protobuf
message MsgMilestone {
option (cosmos.msg.v1.signer) = "proposer";
option (amino.name) = "heimdallv2/MsgMilestone";
option (cosmos.msg.v1.signer) = "proposer";

option (gogoproto.equal) = true;
option (gogoproto.goproto_getters) = true;

string proposer = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];

uint64 start_block = 2 [ (amino.dont_omitempty) = true ];

uint64 end_block = 3 [ (amino.dont_omitempty) = true ];

bytes hash = 4 [ (amino.dont_omitempty) = true ];

string bor_chain_id = 5 [ (amino.dont_omitempty) = true ];

string milestone_id = 6 [ (amino.dont_omitempty) = true ];
}
```

### MsgMilestoneTimeout

`MsgMilestoneTimeout` defines SDK message to indicate that no milestone was proposed within the timeout period.

```protobuf
message MsgMilestoneTimeout {
option (cosmos.msg.v1.signer) = "from";
option (amino.name) = "heimdallv2/MsgMilestoneTimeout";

option (gogoproto.equal) = true;
option (gogoproto.goproto_getters) = true;

string from = 1 [
(amino.dont_omitempty) = true,
(cosmos_proto.scalar) = "cosmos.AddressString"
];
}
```

## Interact with the Node

### Tx Commands

#### Send Milestone Transaction 
```bash
./build/heimdalld tx milestone milestone [proposer] [startBlock] [endBlock] [hash] [borChainId] [milestoneId]
```

#### Send milestone timeout tx
```bash
./build/heimdalld tx milestone milestone-timeout
```

### CLI Query Commands

One can run the following query commands from the milestone module:

* `get-params` - Get milestone params
* `get-count` - Get milestone count
* `get-latest-milestone` - Get latest milestone
* `get-milestone-by-number` - Get milestone by number
* `get-milestone-proposer` - Get milestone proposer
* `get-latest-no-ack-milestone` - Get latest no ack milestone
* `get-no-ack-milestone-by-id` - Get no ack milestone by id

```bash
./build/heimdalld query milestone get-params
```

```bash
./build/heimdalld query milestone get-count
```

```bash
./build/heimdalld query milestone get-latest-milestone
```

```bash
./build/heimdalld query milestone get-milestone-by-number
```

```bash
./build/heimdalld query milestone get-milestone-proposer
```

```bash
./build/heimdalld query milestone get-latest-no-ack-milestone
```

```bash
./build/heimdalld query milestone get-no-ack-milestone-by-id
```

### GRPC Endpoints

The endpoints and the params are defined in the [milestone/query.proto](/proto/heimdallv2/milestone/query.proto) file. Please refer them for more information about the optional params.

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.milestone.Query/GetMilestoneParams
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.milestone.Query/GetMilestoneCount
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.milestone.Query/GetLatestMilestone
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.milestone.Query/GetMilestoneByNumber
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.milestone.Query/GetMilestoneProposerByTimes
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.milestone.Query/GetLatestNoAckMilestone
```

```bash
grpcurl -plaintext -d '{}' localhost:9090 heimdallv2.milestone.Query/GetNoAckMilestoneById
```

### REST APIs

The endpoints and the params are defined in the [milestone/query.proto](/proto/heimdallv2/milestone/query.proto) file. Please refer them for more information about the optional params.

```bash
curl localhost:1317/milestone/params
```

```bash
curl localhost:1317/milestone/count
```

```bash
curl localhost:1317/milestone/latest
```

```bash
curl localhost:1317/milestone/{number}
```

```bash
curl localhost:1317/milestone/proposer/{times}
```

```bash
curl localhost:1317/milestone/last-no-ack
```

```bash
curl localhost:1317/milestone/no-ack/{id}
```

