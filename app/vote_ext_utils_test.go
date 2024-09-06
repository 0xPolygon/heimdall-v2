package app

import (
	"fmt"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
)

func TestValidateVoteExtensions(t *testing.T) {
	t.Skip("TODO HV2: fix and enable this test")
	_, db, logger := SetupApp(t, 1)
	hApp := NewHeimdallApp(logger, db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	ctx := cosmostestutil.DefaultContextWithKeys(hApp.keys, hApp.tKeys, nil)
	proposer, err := address.NewHexCodec().StringToBytes(ValAddr1)
	require.NoError(t, err)

	tests := []struct {
		name         string
		ctx          sdk.Context
		extVoteInfo  []abci.ExtendedVoteInfo
		round        int32
		keeper       stakeKeeper.Keeper
		shouldPanic  bool
		panicMessage string
		expectedErr  error
	}{
		{
			name: "ves disabled with non-empty vote extension",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 0),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, []byte("extension"), []byte{}, abci.Validator{}),
			},
			round:        1,
			keeper:       hApp.StakeKeeper,
			shouldPanic:  true,
			panicMessage: "mustAddSpecialTransaction should not be called before VoteExtensionsEnableHeight",
		},
		{
			name: "duplicate votes detected",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 10),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, []byte("extension"), []byte("signature"),
					abci.Validator{Address: proposer, Power: 10}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, []byte("extension"), []byte("signature"),
					abci.Validator{Address: proposer, Power: 10}),
			},
			round:        1,
			keeper:       hApp.StakeKeeper,
			shouldPanic:  true,
			panicMessage: "duplicated votes detected for validator",
		},
		{
			name: "vote.BlockIdFlag == types.BlockIDFlagUnknown",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 10),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagUnknown, []byte{}, []byte{}, abci.Validator{}),
			},
			round:       1,
			keeper:      hApp.StakeKeeper,
			shouldPanic: false,
			expectedErr: fmt.Errorf("received vote with unknown block ID flag at height %d", 1),
		},
		{
			name: "len(vote.ExtensionSignature) == 0",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 10),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, []byte("extension"), []byte{}, abci.Validator{}),
			},
			round:       1,
			keeper:      hApp.StakeKeeper,
			shouldPanic: false,
			expectedErr: fmt.Errorf("received empty vote extension signature at height %d", 1),
		},
		{
			name: "failed to verify validator vote extension signature",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 10),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, []byte("extension"), []byte("signature"),
					abci.Validator{}),
			},
			round:       1,
			keeper:      hApp.StakeKeeper,
			shouldPanic: false,
			expectedErr: fmt.Errorf("failed to verify validator %X vote extension signature", proposer),
		},
		{
			name: "sumVP.Int64() <= (2*totalVP)/3",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 10),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, []byte("extension"), []byte("signature"),
					abci.Validator{}),
			},
			round:       1,
			keeper:      hApp.StakeKeeper,
			shouldPanic: false,
			expectedErr: fmt.Errorf("insufficient cumulative voting power received to verify vote extensions; "+
				"got: %d, expected: >=%d", 100, 150),
		},
		{
			name: "sumVP.Int64() > (2*totalVP)/3",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 10),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, []byte("extension"), []byte("signature"),
					abci.Validator{}),
			},
			round:       1,
			keeper:      hApp.StakeKeeper,
			shouldPanic: false,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				require.PanicsWithValue(t, tt.panicMessage, func() {
					_ = ValidateVoteExtensions(tt.ctx, CurrentHeight, proposer, tt.extVoteInfo, tt.round, tt.keeper)
				})
			} else {
				err := ValidateVoteExtensions(tt.ctx, CurrentHeight, proposer, tt.extVoteInfo, tt.round, tt.keeper)
				if tt.expectedErr != nil {
					require.Error(t, err)
					require.EqualError(t, err, tt.expectedErr.Error())
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestTallyVotes(t *testing.T) {
	val1, err := address.NewHexCodec().StringToBytes(ValAddr1)
	require.NoError(t, err)
	val2, err := address.NewHexCodec().StringToBytes(ValAddr2)
	require.NoError(t, err)
	val3, err := address.NewHexCodec().StringToBytes(ValAddr3)
	require.NoError(t, err)
	tests := []struct {
		name            string
		extVoteInfo     []abci.ExtendedVoteInfo
		votingPower     int64
		expectedApprove [][]byte
		expectedReject  [][]byte
		expectedSkip    [][]byte
	}{
		{
			name:        "single tx approved with 2/3+1 majority",
			votingPower: 31,
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   10,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   20,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   1,
					}),
			},
			expectedApprove: [][]byte{[]byte(TxHash1)},
			expectedReject:  make([][]byte, 0, 3),
			expectedSkip:    make([][]byte, 0, 3),
		},
		{
			name:        "one tx approved one rejected one skipped",
			votingPower: 75,
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   40,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash2),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   40,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash3),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   40,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   30,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash2),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   30,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash3),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   30,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   5,
					}),
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash2),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   5,
					}),
			},
			expectedApprove: [][]byte{[]byte(TxHash1)},
			expectedReject:  [][]byte{[]byte(TxHash2)},
			expectedSkip:    [][]byte{[]byte(TxHash3)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			approvedTxs, rejectedTxs, skippedTxs, err := tallyVotes(tc.extVoteInfo, log.NewTestLogger(t), tc.votingPower, CurrentHeight)
			require.NoError(t, err)
			require.Equal(t, tc.expectedApprove, approvedTxs)
			require.Equal(t, tc.expectedReject, rejectedTxs)
			require.Equal(t, tc.expectedSkip, skippedTxs)
		})
	}
}

