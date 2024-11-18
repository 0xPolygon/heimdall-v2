<!-- TODO HV2 - update/verify the models, query, cli, and REST behaviour -->

# Milestone module

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

## CLI Commands

## REST APIs
