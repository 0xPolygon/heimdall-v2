package processor

import (
	"bytes"
	"context"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

// broadcastEvent submits msg, then keeps the task alive: CheckTx acceptance does not mean the later
// side vote applied the event. The caller's next run ends the task once heimdall shows the event applied.
func (bp *BaseProcessor) broadcastEvent(ctx context.Context, msg sdk.Msg, event interface{}, waiting string) error {
	if err := bp.submitEvent(ctx, msg, event); err != nil {
		return err
	}
	return retryEvent(ctx, waiting, util.RetryStateSyncTaskDelay)
}

// submitEvent sends msg unless another task owns the event, this bridge broadcast it within
// rebroadcastDelay, or the same L1 event is already in the mempool.
func (bp *BaseProcessor) submitEvent(ctx context.Context, msg sdk.Msg, event interface{}) error {
	now := time.Now()
	if !liveEvents.claim(ctx, now) || !liveEvents.broadcastDue(ctx, now) {
		return nil
	}
	pending, err := bp.checkTxAgainstMempool(ctx, msg, event)
	if err != nil || pending {
		return err
	}
	response, err := bp.txBroadcaster.BroadcastToHeimdall(ctx, msg, event)
	if err != nil {
		return err
	}
	if response == nil || response.Code != abci.CodeTypeOK {
		return fmt.Errorf("broadcast was not accepted: %v", response)
	}
	liveEvents.recordBroadcast(ctx, now)
	return nil
}

func sameEventMsg(cdc codec.BinaryCodec, a, b sdk.Msg) bool {
	pa, okA := eventPayload(a)
	pb, okB := eventPayload(b)
	return okA && okB && sdk.MsgTypeURL(a) == sdk.MsgTypeURL(b) && bytes.Equal(cdc.MustMarshal(pa), cdc.MustMarshal(pb))
}

// eventPayload strips the local relayer address so a pending broadcast from any validator matches
// only when every L1 event field agrees.
func eventPayload(msg sdk.Msg) (proto.Message, bool) {
	switch msg := msg.(type) {
	case *clerkTypes.MsgEventRecord:
		cloned := *msg
		cloned.From = ""
		return &cloned, true
	case *stakeTypes.MsgValidatorJoin:
		cloned := *msg
		cloned.From = ""
		return &cloned, true
	case *stakeTypes.MsgStakeUpdate:
		cloned := *msg
		cloned.From = ""
		return &cloned, true
	case *stakeTypes.MsgSignerUpdate:
		cloned := *msg
		cloned.From = ""
		return &cloned, true
	case *stakeTypes.MsgValidatorExit:
		cloned := *msg
		cloned.From = ""
		return &cloned, true
	case *topupTypes.MsgTopupTx:
		cloned := *msg
		cloned.Proposer = ""
		return &cloned, true
	case *checkpointTypes.MsgCpAck:
		cloned := *msg
		cloned.From = ""
		return &cloned, true
	default:
		return nil, false
	}
}
