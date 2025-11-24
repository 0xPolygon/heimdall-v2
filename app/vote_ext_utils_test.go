package app

import (
	"bytes"
	"fmt"
	"testing"

	sdklog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/protoio"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestValidateVoteExtensions(t *testing.T) {
	setupAppResult := SetupApp(t, 1)
	hApp := setupAppResult.App
	validatorPrivKeys := setupAppResult.ValidatorKeys
	ctx := hApp.BaseApp.NewContext(true)
	vals := hApp.StakeKeeper.GetAllValidators(ctx)
	valAddr := common.FromHex(vals[0].Signer)

	valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(t, err)
	cometVal := abci.Validator{
		Address: valAddr,
		Power:   vals[0].VotingPower,
	}

	tests := []struct {
		name            string
		ctx             sdk.Context
		extVoteInfo     []abci.ExtendedVoteInfo
		round           int32
		valSet          stakeTypes.ValidatorSet
		milestoneKeeper milestoneKeeper.Keeper
		shouldError     bool
		expectedErr     string
	}{
		{
			name: "ves disabled with non-empty vote extension",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 0),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(t, cmtTypes.BlockIDFlagCommit, common.FromHex(TxHash1), common.FromHex(TxHash2), cometVal, validatorPrivKeys[0]),
			},
			round:       1,
			valSet:      valSet,
			shouldError: true,
		},
		{
			name: "function executed and signature verified successfully",
			ctx:  setupContextWithVoteExtensionsEnableHeight(ctx, 1),
			extVoteInfo: []abci.ExtendedVoteInfo{
				setupExtendedVoteInfo(t, cmtTypes.BlockIDFlagCommit, common.FromHex(TxHash1), common.FromHex(TxHash2), cometVal, validatorPrivKeys[0]),
			},
			round:       1,
			valSet:      valSet,
			shouldError: false,
			expectedErr: "failed to verify validator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldError {
				require.Error(t, ValidateVoteExtensions(tt.ctx, CurrentHeight, tt.extVoteInfo, tt.round, &valSet, tt.milestoneKeeper))
			} else {
				err := ValidateVoteExtensions(tt.ctx, CurrentHeight, tt.extVoteInfo, tt.round, &valSet, tt.milestoneKeeper)
				require.NoError(t, err)
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
		validatorPowers map[string]int64
		expectedApprove [][]byte
		expectedReject  [][]byte
		expectedSkip    [][]byte
		expectError     bool
	}{
		{
			name: "single tx approved with 2/3+1 majority",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 10,
				addrFromBytes(t, val2): 20,
				addrFromBytes(t, val3): 1,
			},
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
						Power:   10, // ignored by tally (canonical voting power from validators' set is used)
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
			expectedApprove: [][]byte{common.FromHex(TxHash1)},
			expectedReject:  make([][]byte, 0, 3),
			expectedSkip:    make([][]byte, 0, 3),
			expectError:     false,
		},
		{
			name: "one tx approved one rejected one skipped",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 40,
				addrFromBytes(t, val2): 30,
				addrFromBytes(t, val3): 5,
			},
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
			expectedApprove: [][]byte{common.FromHex(TxHash1)},
			expectedReject:  [][]byte{common.FromHex(TxHash2)},
			expectedSkip:    [][]byte{common.FromHex(TxHash3)},
			expectError:     false,
		},
		{
			name: "tx approved with just enough voting power",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 6667,
				addrFromBytes(t, val2): 3332,
			},
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
			expectedApprove: [][]byte{common.FromHex(TxHash1)},
			expectedReject:  make([][]byte, 0, 2),
			expectedSkip:    make([][]byte, 0, 2),
			expectError:     false,
		},
		{
			name: "tx not rejected because almost enough voting power",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 6666,
				addrFromBytes(t, val2): 10,
			},
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
			name: "forged_validator_power_in_extcommit_is_ignored",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 90, // canonical power for Val1
				addrFromBytes(t, val2): 11, // canonical power for Val2
			},
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
						Power:   9000, // forged: should be ignored
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
						Power:   11000, // forged: should be ignored
					}),
			},
			expectedApprove: [][]byte{common.FromHex(TxHash1)},
			expectedReject:  make([][]byte, 0, 2),
			expectedSkip:    make([][]byte, 0, 2),
			expectError:     false,
		},
		{
			name: "tx skipped",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 50,
				addrFromBytes(t, val2): 50,
			},
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
			expectedSkip:    [][]byte{common.FromHex(TxHash1)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			validatorSet := buildValidatorSet(t, tc.validatorPowers)

			approvedTxs, rejectedTxs, skippedTxs, err := tallyVotes(
				tc.extVoteInfo,
				sdklog.NewTestLogger(t),
				validatorSet,
				CurrentHeight,
			)

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
				Power:   10, // ignored in tally (canonical voting power from validators' set is used)
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
				Power:   20, // ignored in tally (canonical voting power from validators' set is used)
			}),
	}

	// canonical validator set: the power value is irrelevant for this test
	validatorSet := buildValidatorSet(t, map[string]int64{
		addrFromBytes(t, val1): 30,
	})

	_, _, _, err = tallyVotes(
		extVoteInfo,
		sdklog.NewTestLogger(t),
		validatorSet,
		CurrentHeight,
	)
	require.Error(t, err)
	require.Equal(t, err.Error(), fmt.Sprintf("duplicate vote received from %s", util.FormatAddress(ValAddr1)))
}

