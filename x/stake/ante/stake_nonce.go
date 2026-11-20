package ante

import (
	"context"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type StakeKeeper interface {
	GetValidatorFromValID(context.Context, uint64) (types.Validator, error)
}

type StakeNonceDecorator struct {
	keeper StakeKeeper
	active func(int64) bool
}

func NewStakeNonceDecorator(keeper StakeKeeper, active func(int64) bool) StakeNonceDecorator {
	return StakeNonceDecorator{keeper: keeper, active: active}
}

func (d StakeNonceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if height, ok := admissionHeight(ctx); ok && d.active(height) {
		if err := ValidateStakeNonces(ctx, d.keeper, tx.GetMsgs()); err != nil {
			return ctx, err
		}
	}
	return next(ctx, tx, simulate)
}

// CheckTx runs on the committed block's state, so its next proposal is at height+1.
// FinalizeBlock runs after PreBlocker has already advanced nonces for the prior block's commit.
func admissionHeight(ctx sdk.Context) (int64, bool) {
	if ctx.IsCheckTx() {
		return ctx.BlockHeight() + 1, true
	}
	mode := ctx.ExecMode()
	return ctx.BlockHeight(), mode == sdk.ExecModePrepareProposal || mode == sdk.ExecModeProcessProposal
}

// ValidateStakeNonces rejects nonces past the next stored one. A proposal cannot credit
// another pending event, since its side vote has not been applied yet. Stale nonces stay
// admitted: the post-handler drops them without a state write, and rejecting them would
// evict the relayer's later txs on sequence once a duplicate relay of an applied event
// fails ReCheck or is left out of a proposal.
func ValidateStakeNonces(ctx context.Context, keeper StakeKeeper, msgs []sdk.Msg) error {
	for _, msg := range msgs {
		id, nonce, ok := NonceUpdate(msg)
		if !ok {
			continue
		}
		if keeper == nil {
			return fmt.Errorf("stake keeper is unavailable")
		}
		validator, err := keeper.GetValidatorFromValID(ctx, id)
		if err != nil {
			return fmt.Errorf("read validator %d: %w", id, err)
		}
		if validator.Nonce == math.MaxUint64 || nonce > validator.Nonce+1 {
			return types.ErrInvalidMsg.Wrapf("validator nonce %d is past the next one after %d", nonce, validator.Nonce)
		}
	}
	return nil
}

func NonceUpdate(msg sdk.Msg) (uint64, uint64, bool) {
	switch msg := msg.(type) {
	case *types.MsgStakeUpdate:
		return msg.ValId, msg.Nonce, true
	case *types.MsgSignerUpdate:
		return msg.ValId, msg.Nonce, true
	case *types.MsgValidatorExit:
		return msg.ValId, msg.Nonce, true
	default:
		return 0, 0, false
	}
}
