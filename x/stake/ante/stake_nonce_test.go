package ante

import (
	"context"
	"fmt"
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type nonceKeeper struct{ nonce uint64 }

func (k nonceKeeper) GetValidatorFromValID(_ context.Context, id uint64) (types.Validator, error) {
	if id == 99 {
		return types.Validator{}, fmt.Errorf("unknown validator")
	}
	return types.Validator{Nonce: k.nonce}, nil
}

type nonceTx struct{ msgs []sdk.Msg }

func (tx nonceTx) GetMsgs() []sdk.Msg                    { return tx.msgs }
func (tx nonceTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

// Nonces past the next stored one are rejected, for every nonce-ordered message; stale and next
// nonces pass, and unknown validators fail.
func TestNextStoredStakeNonce(t *testing.T) {
	for _, nonce := range []uint64{0, 4, math.MaxUint64} {
		for _, got := range []uint64{0, 3, 4, 5, 6} {
			for _, msg := range []sdk.Msg{
				&types.MsgStakeUpdate{ValId: 1, Nonce: got},
				&types.MsgSignerUpdate{ValId: 1, Nonce: got},
				&types.MsgValidatorExit{ValId: 1, Nonce: got},
			} {
				err := ValidateStakeNonces(context.Background(), nonceKeeper{nonce}, []sdk.Msg{msg})
				require.Equal(t, nonce == math.MaxUint64 || got > nonce+1, err != nil)
			}
		}
	}
	require.Error(t, ValidateStakeNonces(context.Background(), nonceKeeper{}, []sdk.Msg{&types.MsgStakeUpdate{ValId: 99, Nonce: 1}}))
	require.Error(t, ValidateStakeNonces(context.Background(), nil, []sdk.Msg{&types.MsgStakeUpdate{ValId: 1, Nonce: 1}}))
}

// The admission gate checks CheckTx against height+1, proposal verification against height, and skips FinalizeBlock.
func TestStakeNonceMempoolGate(t *testing.T) {
	for _, height := range []int64{8, 9, 10, 11} {
		for _, mode := range []sdk.ExecMode{sdk.ExecModeCheck, sdk.ExecModePrepareProposal, sdk.ExecModeProcessProposal, sdk.ExecModeFinalize} {
			ctx := sdk.Context{}.WithBlockHeight(height).WithIsCheckTx(mode == sdk.ExecModeCheck).WithExecMode(mode)
			called := false
			next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { called = true; return ctx, nil }
			_, err := NewStakeNonceDecorator(nonceKeeper{4}, func(height int64) bool { return height >= 10 }).AnteHandle(ctx, nonceTx{[]sdk.Msg{&types.MsgStakeUpdate{ValId: 1, Nonce: 6}}}, false, next)
			wantReject := (mode == sdk.ExecModeCheck && height+1 >= 10) || ((mode == sdk.ExecModePrepareProposal || mode == sdk.ExecModeProcessProposal) && height >= 10)
			require.Equal(t, wantReject, err != nil)
			require.Equal(t, !wantReject, called)
		}
	}
}

// ReCheck keeps a duplicate relay that went stale after the first copy applied, and still evicts future nonces.
func TestStakeNonceRecheckKeepsStale(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(10).WithIsReCheckTx(true).WithExecMode(sdk.ExecModeReCheck)
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }
	decorator := NewStakeNonceDecorator(nonceKeeper{4}, func(height int64) bool { return height >= 10 })
	for nonce, wantErr := range map[uint64]bool{3: false, 4: false, 5: false, 6: true} {
		_, err := decorator.AnteHandle(ctx, nonceTx{[]sdk.Msg{&types.MsgSignerUpdate{ValId: 1, Nonce: nonce}}}, false, next)
		require.Equal(t, wantErr, err != nil, "nonce %d", nonce)
	}
}

// Messages without a validator nonce, joins included, pass untouched.
func TestStakeNonceIgnoresOtherMessages(t *testing.T) {
	for _, msg := range []sdk.Msg{&banktypes.MsgSend{}, &types.MsgValidatorJoin{ValId: 1, Nonce: 0}} {
		require.NoError(t, ValidateStakeNonces(context.Background(), nil, []sdk.Msg{msg}))
	}
}
