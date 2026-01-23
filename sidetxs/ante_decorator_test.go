package sidetxs_test

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

type mockAuthSigningTx struct {
	msgs []sdk.Msg
}

func (m *mockAuthSigningTx) GetMsgs() []sdk.Msg {
	return m.msgs
}

func (m *mockAuthSigningTx) GetMsgsV2() ([]proto.Message, error) {
	msgs := make([]proto.Message, len(m.msgs))
	for i, msg := range m.msgs {
		if protoMsg, ok := msg.(proto.Message); ok {
			msgs[i] = protoMsg
		}
	}
	return msgs, nil
}

func (m *mockAuthSigningTx) GetSigners() ([][]byte, error) {
	return [][]byte{[]byte("test-signer")}, nil
}

func (m *mockAuthSigningTx) GetPubKeys() ([]cryptotypes.PubKey, error) {
	return []cryptotypes.PubKey{}, nil
}

func (m *mockAuthSigningTx) GetSignaturesV2() ([]signing.SignatureV2, error) {
	return []signing.SignatureV2{}, nil
}

func (m *mockAuthSigningTx) GetGas() uint64 {
	return 100000
}

func (m *mockAuthSigningTx) GetFee() sdk.Coins {
	return sdk.NewCoins()
}

func (m *mockAuthSigningTx) FeePayer() []byte {
	return []byte("test-payer")
}

func (m *mockAuthSigningTx) FeeGranter() []byte {
	return nil
}

func (m *mockAuthSigningTx) GetMemo() string {
	return ""
}

func (m *mockAuthSigningTx) GetTimeoutHeight() uint64 {
	return 0
}

func (m *mockAuthSigningTx) ValidateBasic() error {
	return nil
}

type nonAuthSigningTx struct {
	msgs []sdk.Msg
}

func (n nonAuthSigningTx) GetMsgs() []sdk.Msg {
	return n.msgs
}

func (n nonAuthSigningTx) GetMsgsV2() ([]proto.Message, error) {
	return nil, nil
}

func TestNewSideTxDecorator(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()
	decorator := sidetxs.NewSideTxDecorator(cfg)
	require.NotNil(t, decorator)
}

func TestSideTxDecorator_AnteHandle_ValidTxNoSideHandlers(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()
	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{
			&banktypes.MsgSend{
				FromAddress: "test",
				ToAddress:   "test",
			},
		},
	}

	ctx := sdk.Context{}
	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	_, err := decorator.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, nextCalled)
}

func TestSideTxDecorator_AnteHandle_ValidTxOneSideHandler(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	msg := &banktypes.MsgSend{
		FromAddress: "test",
		ToAddress:   "test",
	}
	msgURL := sdk.MsgTypeURL(msg)

	err := cfg.RegisterSideHandler(msgURL, func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	})
	require.NoError(t, err)

	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{msg},
	}

	ctx := sdk.Context{}
	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	_, err = decorator.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, nextCalled)
}

func TestSideTxDecorator_AnteHandle_MultipleSideHandlersError(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	msg1 := &banktypes.MsgSend{
		FromAddress: "test1",
		ToAddress:   "test1",
	}
	msg2 := &banktypes.MsgMultiSend{}

	err := cfg.RegisterSideHandler(sdk.MsgTypeURL(msg1), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	})
	require.NoError(t, err)

	err = cfg.RegisterSideHandler(sdk.MsgTypeURL(msg2), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	})
	require.NoError(t, err)

	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{msg1, msg2},
	}

	ctx := sdk.Context{}
	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	_, err = decorator.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple messages in a single side transaction")
	require.False(t, nextCalled)
}

func TestSideTxDecorator_AnteHandle_MixedMessages(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	sideMsg := &banktypes.MsgSend{
		FromAddress: "test",
		ToAddress:   "test",
	}
	regularMsg := &banktypes.MsgMultiSend{}

	err := cfg.RegisterSideHandler(sdk.MsgTypeURL(sideMsg), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	})
	require.NoError(t, err)

	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{sideMsg, regularMsg},
	}

	ctx := sdk.Context{}
	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	_, err = decorator.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, nextCalled)
}

func TestSideTxDecorator_AnteHandle_InvalidTxType(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()
	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := nonAuthSigningTx{
		msgs: []sdk.Msg{
			&banktypes.MsgSend{
				FromAddress: "test",
				ToAddress:   "test",
			},
		},
	}

	ctx := sdk.Context{}
	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	_, err := decorator.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid transaction type")
	require.False(t, nextCalled)
}

func TestSideTxDecorator_AnteHandle_EmptyTx(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()
	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{},
	}

	ctx := sdk.Context{}
	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	_, err := decorator.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, nextCalled)
}

