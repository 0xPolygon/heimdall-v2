package app

import (
	"bytes"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/protoio"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestValidateVoteExtensions(t *testing.T) {
	// TODO HV2: find a way to extend these tests (see https://github.com/0xPolygon/heimdall-v2/pull/60/#discussion_r1768825790)
	hApp, _, _ := SetupApp(t, 1)
	ctx := hApp.BaseApp.NewContext(true)
	vals := hApp.StakeKeeper.GetAllValidators(ctx)
	valAddr := common.FromHex(vals[0].Signer)

	cometVal := abci.Validator{
		Address: valAddr,
		Power:   vals[0].VotingPower,
	}

	tests := []struct {
		name        string
		ctx         sdk.Context
		extVoteInfo []abci.ExtendedVoteInfo
		round       int32
		keeper      stakeKeeper.Keeper
		shouldPanic bool
		expectedErr string
	}{
		{
			name: "ves disabled with non-empty vote extension",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 0),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(cmtTypes.BlockIDFlagCommit, common.Hex2Bytes(TxHash1), common.Hex2Bytes(TxHash2), cometVal),
			},
			round:       1,
			keeper:      hApp.StakeKeeper,
			shouldPanic: true,
		},
		{
			name: "function executed correctly, but failing on signature verification",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 1),
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
				require.Panics(t, func() {
					err := ValidateVoteExtensions(tt.ctx, CurrentHeight, cometVal.Address, tt.extVoteInfo, tt.round, tt.keeper)
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
		expectError     bool
	}{
		{
			name:        "single tx approved with 2/3+1 majority",
			votingPower: 31,
			extVoteInfo: []abci.ExtendedVoteInfo{
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   10,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   20,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   1,
					}),
			},
			expectedApprove: [][]byte{common.Hex2Bytes(TxHash1)},
			expectedReject:  make([][]byte, 0, 3),
			expectedSkip:    make([][]byte, 0, 3),
			expectError:     false,
		},
		{
			name:        "one tx approved one rejected one skipped",
			votingPower: 75,
			extVoteInfo: []abci.ExtendedVoteInfo{
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1, TxHash3,
						),
						createSideTxResponses(
							sidetxs.Vote_VOTE_NO,
							TxHash2,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   40,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1,
						),
						createSideTxResponses(
							sidetxs.Vote_VOTE_NO,
							TxHash2, TxHash3,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   30,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_NO,
							TxHash1,
						),
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash2,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val3,
						Power:   5,
					}),
			},
			expectedApprove: [][]byte{common.Hex2Bytes(TxHash1)},
			expectedReject:  [][]byte{common.Hex2Bytes(TxHash2)},
			expectedSkip:    [][]byte{common.Hex2Bytes(TxHash3)},
			expectError:     false,
		},
		{
			name:        "tx approved with just enough voting power",
			votingPower: 9999,
			extVoteInfo: []abci.ExtendedVoteInfo{
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   6667,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_NO,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   3332,
					}),
			},
			expectedApprove: [][]byte{common.Hex2Bytes(TxHash1)},
			expectedReject:  make([][]byte, 0, 2),
			expectedSkip:    make([][]byte, 0, 2),
			expectError:     false,
		},
		{
			name:        "tx not rejected because almost enough voting power",
			votingPower: 9999,
			extVoteInfo: []abci.ExtendedVoteInfo{
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_NO,
						),
					),
					[]byte("signature1"),
					abci.Validator{
						Address: val1,
						Power:   6666,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
						),
					),
					[]byte("signature2"),
					abci.Validator{
						Address: val2,
						Power:   10,
					}),
			},
			expectedApprove: make([][]byte, 0, 2),
			expectedReject:  make([][]byte, 0, 2),
			expectedSkip:    make([][]byte, 0, 2),
			expectError:     false,
		},
		{
			name:        "sum of the votes exceeds the total voting power",
			votingPower: 100,
			extVoteInfo: []abci.ExtendedVoteInfo{
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   90,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_VOTE_YES,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   11,
					}),
			},
			expectedApprove: make([][]byte, 0, 2),
			expectedReject:  make([][]byte, 0, 2),
			expectedSkip:    make([][]byte, 0, 2),
			expectError:     true,
		},
		{
			name:        "tx skipped",
			votingPower: 100,
			extVoteInfo: []abci.ExtendedVoteInfo{
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_UNSPECIFIED,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val1,
						Power:   50,
					}),
				returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
					mustMarshalSideTxResponses(t,
						createSideTxResponses(
							sidetxs.Vote_UNSPECIFIED,
							TxHash1,
						),
					),
					[]byte("signature"),
					abci.Validator{
						Address: val2,
						Power:   50,
					}),
			},
			expectedApprove: make([][]byte, 0, 2),
			expectedReject:  make([][]byte, 0, 2),
			expectedSkip:    [][]byte{common.Hex2Bytes(TxHash1)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			approvedTxs, rejectedTxs, skippedTxs, err := tallyVotes(tc.extVoteInfo, log.NewTestLogger(t), tc.votingPower, CurrentHeight)
			if tc.expectError {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expectedApprove, approvedTxs)
			require.Equal(t, tc.expectedReject, rejectedTxs)
			require.Equal(t, tc.expectedSkip, skippedTxs)
		})
	}
}

