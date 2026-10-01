package ante

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg { return m.msgs }

func (m mockTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

func TestTxHashDecorator(t *testing.T) {
	validHash := "0x" + strings.Repeat("ab", 32)
	oversizedHash := "0x00" + validHash[2:]

	tests := []struct {
		name           string
		active         bool
		msgs           []sdk.Msg
		wantErr        bool
		wantNextCalled bool
	}{
		{name: "inactive accepts oversized hash", active: false, msgs: []sdk.Msg{&types.MsgEventRecord{TxHash: oversizedHash}}, wantNextCalled: true},
		{name: "active rejects oversized hash", active: true, msgs: []sdk.Msg{&types.MsgEventRecord{TxHash: oversizedHash}}, wantErr: true},
		{name: "active accepts fixed-width hash", active: true, msgs: []sdk.Msg{&types.MsgEventRecord{TxHash: validHash}}, wantNextCalled: true},
		{name: "active ignores other message types", active: true, msgs: []sdk.Msg{&banktypes.MsgSend{}}, wantNextCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decorator := NewTxHashDecorator(func(int64) bool { return tt.active })
			nextCalled := false
			next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
				nextCalled = true
				return ctx, nil
			}

			_, err := decorator.AnteHandle(sdk.Context{}, mockTx{msgs: tt.msgs}, false, next)
			if tt.wantErr {
				require.ErrorIs(t, err, types.ErrInvalidTxHash)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantNextCalled, nextCalled)
		})
	}
}
