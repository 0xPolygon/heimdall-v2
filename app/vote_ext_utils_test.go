package app

import (
	"bytes"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/protoio"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
)

func TestValidateVoteExtensions(t *testing.T) {
	hApp, _, _ := SetupApp(t, 1)
	ctx := hApp.BaseApp.NewContext(false)
	vals := hApp.StakeKeeper.GetAllValidators(ctx)
	valAddr, err := address.NewHexCodec().StringToBytes(vals[0].Signer)
	require.NoError(t, err)

	cometVal := abci.Validator{
		Address: valAddr,
		Power:   vals[0].VotingPower,
	}
	_, err = hApp.Commit()
	require.NoError(t, err)

	tests := []struct {
		name         string
		ctx          sdk.Context
		extVoteInfo  []abci.ExtendedVoteInfo
		round        int32
		keeper       stakeKeeper.Keeper
		shouldPanic  bool
		panicMessage string
		expectedErr  string
	}{
		// HV2: we only test the basic cases here
		{
			name: "ves disabled with non-empty vote extension",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 0),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, common.Hex2Bytes(TxHash1), common.Hex2Bytes(TxHash2), cometVal),
			},
			round:        1,
			keeper:       hApp.StakeKeeper,
			shouldPanic:  true,
			panicMessage: "VoteExtensions are disabled!",
		},
		// HV2: this test shows the correct execution of ValidateVoteExtensions function
		// It stops on `!cmtPubKey.VerifySignature(...)`. To proceed, we'd need to mock the pubKey
		{
			name: "function executed correctly, but failing on signature verification",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 10),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, common.Hex2Bytes(TxHash1), common.Hex2Bytes(TxHash2), cometVal),
			},
			round:       1,
			keeper:      hApp.StakeKeeper,
			shouldPanic: false,
			expectedErr: "failed to verify validator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				require.PanicsWithValue(t, tt.panicMessage, func() {
					err = ValidateVoteExtensions(tt.ctx, CurrentHeight, cometVal.Address, tt.extVoteInfo, tt.round, tt.keeper)
					fmt.Printf("err: %v\n", err)
				})
			} else {
				err := ValidateVoteExtensions(tt.ctx, CurrentHeight, cometVal.Address, tt.extVoteInfo, tt.round, tt.keeper)
				if tt.expectedErr != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.expectedErr)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestTallyVotes(t *testing.T) {
	// TODO add more tests:
	//  1. add some SKIP votes to test that too
	//  2. tx approved with just enough voting power (maybe using big numbers, to test the int-aliasing we do in the function). An example: totalVP: 9999, vote_YES 6667, vote_NO 3333
	//  3. tx rejected with almost enough voting power. Example: totalVP: 9999, vote_YES 6666, vote_NO 10
	//  4. sum of the votes exceeds the total voting power. Example: totalVP: 100, vote_YES 90, vote_NO 11
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
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   10,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   20,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   1,
					}),
			},
			expectedApprove: [][]byte{common.Hex2Bytes(TxHash1)},
			expectedReject:  make([][]byte, 0, 3),
			expectedSkip:    make([][]byte, 0, 3),
		},
		{
			name:        "one tx approved one rejected one skipped",
			votingPower: 75,
			extVoteInfo: []abci.ExtendedVoteInfo{
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   40,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash2),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   40,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash3),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   40,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   30,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash2),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   30,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash3),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   30,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash1),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   5,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash2),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   5,
					}),
			},
			expectedApprove: [][]byte{common.Hex2Bytes(TxHash1)},
			expectedReject:  [][]byte{common.Hex2Bytes(TxHash2)},
			expectedSkip:    [][]byte{common.Hex2Bytes(TxHash3)},
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
	txHashBytes := common.Hex2Bytes(TxHash1)
	blockHashBytes := common.Hex2Bytes(TxHash2)

	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.ConsolidatedSideTxResponse{
		SideTxResponses: []*sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
		},
		BlockHash: blockHashBytes,
		Height:    VoteExtBlockHeight,
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

	actualVotes, err := aggregateVotes(extVoteInfo, CurrentHeight, nil)
	require.NoError(t, err)
	require.NotEmpty(t, actualVotes)
	require.Equal(t, expectedVotes, actualVotes)
}

