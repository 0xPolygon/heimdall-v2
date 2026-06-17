package ante

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	types "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg { return m.msgs }

func (m mockTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

func checkpointWithRootLen(n int) *types.MsgCheckpoint {
	return &types.MsgCheckpoint{AccountRootHash: make([]byte, n)}
}

func TestAccountRootHashLenDecorator_AnteHandle(t *testing.T) {
	tests := []struct {
		name           string
		active         bool
		msgs           []sdk.Msg
		wantErr        bool
		wantNextCalled bool
	}{
		{
			name:           "active rejects oversized accountRootHash",
			active:         true,
			msgs:           []sdk.Msg{checkpointWithRootLen(common.HashLength + 1)},
			wantErr:        true,
			wantNextCalled: false,
		},
		{
			name:           "active rejects short accountRootHash",
			active:         true,
			msgs:           []sdk.Msg{checkpointWithRootLen(common.HashLength - 1)},
			wantErr:        true,
			wantNextCalled: false,
		},
		{
			name:           "active rejects empty accountRootHash",
			active:         true,
			msgs:           []sdk.Msg{checkpointWithRootLen(0)},
			wantErr:        true,
			wantNextCalled: false,
		},
		{
			name:           "active accepts exact 32-byte accountRootHash",
			active:         true,
			msgs:           []sdk.Msg{checkpointWithRootLen(common.HashLength)},
			wantErr:        false,
			wantNextCalled: true,
		},
		{
			name:           "inactive accepts oversized accountRootHash",
			active:         false,
			msgs:           []sdk.Msg{checkpointWithRootLen(common.HashLength + 100)},
			wantErr:        false,
			wantNextCalled: true,
		},
		{
			name:           "active ignores non-checkpoint message",
			active:         true,
			msgs:           []sdk.Msg{&banktypes.MsgSend{FromAddress: "a", ToAddress: "b"}},
			wantErr:        false,
			wantNextCalled: true,
		},
		{
			name:           "active rejects checkpoint among multiple messages",
			active:         true,
			msgs:           []sdk.Msg{&banktypes.MsgSend{}, checkpointWithRootLen(10)},
			wantErr:        true,
			wantNextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := NewAccountRootHashLenDecorator(func(int64) bool { return tt.active })

			nextCalled := false
			next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
				nextCalled = true
				return ctx, nil
			}

			_, err := dec.AnteHandle(sdk.Context{}, mockTx{msgs: tt.msgs}, false, next)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantNextCalled, nextCalled)
		})
	}
}
