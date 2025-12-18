package app

import (
	"bytes"
	"encoding/binary"
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

func TestRejectUnknownVoteExtensionFields(t *testing.T) {
	ve := sidetxs.VoteExtension{
		Height: VoteExtBlockHeight,
		SideTxResponses: []sidetxs.SideTxResponse{
			{
				TxHash: common.FromHex(TxHash1),
				Result: sidetxs.Vote_VOTE_YES,
			},
		},
	}

	cleanBytes, err := ve.Marshal()
	require.NoError(t, err)

	// clean VE must pass
	require.NoError(t, rejectUnknownVoteExtFields(cleanBytes))

	// pad with unknown protobuf field
	padded := appendProtobufPadding(cleanBytes, 32*1024)

	// this must be rejected
	err = rejectUnknownVoteExtFields(padded)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown", "error should mention unknown fields")
}

func TestValidateVoteExtensions_RejectsPaddedVoteExtension(t *testing.T) {
	setupAppResult := SetupApp(t, 1)
	hApp := setupAppResult.App
	validatorPrivKeys := setupAppResult.ValidatorKeys

	ctx := hApp.BaseApp.NewContext(true)
	ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)

	vals := hApp.StakeKeeper.GetAllValidators(ctx)
	valAddr := common.FromHex(vals[0].Signer)

	valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(t, err)

	cometVal := abci.Validator{
		Address: valAddr,
		Power:   vals[0].VotingPower,
	}

	ext := setupExtendedVoteInfo(
		t,
		cmtTypes.BlockIDFlagCommit,
		common.FromHex(TxHash1),
		common.FromHex(TxHash2),
		cometVal,
		validatorPrivKeys[0],
	)

	// padding with unknown protobuf field
	ext.VoteExtension = appendProtobufPadding(ext.VoteExtension, 64*1024)

	err = ValidateVoteExtensions(
		ctx,
		CurrentHeight,
		[]abci.ExtendedVoteInfo{ext},
		1,
		&valSet,
		hApp.MilestoneKeeper,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown fields detected")
}

func TestFilterVoteExtensions_SkipsPaddedVoteExtension(t *testing.T) {
	setupAppResult := SetupApp(t, 4)
	hApp := setupAppResult.App
	validatorPrivKeys := setupAppResult.ValidatorKeys

	ctx := hApp.BaseApp.NewContext(true)
	ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)

	vals := hApp.StakeKeeper.GetAllValidators(ctx)
	require.GreaterOrEqual(t, len(vals), 4)

	valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(t, err)

	reqHeight := int64(3)
	round := int32(1)

	privByAddr := make(map[string]cmtcrypto.PrivKey, len(validatorPrivKeys))
	for _, pk := range validatorPrivKeys {
		addr := pk.PubKey().Address() // []byte
		privByAddr[common.Bytes2Hex(addr)] = pk
	}

	// sign CanonicalVoteExtension
	signVE := func(priv cmtcrypto.PrivKey, extension []byte) []byte {
		cve := cmtTypes.CanonicalVoteExtension{
			Extension: extension,
			Height:    reqHeight - 1,
			Round:     int64(round),
			ChainId:   ctx.ChainID(),
		}

		var buf bytes.Buffer
		_, err := protoio.NewDelimitedWriter(&buf).WriteMsg(&cve)
		require.NoError(t, err)

		sig, err := priv.Sign(buf.Bytes())
		require.NoError(t, err)
		require.NotEmpty(t, sig)
		return sig
	}

	// Prepare a valid VoteExtension payload
	voteExtensionProto := sidetxs.VoteExtension{
		SideTxResponses: []sidetxs.SideTxResponse{
			{TxHash: common.FromHex(TxHash1), Result: sidetxs.Vote_VOTE_YES},
		},
		BlockHash: common.FromHex(TxHash2),
		Height:    reqHeight - 1,
	}
	veBytes, err := voteExtensionProto.Marshal()
	require.NoError(t, err)

	// Pick 3 honest validators + 1 malicious validator
	type picked struct {
		addrBytes []byte
		power     int64
		priv      cmtcrypto.PrivKey
	}
	picks := make([]picked, 0, 4)

	for _, v := range vals {
		addrBytes := common.FromHex(v.Signer)
		addrHex := common.Bytes2Hex(addrBytes)

		priv, ok := privByAddr[addrHex]
		if !ok {
			continue
		}
		picks = append(picks, picked{
			addrBytes: addrBytes,
			power:     v.VotingPower,
			priv:      priv,
		})
		if len(picks) == 4 {
			break
		}
	}
	require.Len(t, picks, 4, "could not match 4 validators to privKeys; validator ordering may differ or address formats differ")

	// Build a vote
	mkVote := func(p picked, ext []byte, sig []byte) abci.ExtendedVoteInfo {
		return abci.ExtendedVoteInfo{
			BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
			VoteExtension:      ext,
			ExtensionSignature: sig,
			Validator: abci.Validator{
				Address: p.addrBytes,
				Power:   p.power,
			},
		}
	}

	// 3 clean votes
	clean0 := mkVote(picks[0], veBytes, signVE(picks[0].priv, veBytes))
	clean1 := mkVote(picks[1], veBytes, signVE(picks[1].priv, veBytes))
	clean2 := mkVote(picks[2], veBytes, signVE(picks[2].priv, veBytes))

	// 1 padded vote (unknown field)
	paddedBytes := appendProtobufPadding(veBytes, 64*1024)
	padded := mkVote(picks[3], paddedBytes, signVE(picks[3].priv, paddedBytes))

	extVoteInfo := []abci.ExtendedVoteInfo{clean0, clean1, clean2, padded}

	filtered, err := FilterVoteExtensions(
		ctx,
		reqHeight,
		extVoteInfo,
		round,
		&valSet,
		hApp.MilestoneKeeper,
		sdklog.NewTestLogger(t),
	)
	require.NoError(t, err)

	// padded must be skipped
	require.Len(t, filtered, 3)
	for _, got := range filtered {
		require.Equal(t, veBytes, got.VoteExtension)
		// ensure the padded validator is not present
		require.NotEqual(t, picks[3].addrBytes, got.Validator.Address)
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
// The map keys should be hex-encoded addresses (from addrFromBytes)
func buildValidatorSet(t *testing.T, addrToPower map[string]int64) *stakeTypes.ValidatorSet {
	t.Helper()

	validators := make([]*stakeTypes.Validator, 0, len(addrToPower))

	// Convert hex string addresses back to the format expected by ValidatorSet
	ac := address.NewHexCodec()

	for addrStr, power := range addrToPower {
		// Verify the address format is the correct hex string
		addrBytes, err := ac.StringToBytes(addrStr)
		require.NoError(t, err, "invalid address format in test setup: %s", addrStr)

		// Convert back to string to ensure consistency
		normalizedAddr, err := ac.BytesToString(addrBytes)
		require.NoError(t, err, "failed to normalize address")

		validators = append(validators, &stakeTypes.Validator{
			Signer:      normalizedAddr,
			VotingPower: power,
		})
	}

	return stakeTypes.NewValidatorSet(validators)
}

// addrFromBytes converts a byte slice representation of an address into its string format using HexCodec.
// An error in the conversion process will cause the test to fail.
func addrFromBytes(t *testing.T, b []byte) string {
	t.Helper()
	s, err := address.NewHexCodec().BytesToString(b)
	require.NoError(t, err)
	return s
}

// appendProtobufPadding appends an unknown protobuf field with arbitrary payload.
func appendProtobufPadding(data []byte, paddingSize int) []byte {
	tag := byte(58)

	var lenBuf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(lenBuf[:], uint64(paddingSize))

	out := make([]byte, len(data)+1+n+paddingSize)
	copy(out, data)

	offset := len(data)
	out[offset] = tag
	offset++

	copy(out[offset:], lenBuf[:n])
	offset += n

	for i := 0; i < paddingSize; i++ {
		out[offset+i] = byte(i)
	}

	return out
}
