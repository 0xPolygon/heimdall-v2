package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	types "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// AccountRootHashLenDecorator rejects a MsgCheckpoint whose AccountRootHash is not
// a 32-byte hash.
type AccountRootHashLenDecorator struct {
	activeFn func(int64) bool
}

func NewAccountRootHashLenDecorator(activeFn func(int64) bool) AccountRootHashLenDecorator {
	return AccountRootHashLenDecorator{activeFn: activeFn}
}

func (d AccountRootHashLenDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !d.activeFn(ctx.BlockHeight()) {
		return next(ctx, tx, simulate)
	}
	for _, msg := range tx.GetMsgs() {
		cp, ok := msg.(*types.MsgCheckpoint)
		if !ok {
			continue
		}
		if len(cp.AccountRootHash) != common.HashLength {
			return ctx, types.ErrInvalidMsg.Wrapf("invalid accountRootHash length %d, expected %d", len(cp.AccountRootHash), common.HashLength)
		}
	}
	return next(ctx, tx, simulate)
}
