package sidetxs_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

func TestNewSideTxConfigurator(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()
	require.NotNil(t, cfg)
	require.Implements(t, (*sidetxs.SideTxConfigurator)(nil), cfg)
}

func TestSideTxConfigurator_RegisterSideHandler(t *testing.T) {
	tests := []struct {
		name        string
		msgURL      string
		handler     sidetxs.SideTxHandler
		expectError bool
		errContains string
	}{
		{
			name:   "register new side handler successfully",
			msgURL: "/test.v1.MsgTest",
			handler: func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
				return sidetxs.Vote_VOTE_YES
			},
			expectError: false,
		},
		{
			name:   "register duplicate side handler returns error",
			msgURL: "/test.v1.MsgTest",
			handler: func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
				return sidetxs.Vote_VOTE_NO
			},
			expectError: true,
			errContains: "already exists",
		},
		{
			name:        "register nil side handler",
			msgURL:      "/test.v1.MsgTestNil",
			handler:     nil,
			expectError: false,
		},
		{
			name:   "register side handler for different message type",
			msgURL: "/test.v1.MsgTest2",
			handler: func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
				return sidetxs.Vote_VOTE_YES
			},
			expectError: false,
		},
		{
			name:   "register side handler with empty msgURL",
			msgURL: "",
			handler: func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
				return sidetxs.Vote_VOTE_YES
			},
			expectError: false,
		},
	}

	cfg := sidetxs.NewSideTxConfigurator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cfg.RegisterSideHandler(tt.msgURL, tt.handler)
			if tt.expectError {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSideTxConfigurator_RegisterPostHandler(t *testing.T) {
	tests := []struct {
		name        string
		msgURL      string
		handler     sidetxs.PostTxHandler
		expectError bool
		errContains string
	}{
		{
			name:   "register new post handler successfully",
			msgURL: "/test.v1.MsgTest",
			handler: func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
				return nil
			},
			expectError: false,
		},
		{
			name:   "register duplicate post handler returns error",
			msgURL: "/test.v1.MsgTest",
			handler: func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
				return nil
			},
			expectError: true,
			errContains: "already exists",
		},
		{
			name:        "register nil post handler",
			msgURL:      "/test.v1.MsgTestNil",
			handler:     nil,
			expectError: false,
		},
		{
			name:   "register post handler for different message type",
			msgURL: "/test.v1.MsgTest2",
			handler: func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
				return nil
			},
			expectError: false,
		},
		{
			name:   "register post handler with empty msgURL",
			msgURL: "",
			handler: func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
				return nil
			},
			expectError: false,
		},
	}

	cfg := sidetxs.NewSideTxConfigurator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cfg.RegisterPostHandler(tt.msgURL, tt.handler)
			if tt.expectError {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSideTxConfigurator_GetSideHandler(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	// Register a handler
	msg := &banktypes.MsgSend{
		FromAddress: "from",
		ToAddress:   "to",
	}
	msgURL := sdk.MsgTypeURL(msg)

	expectedHandler := func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	}
	err := cfg.RegisterSideHandler(msgURL, expectedHandler)
	require.NoError(t, err)

	tests := []struct {
		name           string
		msg            sdk.Msg
		expectNonNil   bool
		expectedResult sidetxs.Vote
	}{
		{
			name:           "get existing handler",
			msg:            msg,
			expectNonNil:   true,
			expectedResult: sidetxs.Vote_VOTE_YES,
		},
		{
			name: "get non-existent handler",
			msg: &banktypes.MsgMultiSend{
				Inputs:  nil,
				Outputs: nil,
			},
			expectNonNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := cfg.GetSideHandler(tt.msg)
			if tt.expectNonNil {
				require.NotNil(t, handler)
				// Test handler functionality
				ctx := sdk.Context{}
				vote := handler(ctx, tt.msg)
				require.Equal(t, tt.expectedResult, vote)
			} else {
				require.Nil(t, handler)
			}
		})
	}
}