func TestTallyVotesErrorDuplicateVote(t *testing.T) {
	val1, err := address.NewHexCodec().StringToBytes(ValAddr1)
	require.NoError(t, err)

	extVoteInfo := []abci.ExtendedVoteInfo{
		returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
			mustMarshalSideTxResponses(t,
				createSideTxResponses(
					sidetxs.Vote_VOTE_YES,
					TxHash1,
				),
			),
			[]byte("signature"),
			abci.Validator{
				Address: val1,
				Power:   10,
			}),
		returnExtendedVoteInfo(cmtTypes.BlockIDFlagCommit,
			mustMarshalSideTxResponses(t,
				createSideTxResponses(
					sidetxs.Vote_VOTE_NO,
					TxHash2,
				),
			),
			[]byte("signature"),
			abci.Validator{
				Address: val1,
				Power:   20,
			}),
	}

	_, _, _, err = tallyVotes(extVoteInfo, log.NewTestLogger(t), 30, CurrentHeight)
	require.Error(t, err)
	require.Equal(t, err.Error(), fmt.Sprintf("duplicate vote received from %s", util.FormatAddress(ValAddr1)))
}

func TestAggregateVotes(t *testing.T) {
	txHashBytes := common.Hex2Bytes(TxHash1)
	blockHashBytes := common.Hex2Bytes(TxHash2)

	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.ConsolidatedSideTxResponse{
		SideTxResponses: []sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
		},
		BlockHash: blockHashBytes,
		Height:    VoteExtBlockHeight,
	}

	// marshal it into Protobuf bytes
	voteExtensionBytes, err := voteExtensionProto.Marshal()
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
		sideTxResponses []sidetxs.SideTxResponse
		expectedError   bool
		expectedTxHash  []byte
	}{
		{
			name: "no duplicates",
			sideTxResponses: []sidetxs.SideTxResponse{
				{TxHash: common.Hex2Bytes(TxHash1)},
				{TxHash: common.Hex2Bytes(TxHash2)},
				{TxHash: common.Hex2Bytes(TxHash3)},
			},
			expectedError:  false,
			expectedTxHash: nil,
		},
		{
			name: "one duplicate",
			sideTxResponses: []sidetxs.SideTxResponse{
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

func TestIsBlockIDFlagValid(t *testing.T) {
	require.True(t, isBlockIdFlagValid(cmtTypes.BlockIDFlagAbsent))
	require.True(t, isBlockIdFlagValid(cmtTypes.BlockIDFlagCommit))
	require.True(t, isBlockIdFlagValid(cmtTypes.BlockIDFlagNil))
	require.False(t, isBlockIdFlagValid(cmtTypes.BlockIDFlagUnknown))
	require.False(t, isBlockIdFlagValid(100))
	require.False(t, isBlockIdFlagValid(-1))
}

func TestPanicOnVoteExtensionsDisabled(t *testing.T) {
	VoteExtEnableHeight := 1
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
		SideTxResponses: []sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
		},
		BlockHash: blockHashBytes,
		Height:    VoteExtBlockHeight,
	}

	// marshal it into Protobuf bytes
	voteExtensionBytes, _ := voteExtensionProto.Marshal()

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