func TestAggregateVotes(t *testing.T) {
	txHashBytes := []byte(TxHash1)
	blockHashBytes := []byte(TxHash2)

	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.ConsolidatedSideTxResponse{
		SideTxResponses: []*sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
		},
		Hash:   blockHashBytes,
		Height: VoteExtBlockHeight,
	}

	// marshal it into Protobuf bytes
	voteExtensionBytes, err := proto.Marshal(&voteExtensionProto)
	require.NoError(t, err)

	val1, err := address.NewHexCodec().StringToBytes(ValAddr1)
	require.NoError(t, err)

	extVoteInfo := []abci.ExtendedVoteInfo{
		{
			Validator: abci.Validator{
				Address: val1,
				Power:   10,
			},
			VoteExtension:      voteExtensionBytes,
			ExtensionSignature: []byte("signature"),
			BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
		},
	}

	expectedVotes := map[string]map[sidetxs.Vote]int64{
		TxHash1: {
			sidetxs.Vote_VOTE_YES: 10,
		},
	}

	actualVotes, err := aggregateVotes(extVoteInfo, CurrentHeight)
	require.NoError(t, err)
	require.NotEmpty(t, actualVotes)
	require.Equal(t, expectedVotes, actualVotes)
}

func TestCheckDuplicateVotes(t *testing.T) {
	tests := []struct {
		name              string
		sideTxResponses   []*sidetxs.SideTxResponse
		expectedDuplicate bool
		expectedTxHash    []byte
	}{
		{
			name: "no duplicates",
			sideTxResponses: []*sidetxs.SideTxResponse{
				{TxHash: []byte(TxHash1)},
				{TxHash: []byte(TxHash2)},
				{TxHash: []byte(TxHash3)},
			},
			expectedDuplicate: false,
			expectedTxHash:    nil,
		},
		{
			name: "one duplicate",
			sideTxResponses: []*sidetxs.SideTxResponse{
				{TxHash: []byte(TxHash1)},
				{TxHash: []byte(TxHash2)},
				{TxHash: []byte(TxHash3)},
				{TxHash: []byte(TxHash3)},
			},
			expectedDuplicate: true,
			expectedTxHash:    []byte(TxHash3),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			duplicate, txHash := checkDuplicateVotes(tc.sideTxResponses)
			require.Equal(t, tc.expectedDuplicate, duplicate)
			require.Equal(t, tc.expectedTxHash, txHash)
		})
	}
}

func TestIsVoteValid(t *testing.T) {
	require.True(t, isVoteValid(sidetxs.Vote_UNSPECIFIED))
	require.True(t, isVoteValid(sidetxs.Vote_VOTE_YES))
	require.True(t, isVoteValid(sidetxs.Vote_VOTE_NO))
	require.False(t, isVoteValid(100))
	require.False(t, isVoteValid(-1))
}

func TestMustAddSpecialTransaction(t *testing.T) {
	VoteExtEnableHeight := 100
	key := storetypes.NewKVStoreKey("testStoreKey")
	testCtx := cosmostestutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := setupContextWithVoteExtensionsEnableHeight(testCtx.Ctx, int64(VoteExtEnableHeight))

	tests := []struct {
		name   string
		height int64
		panics bool
	}{
		{"height is less than VoteExtensionsEnableHeight", int64(VoteExtEnableHeight) - 1, true},
		{"height is equal to VoteExtensionsEnableHeight", int64(VoteExtEnableHeight), true},
		{"height is greater than VoteExtensionsEnableHeight", int64(VoteExtEnableHeight) + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if !tt.panics {
				require.NotPanics(t, func() {
					mustAddSpecialTransaction(ctx, tt.height)
				}, "mustAddSpecialTransaction panicked unexpectedly")
			} else {
				require.Panics(t, func() {
					mustAddSpecialTransaction(ctx, tt.height)
				}, "mustAddSpecialTransaction did not panic, but it should have")
			}
		})
	}
}

func setupContextWithVoteExtensionsEnableHeight(ctx sdk.Context, vesEnableHeight int64) sdk.Context {
	return ctx.WithConsensusParams(cmtTypes.ConsensusParams{
		Abci: &cmtTypes.ABCIParams{
			VoteExtensionsEnableHeight: vesEnableHeight,
		},
	})
}

func setupExtendedVoteInfo(flag cmtTypes.BlockIDFlag, extension, signature []byte, validator abci.Validator) abci.ExtendedVoteInfo {
	return abci.ExtendedVoteInfo{
		BlockIdFlag:        flag,
		VoteExtension:      extension,
		ExtensionSignature: signature,
		Validator:          validator,
	}
}