func TestSideTxConfigurator_GetPostHandler(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	// Register a post-handler
	msg := &banktypes.MsgSend{
		FromAddress: "from",
		ToAddress:   "to",
	}
	msgURL := sdk.MsgTypeURL(msg)

	expectedHandler := func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
		return nil
	}
	err := cfg.RegisterPostHandler(msgURL, expectedHandler)
	require.NoError(t, err)

	tests := []struct {
		name         string
		msg          sdk.Msg
		expectNonNil bool
	}{
		{
			name:         "get existing post handler",
			msg:          msg,
			expectNonNil: true,
		},
		{
			name: "get non-existent post handler",
			msg: &banktypes.MsgMultiSend{
				Inputs:  nil,
				Outputs: nil,
			},
			expectNonNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := cfg.GetPostHandler(tt.msg)
			if tt.expectNonNil {
				require.NotNil(t, handler)
				// Test handler functionality
				ctx := sdk.Context{}
				err := handler(ctx, tt.msg, sidetxs.Vote_VOTE_YES)
				require.NoError(t, err)
			} else {
				require.Nil(t, handler)
			}
		})
	}
}

func TestSideTxConfigurator_MultipleInstances(t *testing.T) {
	cfg1 := sidetxs.NewSideTxConfigurator()
	cfg2 := sidetxs.NewSideTxConfigurator()

	msg := &banktypes.MsgSend{
		FromAddress: "from",
		ToAddress:   "to",
	}
	msgURL := sdk.MsgTypeURL(msg)

	// Register handler in cfg1
	handler := func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	}
	err := cfg1.RegisterSideHandler(msgURL, handler)
	require.NoError(t, err)

	// cfg2 should not have the handler
	handler2 := cfg2.GetSideHandler(msg)
	require.Nil(t, handler2)

	// cfg1 should have the handler
	handler1 := cfg1.GetSideHandler(msg)
	require.NotNil(t, handler1)
}

func TestSideTxConfigurator_RegisterBothHandlers(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	msg := &banktypes.MsgSend{
		FromAddress: "from",
		ToAddress:   "to",
	}
	msgURL := sdk.MsgTypeURL(msg)

	// Register both side and post handler for the same message
	sideHandler := func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
		return sidetxs.Vote_VOTE_YES
	}
	postHandler := func(ctx sdk.Context, msg sdk.Msg, sideTxResult sidetxs.Vote) error {
		return nil
	}

	err := cfg.RegisterSideHandler(msgURL, sideHandler)
	require.NoError(t, err)

	err = cfg.RegisterPostHandler(msgURL, postHandler)
	require.NoError(t, err)

	// Both should be retrievable
	retrievedSideHandler := cfg.GetSideHandler(msg)
	require.NotNil(t, retrievedSideHandler)

	retrievedPostHandler := cfg.GetPostHandler(msg)
	require.NotNil(t, retrievedPostHandler)
}

func TestSideTxConfigurator_NilHandlersBehavior(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	msg := &banktypes.MsgSend{
		FromAddress: "from",
		ToAddress:   "to",
	}
	msgURL := sdk.MsgTypeURL(msg)

	// Register nil handlers
	err := cfg.RegisterSideHandler(msgURL, nil)
	require.NoError(t, err)

	err = cfg.RegisterPostHandler(msgURL, nil)
	require.NoError(t, err)

	// Getting nil handlers should return nil (not panic)
	sideHandler := cfg.GetSideHandler(msg)
	require.Nil(t, sideHandler)

	postHandler := cfg.GetPostHandler(msg)
	require.Nil(t, postHandler)
}

func TestSideTxConfigurator_VoteTypes(t *testing.T) {
	cfg := sidetxs.NewSideTxConfigurator()

	msg := &banktypes.MsgSend{
		FromAddress: "from",
		ToAddress:   "to",
	}
	msgURL := sdk.MsgTypeURL(msg)

	tests := []struct {
		name         string
		voteResult   sidetxs.Vote
		setupHandler func() sidetxs.SideTxHandler
	}{
		{
			name:       "handler returns VOTE_YES",
			voteResult: sidetxs.Vote_VOTE_YES,
			setupHandler: func() sidetxs.SideTxHandler {
				return func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
					return sidetxs.Vote_VOTE_YES
				}
			},
		},
		{
			name:       "handler returns VOTE_NO",
			voteResult: sidetxs.Vote_VOTE_NO,
			setupHandler: func() sidetxs.SideTxHandler {
				return func(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
					return sidetxs.Vote_VOTE_NO
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			testMsgURL := msgURL + "/" + tt.name
			err := cfg.RegisterSideHandler(testMsgURL, handler)
			require.NoError(t, err)

			// Test that handler executes correctly
			ctx := sdk.Context{}
			vote := handler(ctx, msg)
			require.Equal(t, tt.voteResult, vote)
		})
	}
}
