package sidetxs_test

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

// mockSideMsgServer implements SideMsgServer interface for testing
type mockSideMsgServer struct {
	sideHandlers map[string]sidetxs.SideTxHandler
	postHandlers map[string]sidetxs.PostTxHandler
}

func newMockSideMsgServer() *mockSideMsgServer {
	return &mockSideMsgServer{
		sideHandlers: make(map[string]sidetxs.SideTxHandler),
		postHandlers: make(map[string]sidetxs.PostTxHandler),
	}
}

func (m *mockSideMsgServer) SideTxHandler(methodName string) sidetxs.SideTxHandler {
	return m.sideHandlers[methodName]
}

func (m *mockSideMsgServer) PostTxHandler(methodName string) sidetxs.PostTxHandler {
	return m.postHandlers[methodName]
}

func (m *mockSideMsgServer) RegisterHandlers(msgURL string, sideHandler sidetxs.SideTxHandler, postHandler sidetxs.PostTxHandler) {
	m.sideHandlers[msgURL] = sideHandler
	m.postHandlers[msgURL] = postHandler
}

func TestSideTxHandler_FunctionType(t *testing.T) {
	var handler sidetxs.SideTxHandler

	handler = func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	}

	require.NotNil(t, handler)

	ctx := sdk.Context{}
	msg := &banktypes.MsgSend{}

	vote := handler(ctx, msg)
	require.Equal(t, sidetxs.Vote_VOTE_YES, vote)
}

func TestSideTxHandler_DifferentVoteResults(t *testing.T) {
	tests := []struct {
		name         string
		handler      sidetxs.SideTxHandler
		expectedVote sidetxs.Vote
	}{
		{
			name: "returns VOTE_YES",
			handler: func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
				return sidetxs.Vote_VOTE_YES
			},
			expectedVote: sidetxs.Vote_VOTE_YES,
		},
		{
			name: "returns VOTE_NO",
			handler: func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
				return sidetxs.Vote_VOTE_NO
			},
			expectedVote: sidetxs.Vote_VOTE_NO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := sdk.Context{}
			msg := &banktypes.MsgSend{}
			vote := tt.handler(ctx, msg)
			require.Equal(t, tt.expectedVote, vote)
		})
	}
}

func TestPostTxHandler_FunctionType(t *testing.T) {
	var handler sidetxs.PostTxHandler

	handler = func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
		require.Equal(t, sidetxs.Vote_VOTE_YES, sideTxResult)
		return nil
	}

	require.NotNil(t, handler)

	ctx := sdk.Context{}
	msg := &banktypes.MsgSend{}

	err := handler(ctx, msg, sidetxs.Vote_VOTE_YES)
	require.NoError(t, err)
}

func TestPostTxHandler_WithError(t *testing.T) {
	handler := func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
		if sideTxResult == sidetxs.Vote_VOTE_NO {
			return errors.New("unauthorized")
		}
		return nil
	}

	ctx := sdk.Context{}
	msg := &banktypes.MsgSend{}

	err := handler(ctx, msg, sidetxs.Vote_VOTE_YES)
	require.NoError(t, err)

	err = handler(ctx, msg, sidetxs.Vote_VOTE_NO)
	require.Error(t, err)
}

func TestSideMsgServer_Interface(t *testing.T) {
	srv := newMockSideMsgServer()

	require.Implements(t, (*sidetxs.SideMsgServer)(nil), srv)
}

func TestSideMsgServer_SideTxHandler(t *testing.T) {
	srv := newMockSideMsgServer()

	msgURL := "/test.v1.MsgTest"
	handler := func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	}

	srv.RegisterHandlers(msgURL, handler, nil)

	retrievedHandler := srv.SideTxHandler(msgURL)
	require.NotNil(t, retrievedHandler)

	ctx := sdk.Context{}
	msg := &banktypes.MsgSend{}
	vote := retrievedHandler(ctx, msg)
	require.Equal(t, sidetxs.Vote_VOTE_YES, vote)
}

func TestSideMsgServer_PostTxHandler(t *testing.T) {
	srv := newMockSideMsgServer()

	msgURL := "/test.v1.MsgTest"
	handler := func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
		return nil
	}

	srv.RegisterHandlers(msgURL, nil, handler)

	retrievedHandler := srv.PostTxHandler(msgURL)
	require.NotNil(t, retrievedHandler)

	ctx := sdk.Context{}
	msg := &banktypes.MsgSend{}
	err := retrievedHandler(ctx, msg, sidetxs.Vote_VOTE_YES)
	require.NoError(t, err)
}

func TestSideMsgServer_NilHandlers(t *testing.T) {
	srv := newMockSideMsgServer()

	msgURL := "/test.v1.MsgTest"
	srv.RegisterHandlers(msgURL, nil, nil)

	sideHandler := srv.SideTxHandler(msgURL)
	require.Nil(t, sideHandler)

	postHandler := srv.PostTxHandler(msgURL)
	require.Nil(t, postHandler)
}

func TestSideMsgServer_NonExistentHandler(t *testing.T) {
	srv := newMockSideMsgServer()

	sideHandler := srv.SideTxHandler("/nonexistent")
	require.Nil(t, sideHandler)

	postHandler := srv.PostTxHandler("/nonexistent")
	require.Nil(t, postHandler)
}