func TestAggregateVotes(t *testing.T) {
	txHashBytes := common.FromHex(TxHash1)
	blockHashBytes := common.FromHex(TxHash2)

	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.VoteExtension{
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
				Power:   10, // ignored for tally: canonical valSet defines the voting power
			},
			VoteExtension:      voteExtensionBytes,
			ExtensionSignature: []byte("signature"),
			BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
		},
	}

	expectedVotes := map[string]map[sidetxs.Vote]int64{
		TxHash1: {
			sidetxs.Vote_VOTE_YES: 10, // canonical power for ValAddr1
		},
	}

	// canonical validator set: ValAddr1 has power 10
	validatorSet := buildValidatorSet(t, map[string]int64{
		addrFromBytes(t, val1): 10,
	})

	actualVotes, err := aggregateVotes(
		extVoteInfo,
		validatorSet,
		CurrentHeight,
		sdklog.NewTestLogger(t),
	)
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
				{TxHash: common.FromHex(TxHash1)},
				{TxHash: common.FromHex(TxHash2)},
				{TxHash: common.FromHex(TxHash3)},
			},
			expectedError:  false,
			expectedTxHash: nil,
		},
		{
			name: "one duplicate",
			sideTxResponses: []sidetxs.SideTxResponse{
				{TxHash: common.FromHex(TxHash1)},
				{TxHash: common.FromHex(TxHash2)},
				{TxHash: common.FromHex(TxHash3)},
				{TxHash: common.FromHex(TxHash3)},
			},
			expectedError:  true,
			expectedTxHash: common.FromHex(TxHash3),
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

func TestCheckIfVoteExtensionsDisabled(t *testing.T) {
	VoteExtEnableHeight := 1
	key := storetypes.NewKVStoreKey("testStoreKey")
	testCtx := cosmostestutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := setupContextWithVoteExtensionsEnableHeight(testCtx.Ctx, int64(VoteExtEnableHeight))

	tests := []struct {
		name   string
		height int64
		errors bool
	}{
		{"height is less than VoteExtensionsEnableHeight", int64(VoteExtEnableHeight) - 1, true},
		{"height is equal to VoteExtensionsEnableHeight", int64(VoteExtEnableHeight), false},
		{"height is greater than VoteExtensionsEnableHeight", int64(VoteExtEnableHeight) + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.errors {
				require.NoError(t,
					checkIfVoteExtensionsDisabled(ctx, tt.height),
					"checkIfVoteExtensionsDisabled returned error unexpectedly")
			} else {
				require.Error(t,
					checkIfVoteExtensionsDisabled(ctx, tt.height),
					"checkIfVoteExtensionsDisabled did not returned error, but it should have")
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

func setupExtendedVoteInfo(t *testing.T, flag cmtTypes.BlockIDFlag, txHashBytes, blockHashBytes []byte, validator abci.Validator, privKey cmtcrypto.PrivKey) abci.ExtendedVoteInfo {
	t.Helper()
	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.VoteExtension{
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
	require.NoErrorf(t, err, "failed to marshal voteExtensionProto: %v", err)

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
	require.NoErrorf(t, err, "failed to encode CanonicalVoteExtension: %v", err)

	// Sign the vote extension
	signature, err := privKey.Sign(extSignBytes)
	require.NoErrorf(t, err, "failed to sign extSignBytes: %v", err)

	return abci.ExtendedVoteInfo{
		BlockIdFlag:             flag,
		VoteExtension:           voteExtensionBytes,
		ExtensionSignature:      signature,
		Validator:               validator,
		NonRpVoteExtension:      []byte("\t\r\n#HEIMDALL-VOTE-EXTENSION#\r\n\t"),
		NonRpExtensionSignature: signature,
	}
}

func setupExtendedVoteInfoWithNonRp(t *testing.T, flag cmtTypes.BlockIDFlag, txHashBytes, blockHashBytes []byte, validator abci.Validator, privKey cmtcrypto.PrivKey, height int64, app *HeimdallApp, cmtPubKey cmtcrypto.PubKey) abci.ExtendedVoteInfo {
	t.Helper()

	dummyExt, err := GetDummyNonRpVoteExtension(height, app.ChainID())
	if err != nil {
		panic(err)
	}
	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.VoteExtension{
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
	require.NoErrorf(t, err, "failed to marshal voteExtensionProto: %v", err)

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
	require.NoErrorf(t, err, "failed to encode CanonicalVoteExtension: %v", err)

	// Sign the vote extension
	signature, err := privKey.Sign(extSignBytes)
	require.NoErrorf(t, err, "failed to sign extSignBytes: %v", err)

	// Sign nonRpVE
	signatureNonRpVE, err := privKey.Sign(dummyExt)
	ok := cmtPubKey.VerifySignature(dummyExt, signatureNonRpVE)
	if !ok {
		fmt.Println(" Error : Signature verification failed!")
	}

	return abci.ExtendedVoteInfo{
		BlockIdFlag:             flag,
		VoteExtension:           voteExtensionBytes,
		ExtensionSignature:      signature,
		Validator:               validator,
		NonRpVoteExtension:      dummyExt,
		NonRpExtensionSignature: signatureNonRpVE,
	}
}

func setupExtendedVoteInfoWithMilestoneProposition(t *testing.T, flag cmtTypes.BlockIDFlag, txHashBytes, blockHashBytes []byte, validator abci.Validator, privKey cmtcrypto.PrivKey, height int64, app *HeimdallApp, cmtPubKey cmtcrypto.PubKey, milestoneProposition milestoneTypes.MilestoneProposition) abci.ExtendedVoteInfo {
	t.Helper()

	dummyExt, err := GetDummyNonRpVoteExtension(height, app.ChainID())
	if err != nil {
		panic(err)
	}
	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.VoteExtension{
		SideTxResponses: []sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
		},
		BlockHash:            blockHashBytes,
		Height:               VoteExtBlockHeight,
		MilestoneProposition: &milestoneProposition,
	}

	// marshal it into Protobuf bytes
	voteExtensionBytes, err := voteExtensionProto.Marshal()
	require.NoErrorf(t, err, "failed to marshal voteExtensionProto: %v", err)

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
	require.NoErrorf(t, err, "failed to encode CanonicalVoteExtension: %v", err)

	// Sign the vote extension
	signature, err := privKey.Sign(extSignBytes)
	require.NoErrorf(t, err, "failed to sign extSignBytes: %v", err)

	// Sign nonRpVE
	signatureNonRpVE, err := privKey.Sign(dummyExt)
	ok := cmtPubKey.VerifySignature(dummyExt, signatureNonRpVE)
	if !ok {
		fmt.Println(" Error : Signature verification failed!")
	}

	return abci.ExtendedVoteInfo{
		BlockIdFlag:             flag,
		VoteExtension:           voteExtensionBytes,
		ExtensionSignature:      signature,
		Validator:               validator,
		NonRpVoteExtension:      dummyExt,
		NonRpExtensionSignature: signatureNonRpVE,
	}
}

// buildValidatorSet is a helper method to create a validators' set for tests
func buildValidatorSet(t *testing.T, addrToPower map[string]int64) *stakeTypes.ValidatorSet {
	t.Helper()

	validators := make([]*stakeTypes.Validator, 0, len(addrToPower))
	for addr, power := range addrToPower {
		validators = append(validators, &stakeTypes.Validator{
			Signer:      addr,
			VotingPower: power,
		})
	}

	return &stakeTypes.ValidatorSet{
		Validators: validators,
	}
}

// addrFromBytes converts a byte slice representation of an address into its string format using HexCodec.
// An error in the conversion process will cause the test to fail.
func addrFromBytes(t *testing.T, b []byte) string {
	t.Helper()
	s, err := address.NewHexCodec().BytesToString(b)
	require.NoError(t, err)
	return s
}