func TestValidateSideTxResponses(t *testing.T) {
	tests := []struct {
		name            string
		sideTxResponses []*sidetxs.SideTxResponse
		expectedError   bool
		expectedTxHash  []byte
	}{
		{
			name: "no duplicates",
			sideTxResponses: []*sidetxs.SideTxResponse{
				{TxHash: common.Hex2Bytes(TxHash1)},
				{TxHash: common.Hex2Bytes(TxHash2)},
				{TxHash: common.Hex2Bytes(TxHash3)},
			},
			expectedError:  false,
			expectedTxHash: nil,
		},
		{
			name: "one duplicate",
			sideTxResponses: []*sidetxs.SideTxResponse{
				{TxHash: common.Hex2Bytes(TxHash1)},
				{TxHash: common.Hex2Bytes(TxHash2)},
				{TxHash: common.Hex2Bytes(TxHash3)},
				{TxHash: common.Hex2Bytes(TxHash3)},
			},
			expectedError:  true,
			expectedTxHash: common.Hex2Bytes(TxHash3),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			txHash, err := validateSideTxResponses(tc.sideTxResponses)
			if tc.expectedError {
				require.Error(t, err)
			}
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

func TestPanicOnVoteExtensionsDisabled(t *testing.T) {
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
		{"height is equal to VoteExtensionsEnableHeight", int64(VoteExtEnableHeight), false},
		{"height is greater than VoteExtensionsEnableHeight", int64(VoteExtEnableHeight) + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if !tt.panics {
				require.NotPanics(t, func() {
					panicOnVoteExtensionsDisabled(ctx, tt.height)
				}, "panicOnVoteExtensionsDisabled panicked unexpectedly")
			} else {
				require.Panics(t, func() {
					panicOnVoteExtensionsDisabled(ctx, tt.height)
				}, "panicOnVoteExtensionsDisabled did not panic, but it should have")
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

func returnExtendedVoteInfo(flag cmtTypes.BlockIDFlag, extension, signature []byte, validator abci.Validator) abci.ExtendedVoteInfo {
	return abci.ExtendedVoteInfo{
		BlockIdFlag:        flag,
		VoteExtension:      extension,
		ExtensionSignature: signature,
		Validator:          validator,
	}
}

func setupExtendedVoteInfo(flag cmtTypes.BlockIDFlag, txHashBytes, blockHashBytes []byte, validator abci.Validator) abci.ExtendedVoteInfo {
	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.ConsolidatedSideTxResponse{
		SideTxResponses: []*sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
		},
		BlockHash: blockHashBytes,
		Height:    VoteExtBlockHeight,
	}

	// marshal it into Protobuf bytes
	voteExtensionBytes, _ := proto.Marshal(&voteExtensionProto)

	cve := cmtTypes.CanonicalVoteExtension{
		Extension: voteExtensionBytes,
		Height:    CurrentHeight - 1, // the vote extension was signed in the previous height
		Round:     int64(1),
		ChainId:   "",
	}

	marshalDelimitedFn := func(msg proto.Message) ([]byte, error) {
		var buf bytes.Buffer
		if _, err := protoio.NewDelimitedWriter(&buf).WriteMsg(msg); err != nil {
			return nil, err
		}

		return buf.Bytes(), nil
	}
	extSignBytes, err := marshalDelimitedFn(&cve)
	if err != nil {
		fmt.Printf("failed to encode CanonicalVoteExtension: %v", err)
	}

	return abci.ExtendedVoteInfo{
		BlockIdFlag:        flag,
		VoteExtension:      voteExtensionBytes,
		ExtensionSignature: extSignBytes,
		Validator:          validator,
	}
}
