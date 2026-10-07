package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// TxHashDecorator applies fixed-width event-record hash validation once its
// activation predicate is true.
type TxHashDecorator struct {
	activeFn func(int64) bool
}

func NewTxHashDecorator(activeFn func(int64) bool) TxHashDecorator {
	return TxHashDecorator{activeFn: activeFn}
}

func (d TxHashDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !d.activeFn(ctx.BlockHeight()) {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
		eventRecord, ok := msg.(*types.MsgEventRecord)
		if !ok {
			continue
		}
		if err := eventRecord.ValidateTxHash(); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}
