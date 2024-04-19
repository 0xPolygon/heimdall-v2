package types

import sdk "github.com/cosmos/cosmos-sdk/types"

var (
	// CheckpointAdjust method name
	CheckpointAdjustMethod = sdk.MsgTypeURL(&MsgCheckpointAdjust{})

	// Checkpoint method name
	CheckpointMethod = sdk.MsgTypeURL(&MsgCheckpoint{})

	// CheckpointAck method name
	CheckpointAckMethod = sdk.MsgTypeURL(&MsgCheckpointAck{})
)
