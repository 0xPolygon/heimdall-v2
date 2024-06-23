package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis sets initial state for checkpoint module
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
	k.SetParams(ctx, data.Params)

	// Set last no-ack
	if data.LastNoACK > 0 {
		k.SetLastNoAck(ctx, data.LastNoACK)
	}

	// Add finalised checkpoints to state
	if len(data.Checkpoints) != 0 {
		// check if we are provided all the headers
		if int(data.AckCount) != len(data.Checkpoints) {
			panic(errors.New(fmt.Sprintf("incorrect state in state-dump , please Check", "ack count", data.AckCount, "checkpoints length", data.Checkpoints)))
		}
		// sort headers before loading to state
		data.Checkpoints = types.SortHeaders(data.Checkpoints)
		// load checkpoints to state
		for i, checkpoint := range data.Checkpoints {
			checkpointIndex := uint64(i) + 1
			if err := k.AddCheckpoint(ctx, checkpointIndex, checkpoint); err != nil {
				k.Logger(ctx).Error("error while adding the checkpoint to store",
					"checkpointIndex", checkpointIndex,
					"checkpoint", checkpoint.String(),
					"error", err)
			}
		}
	}

	// Add checkpoint in buffer
	if data.BufferedCheckpoint != nil {
		if err := k.SetCheckpointBuffer(ctx, *data.BufferedCheckpoint); err != nil {
			k.Logger(ctx).Error("error while setting the checkpoint in buffer", "error", err)
		}
	}

	// Set initial ack count
	k.UpdateACKCountWithValue(ctx, data.AckCount)

	return
}

// ExportGenesis returns a GenesisState for a given context and keeper of
// checkpoint module
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in getting checkpoint params in export genesis call", "error", err)

		return nil
	}

	checkpoints, err := k.GetCheckpoints(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in getting checkpoints in export genesis call", "error", err)
		return nil
	}

	bufferedCheckpoint, _ := k.GetCheckpointFromBuffer(ctx)
	return &types.GenesisState{
		Params:             params,
		BufferedCheckpoint: bufferedCheckpoint,
		LastNoACK:          k.GetLastNoAck(ctx),
		AckCount:           k.GetACKCount(ctx),
		Checkpoints:        types.SortHeaders(checkpoints),
	}
}