func TestSideTxDecorator_AnteHandle_SimulateMode(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	msg := &banktypes.MsgSend{
		FromAddress: "test",
		ToAddress:   "test",
	}
	err := cfg.RegisterSideHandler(sdk.MsgTypeURL(msg), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	})
	require.NoError(t, err)

	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{msg},
	}

	ctx := sdk.Context{}
	nextCalled := false
	simulateFlag := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		simulateFlag = simulate
		return ctx, nil
	}

	_, err = decorator.AnteHandle(ctx, tx, true, next)
	require.NoError(t, err)
	require.True(t, nextCalled)
	require.True(t, simulateFlag)
}

func TestSideTxDecorator_AnteHandle_NextHandlerError(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()
	decorator := sidetxs.NewSideTxDecorator(cfg)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{
			&banktypes.MsgSend{
				FromAddress: "test",
				ToAddress:   "test",
			},
		},
	}

	ctx := sdk.Context{}
	expectedErr := errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "next handler error")
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, expectedErr
	}

	_, err := decorator.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.ErrorIs(t, err, expectedErr)
}

func TestCountSideHandlers(t *testing.T) {
	tests := []struct {
		name          string
		msgs          []sdk.Msg
		setupCfg      func(cfg sidetxs.SideTxConfigurator)
		expectedCount int
	}{
		{
			name: "no side handlers",
			msgs: []sdk.Msg{
				&banktypes.MsgSend{FromAddress: "test1", ToAddress: "test2"},
				&banktypes.MsgMultiSend{},
			},
			setupCfg:      func(cfg sidetxs.SideTxConfigurator) {},
			expectedCount: 0,
		},
		{
			name: "one side handler",
			msgs: []sdk.Msg{
				&banktypes.MsgSend{FromAddress: "test", ToAddress: "test"},
				&banktypes.MsgMultiSend{},
			},
			setupCfg: func(cfg sidetxs.SideTxConfigurator) {
				msg := &banktypes.MsgSend{}
				err := cfg.RegisterSideHandler(sdk.MsgTypeURL(msg), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
					return sidetxs.Vote_VOTE_YES
				})
				require.NoError(t, err)
			},
			expectedCount: 1,
		},
		{
			name: "multiple side handlers",
			msgs: []sdk.Msg{
				&banktypes.MsgSend{FromAddress: "test", ToAddress: "test"},
				&banktypes.MsgMultiSend{},
			},
			setupCfg: func(cfg sidetxs.SideTxConfigurator) {
				msg1 := &banktypes.MsgSend{}
				err := cfg.RegisterSideHandler(sdk.MsgTypeURL(msg1), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
					return sidetxs.Vote_VOTE_YES
				})
				require.NoError(t, err)
				msg2 := &banktypes.MsgMultiSend{}
				err = cfg.RegisterSideHandler(sdk.MsgTypeURL(msg2), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
					return sidetxs.Vote_VOTE_NO
				})
				require.NoError(t, err)
			},
			expectedCount: 2,
		},
		{
			name:          "empty transaction",
			msgs:          []sdk.Msg{},
			setupCfg:      func(cfg sidetxs.SideTxConfigurator) {},
			expectedCount: 0,
		},
		{
			name: "all messages have side handlers",
			msgs: []sdk.Msg{
				&banktypes.MsgSend{FromAddress: "test", ToAddress: "test"},
				&banktypes.MsgMultiSend{},
			},
			setupCfg: func(cfg sidetxs.SideTxConfigurator) {
				msg1 := &banktypes.MsgSend{}
				err := cfg.RegisterSideHandler(sdk.MsgTypeURL(msg1), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
					return sidetxs.Vote_VOTE_YES
				})
				require.NoError(t, err)
				msg2 := &banktypes.MsgMultiSend{}
				err = cfg.RegisterSideHandler(sdk.MsgTypeURL(msg2), func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
					return sidetxs.Vote_VOTE_NO
				})
				require.NoError(t, err)
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := sidetxs.NewSideTxConfigurator()
			tt.setupCfg(cfg)

			tx := &mockAuthSigningTx{msgs: tt.msgs}
			count := sidetxs.CountSideHandlers(cfg, tx)
			require.Equal(t, tt.expectedCount, count)
		})
	}
}

func TestCountSideHandlers_WithNilHandler(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	msg := &banktypes.MsgSend{}
	err := cfg.RegisterSideHandler(sdk.MsgTypeURL(msg), nil)
	require.NoError(t, err)

	tx := &mockAuthSigningTx{
		msgs: []sdk.Msg{msg},
	}

	count := sidetxs.CountSideHandlers(cfg, tx)
	require.Equal(t, 0, count)
}
