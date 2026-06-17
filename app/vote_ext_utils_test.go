package app

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
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
	"github.com/0xPolygon/heimdall-v2/helper"
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
			}
			require.NoError(t, err)
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

func TestGetMajorityNonRpVoteExtension(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})

	val1, err := address.NewHexCodec().StringToBytes(ValAddr1)
	require.NoError(t, err)
	val2, err := address.NewHexCodec().StringToBytes(ValAddr2)
	require.NoError(t, err)
	val3, err := address.NewHexCodec().StringToBytes(ValAddr3)
	require.NoError(t, err)

	dummyExt1 := []byte("dummy_extension_1")
	dummyExt2 := []byte("dummy_extension_2")

	tests := []struct {
		name            string
		extVoteInfo     []abci.ExtendedVoteInfo
		validatorPowers map[string]int64
		expectError     bool
		errorContains   string
	}{
		{
			name: "extension with >2/3 voting power succeeds",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 70,
				addrFromBytes(t, val2): 20,
				addrFromBytes(t, val3): 10,
			},
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator:          abci.Validator{Address: val1, Power: 70},
					NonRpVoteExtension: dummyExt1,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val2, Power: 20},
					NonRpVoteExtension: dummyExt1,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val3, Power: 10},
					NonRpVoteExtension: dummyExt2,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
			},
			expectError: false,
		},
		{
			name: "extension with exactly 2/3 voting power fails (needs >2/3)",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 66,
				addrFromBytes(t, val2): 33,
				addrFromBytes(t, val3): 1,
			},
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator:          abci.Validator{Address: val1, Power: 66},
					NonRpVoteExtension: dummyExt1,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val2, Power: 33},
					NonRpVoteExtension: dummyExt2,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val3, Power: 1},
					NonRpVoteExtension: dummyExt2,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
			},
			expectError:   true,
			errorContains: "insufficient voting power",
		},
		{
			name: "extension with >50% but <67% voting power fails (replay attack prevented)",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 60,
				addrFromBytes(t, val2): 30,
				addrFromBytes(t, val3): 10,
			},
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator:          abci.Validator{Address: val1, Power: 60},
					NonRpVoteExtension: dummyExt1,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val2, Power: 30},
					NonRpVoteExtension: dummyExt2,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val3, Power: 10},
					NonRpVoteExtension: dummyExt2,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
			},
			expectError:   true,
			errorContains: "insufficient voting power",
		},
		{
			name: "all validators agree on same extension (100% voting power)",
			validatorPowers: map[string]int64{
				addrFromBytes(t, val1): 40,
				addrFromBytes(t, val2): 30,
				addrFromBytes(t, val3): 30,
			},
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator:          abci.Validator{Address: val1, Power: 40},
					NonRpVoteExtension: dummyExt1,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val2, Power: 30},
					NonRpVoteExtension: dummyExt1,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:          abci.Validator{Address: val3, Power: 30},
					NonRpVoteExtension: dummyExt1,
					BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				},
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			validatorSet := buildValidatorSet(t, tc.validatorPowers)
			ctx := setupContextWithVoteExtensionsEnableHeight(cosmostestutil.DefaultContext(storetypes.NewKVStoreKey("test"), storetypes.NewTransientStoreKey("transient_test")), 1)
			ctx = ctx.WithBlockHeight(1)

			result, err := getMajorityNonRpVoteExtension(ctx, tc.extVoteInfo, validatorSet, sdklog.NewTestLogger(t))

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

func TestGetCheckpointSignatures(t *testing.T) {
	helper.SetZurichHardforkHeight(10)
	t.Cleanup(func() {
		helper.SetZurichHardforkHeight(0)
	})

	val1, err := address.NewHexCodec().StringToBytes(ValAddr1)
	require.NoError(t, err)
	val2, err := address.NewHexCodec().StringToBytes(ValAddr2)
	require.NoError(t, err)
	val3, err := address.NewHexCodec().StringToBytes(ValAddr3)
	require.NoError(t, err)
	injectedAddr := bytes.Repeat([]byte{0xAB}, common.AddressLength)

	majorityExt := []byte("majority_extension")
	otherExt := []byte("other_extension")
	sig1 := []byte("sig-1")
	sig2 := []byte("sig-2")
	sig3 := []byte("sig-3")
	poisonedSig := []byte{0x01, 0x02, 0x03}

	twoCommitsAndInjectedNonCommit := []abci.ExtendedVoteInfo{
		{
			Validator:               abci.Validator{Address: val1},
			NonRpVoteExtension:      majorityExt,
			NonRpExtensionSignature: sig1,
			BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
		},
		{
			Validator:               abci.Validator{Address: val2},
			NonRpVoteExtension:      majorityExt,
			NonRpExtensionSignature: sig2,
			BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
		},
		{
			Validator:               abci.Validator{Address: injectedAddr},
			NonRpVoteExtension:      majorityExt,
			NonRpExtensionSignature: poisonedSig,
			BlockIdFlag:             cmtTypes.BlockIDFlagAbsent,
		},
	}

	const preHardforkHeight = int64(1)
	const postHardforkHeight = int64(11)

	tests := []struct {
		name              string
		height            int64
		extVoteInfo       []abci.ExtendedVoteInfo
		expectedAddrToSig map[string][]byte
		mustExcludeAddrs  [][]byte
	}{
		{
			name:   "post-hardfork: all commit votes matching extension are included",
			height: postHardforkHeight,
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator:               abci.Validator{Address: val1},
					NonRpVoteExtension:      majorityExt,
					NonRpExtensionSignature: sig1,
					BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:               abci.Validator{Address: val2},
					NonRpVoteExtension:      majorityExt,
					NonRpExtensionSignature: sig2,
					BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
				},
			},
			expectedAddrToSig: map[string][]byte{
				string(val1): sig1,
				string(val2): sig2,
			},
		},
		{
			name:   "post-hardfork: commit votes with non-matching extension are skipped",
			height: postHardforkHeight,
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator:               abci.Validator{Address: val1},
					NonRpVoteExtension:      majorityExt,
					NonRpExtensionSignature: sig1,
					BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:               abci.Validator{Address: val3},
					NonRpVoteExtension:      otherExt,
					NonRpExtensionSignature: sig3,
					BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
				},
			},
			expectedAddrToSig: map[string][]byte{
				string(val1): sig1,
			},
			mustExcludeAddrs: [][]byte{val3},
		},
		{
			name:        "post-hardfork: injected non-commit vote with matching extension is excluded",
			height:      postHardforkHeight,
			extVoteInfo: twoCommitsAndInjectedNonCommit,
			expectedAddrToSig: map[string][]byte{
				string(val1): sig1,
				string(val2): sig2,
			},
			mustExcludeAddrs: [][]byte{injectedAddr},
		},
		{
			name:   "post-hardfork: BlockIDFlagNil with matching extension is excluded",
			height: postHardforkHeight,
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator:               abci.Validator{Address: val1},
					NonRpVoteExtension:      majorityExt,
					NonRpExtensionSignature: sig1,
					BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
				},
				{
					Validator:               abci.Validator{Address: injectedAddr},
					NonRpVoteExtension:      majorityExt,
					NonRpExtensionSignature: poisonedSig,
					BlockIdFlag:             cmtTypes.BlockIDFlagNil,
				},
			},
			expectedAddrToSig: map[string][]byte{
				string(val1): sig1,
			},
			mustExcludeAddrs: [][]byte{injectedAddr},
		},
		{
			name:        "pre-hardfork: legacy behavior preserved, injected non-commit still included",
			height:      preHardforkHeight,
			extVoteInfo: twoCommitsAndInjectedNonCommit,
			expectedAddrToSig: map[string][]byte{
				string(val1):         sig1,
				string(val2):         sig2,
				string(injectedAddr): poisonedSig,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getCheckpointSignatures(tc.height, majorityExt, tc.extVoteInfo)

			require.Len(t, got.Signatures, len(tc.expectedAddrToSig))
			gotByAddr := make(map[string][]byte, len(got.Signatures))
			for _, sig := range got.Signatures {
				gotByAddr[string(sig.ValidatorAddress)] = sig.Signature
			}
			for addr, expectedSig := range tc.expectedAddrToSig {
				require.Equal(t, expectedSig, gotByAddr[addr], "signature mismatch for %x", addr)
			}
			for _, addr := range tc.mustExcludeAddrs {
				_, present := gotByAddr[string(addr)]
				require.False(t, present, "address %x must not appear in result", addr)
			}
		})
	}
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
	// Add multiple side tx responses to meet the minimum vote extension size of 10 bytes
	voteExtensionProto := sidetxs.VoteExtension{
		SideTxResponses: []sidetxs.SideTxResponse{
			{TxHash: common.FromHex(TxHash1), Result: sidetxs.Vote_VOTE_YES},
			{TxHash: make([]byte, 32), Result: sidetxs.Vote_VOTE_YES},
			{TxHash: append([]byte{0x01}, common.FromHex(TxHash1)[1:]...), Result: sidetxs.Vote_VOTE_YES},
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
	dummyNonRpVE, err := GetDummyNonRpVoteExtension(reqHeight-1, ctx.ChainID())
	require.NoError(t, err)

	mkVote := func(p picked, ext []byte, sig []byte) abci.ExtendedVoteInfo {
		nonRpSig, err := p.priv.Sign(dummyNonRpVE)
		require.NoError(t, err)
		return abci.ExtendedVoteInfo{
			BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
			VoteExtension:           ext,
			ExtensionSignature:      sig,
			NonRpVoteExtension:      dummyNonRpVE,
			NonRpExtensionSignature: nonRpSig,
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

	filtered, err := filterVoteExtensions(
		ctx,
		reqHeight,
		extVoteInfo,
		round,
		&valSet,
		hApp.MilestoneKeeper,
		1*1024*1024, // 1MB MaxTxBytes
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

// TestfilterVoteExtensions_SizeFiltering tests the size-based filtering logic
func TestFilterVoteExtensions_SizeFiltering(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})

	t.Run("filters out vote extensions exceeding per-validator limit", func(t *testing.T) {
		setupAppResult := SetupApp(t, 4)
		hApp := setupAppResult.App
		validatorPrivKeys := setupAppResult.ValidatorKeys
		ctx := hApp.BaseApp.NewContext(true)
		ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
		validators := hApp.StakeKeeper.GetAllValidators(ctx)

		valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
		require.NoError(t, err)

		maxTxBytes := int64(1048576) // 1MB

		// Create vote extensions of different sizes
		var votes []abci.ExtendedVoteInfo

		dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
		require.NoError(t, err)

		reqHeight := int64(3)
		round := int32(1)

		// All validators with normal-sized VEs (~2KB - well within the 10KB limit)
		normalVE := createVoteExtensionOfSize(2000)

		for i := 0; i < len(validators); i++ {
			privKey := findPrivKeyForValidator(validators[i], validatorPrivKeys)
			require.NotNil(t, privKey)
			vote := createSignedVoteInfo(t, ctx, validators[i], privKey, normalVE, dummyNonRpVE, reqHeight, round)
			votes = append(votes, vote)
		}

		// Filter vote extensions
		filtered, err := filterVoteExtensions(ctx, 3, votes, 1, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
		require.NoError(t, err)

		// All VEs should pass since they're well within the 10KB limit
		require.Equal(t, len(validators), len(filtered), "All VEs within limits should pass")
	})

	t.Run("filters out oversized non-rp vote extensions", func(t *testing.T) {
		setupAppResult := SetupApp(t, 4)
		hApp := setupAppResult.App
		validatorPrivKeys := setupAppResult.ValidatorKeys
		ctx := hApp.BaseApp.NewContext(true)
		ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
		validators := hApp.StakeKeeper.GetAllValidators(ctx)

		valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
		require.NoError(t, err)

		maxTxBytes := int64(1048576) // 1MB

		normalVE := createVoteExtensionOfSize(1024)

		dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
		require.NoError(t, err)

		// Oversized NonRpVE (600 bytes - exceeds maxNonRpVoteExtensionSize of 500)
		oversizedNonRpVE := make([]byte, 600)
		for i := range oversizedNonRpVE {
			oversizedNonRpVE[i] = byte(i)
		}

		reqHeight := int64(3)
		round := int32(1)
		var votes []abci.ExtendedVoteInfo

		// Validator 0: Normal NonRpVE (should pass)
		privKey0 := findPrivKeyForValidator(validators[0], validatorPrivKeys)
		require.NotNil(t, privKey0)
		vote0 := createSignedVoteInfo(t, ctx, validators[0], privKey0, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote0)

		// Validator 1: Oversized NonRpVE (600 bytes - exceeds maxNonRpVoteExtensionSize of 500)
		privKey1 := findPrivKeyForValidator(validators[1], validatorPrivKeys)
		require.NotNil(t, privKey1)
		vote1 := createSignedVoteInfo(t, ctx, validators[1], privKey1, normalVE, oversizedNonRpVE, reqHeight, round)
		votes = append(votes, vote1)

		// Validator 2: Normal NonRpVE (should pass) - need majority voting power
		privKey2 := findPrivKeyForValidator(validators[2], validatorPrivKeys)
		require.NotNil(t, privKey2)
		vote2 := createSignedVoteInfo(t, ctx, validators[2], privKey2, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote2)

		// Validator 3: Normal NonRpVE (should pass) - need majority voting power
		privKey3 := findPrivKeyForValidator(validators[3], validatorPrivKeys)
		require.NotNil(t, privKey3)
		vote3 := createSignedVoteInfo(t, ctx, validators[3], privKey3, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote3)

		// Filter vote extensions
		filtered, err := filterVoteExtensions(ctx, 3, votes, 1, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
		require.NoError(t, err)

		// Validator 1 has oversized NonRpVE: should be a placeholder, not dropped.
		// All 4 entries are preserved; 1 is a filtered placeholder.
		require.Equal(t, 4, len(filtered), "All entries should be preserved (1 as placeholder)")

		// Verify the placeholder entry has nil extension fields
		placeholderFound := false
		for _, v := range filtered {
			if isFilteredPlaceholder(v) {
				placeholderFound = true
			}
		}
		require.True(t, placeholderFound, "Should have at least one filtered placeholder")
	})

	t.Run("filtering works with different validator counts", func(t *testing.T) {
		testCases := []struct {
			name           string
			validatorCount int
		}{
			{name: "4 validators", validatorCount: 4},
			{name: "10 validators", validatorCount: 10},
			{name: "20 validators", validatorCount: 20},
			{name: "100 validators", validatorCount: 100},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				setupAppResult := SetupApp(t, uint64(tc.validatorCount))
				hApp := setupAppResult.App
				validatorPrivKeys := setupAppResult.ValidatorKeys
				ctx := hApp.BaseApp.NewContext(true)
				ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
				validators := hApp.StakeKeeper.GetAllValidators(ctx)

				valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
				require.NoError(t, err)

				maxTxBytes := int64(1048576) // 1MB

				dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
				require.NoError(t, err)

				reqHeight := int64(3)
				round := int32(1)
				var votes []abci.ExtendedVoteInfo

				// Create normal VEs (~2KB) for all validators - all should pass
				normalVE := createVoteExtensionOfSize(2000)
				for i := 0; i < len(validators); i++ {
					privKey := findPrivKeyForValidator(validators[i], validatorPrivKeys)
					require.NotNil(t, privKey)
					vote := createSignedVoteInfo(t, ctx, validators[i], privKey, normalVE, dummyNonRpVE, reqHeight, round)
					votes = append(votes, vote)
				}

				// Filter
				filtered, err := filterVoteExtensions(ctx, 3, votes, 1, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
				require.NoError(t, err)

				// All VEs should pass since they're within limits
				require.Equal(t, len(validators), len(filtered), "All VEs should pass for %s", tc.name)
			})
		}
	})

	t.Run("allows all VEs when all are within limits", func(t *testing.T) {
		setupAppResult := SetupApp(t, 4)
		hApp := setupAppResult.App
		validatorPrivKeys := setupAppResult.ValidatorKeys
		ctx := hApp.BaseApp.NewContext(true)
		ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
		validators := hApp.StakeKeeper.GetAllValidators(ctx)

		valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
		require.NoError(t, err)

		maxTxBytes := int64(1048576)

		dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
		require.NoError(t, err)

		reqHeight := int64(3)
		round := int32(1)
		var votes []abci.ExtendedVoteInfo

		// All validators with normal-sized VEs
		for i := 0; i < len(validators); i++ {
			normalVE := createVoteExtensionOfSize(2048) // 2KB
			privKey := findPrivKeyForValidator(validators[i], validatorPrivKeys)
			require.NotNil(t, privKey, "Private key not found for validator %s", validators[i].Signer)
			vote := createSignedVoteInfo(t, ctx, validators[i], privKey, normalVE, dummyNonRpVE, reqHeight, round)
			votes = append(votes, vote)
		}

		filtered, err := filterVoteExtensions(ctx, 3, votes, 1, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
		require.NoError(t, err)

		// All VEs should pass
		require.Equal(t, len(validators), len(filtered), "All VEs within limits should pass")
	})

	t.Run("filtering works with small MaxTxBytes", func(t *testing.T) {
		setupAppResult := SetupApp(t, 4)
		hApp := setupAppResult.App
		validatorPrivKeys := setupAppResult.ValidatorKeys
		ctx := hApp.BaseApp.NewContext(true)
		ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
		validators := hApp.StakeKeeper.GetAllValidators(ctx)

		valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
		require.NoError(t, err)

		// Use small maxTxBytes - filtering still applies based on the calculated per-validator limit
		// With 24KB and 4 validators: (24000/4/3)-700 = 1300 bytes per validator
		maxTxBytes := int64(24000)

		dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
		require.NoError(t, err)

		reqHeight := int64(3)
		round := int32(1)
		var votes []abci.ExtendedVoteInfo

		// Use VEs small enough to fit within the calculated 1300-byte limit
		for i := 0; i < len(validators); i++ {
			smallVE := createVoteExtensionOfSize(1000) // 1KB - fits within the 1300-byte limit
			privKey := findPrivKeyForValidator(validators[i], validatorPrivKeys)
			require.NotNil(t, privKey, "Private key not found for validator %s", validators[i].Signer)
			vote := createSignedVoteInfo(t, ctx, validators[i], privKey, smallVE, dummyNonRpVE, reqHeight, round)
			votes = append(votes, vote)
		}

		filtered, err := filterVoteExtensions(ctx, 3, votes, 1, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
		// Filtering applies and VEs within the calculated limit should pass
		require.NoError(t, err)
		require.Equal(t, len(validators), len(filtered), "All VEs within calculated limit should pass")
	})

	t.Run("filters out undersized vote extensions", func(t *testing.T) {
		setupAppResult := SetupApp(t, 4)
		hApp := setupAppResult.App
		validatorPrivKeys := setupAppResult.ValidatorKeys
		ctx := hApp.BaseApp.NewContext(true)
		ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
		validators := hApp.StakeKeeper.GetAllValidators(ctx)

		valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
		require.NoError(t, err)

		maxTxBytes := int64(1048576) // 1MB

		dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
		require.NoError(t, err)

		reqHeight := int64(3)
		round := int32(1)
		var votes []abci.ExtendedVoteInfo

		normalVE := createVoteExtensionOfSize(2000)

		// Validator 0: Normal VE (should pass)
		privKey0 := findPrivKeyForValidator(validators[0], validatorPrivKeys)
		require.NotNil(t, privKey0)
		vote0 := createSignedVoteInfo(t, ctx, validators[0], privKey0, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote0)

		// Validator 1: Undersized VE (8 bytes - below minVESize of 10 bytes)
		// Create raw undersized bytes instead of using createVoteExtensionOfSize
		privKey1 := findPrivKeyForValidator(validators[1], validatorPrivKeys)
		require.NotNil(t, privKey1)
		undersizedVEBytes := make([]byte, 8)
		for i := range undersizedVEBytes {
			undersizedVEBytes[i] = byte(i)
		}

		// Create the vote manually with undersized bytes
		cometVal1 := abci.Validator{
			Address: common.FromHex(validators[1].Signer),
			Power:   validators[1].VotingPower,
		}
		vote1 := abci.ExtendedVoteInfo{
			BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
			VoteExtension:           undersizedVEBytes,
			ExtensionSignature:      []byte("dummy_signature"), // Signature check happens after the size check
			Validator:               cometVal1,
			NonRpVoteExtension:      dummyNonRpVE,
			NonRpExtensionSignature: []byte("dummy_nonrp_signature"),
		}
		votes = append(votes, vote1)

		// Validator 2: Normal VE (should pass) - need majority voting power
		privKey2 := findPrivKeyForValidator(validators[2], validatorPrivKeys)
		require.NotNil(t, privKey2)
		vote2 := createSignedVoteInfo(t, ctx, validators[2], privKey2, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote2)

		// Validator 3: Normal VE (should pass) - need majority voting power
		privKey3 := findPrivKeyForValidator(validators[3], validatorPrivKeys)
		require.NotNil(t, privKey3)
		vote3 := createSignedVoteInfo(t, ctx, validators[3], privKey3, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote3)

		// Filter vote extensions
		filtered, err := filterVoteExtensions(ctx, 3, votes, 1, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
		require.NoError(t, err)

		// Validator 1 has undersized VE: should be a placeholder, not dropped.
		require.Equal(t, 4, len(filtered), "All entries should be preserved (1 as placeholder)")
		placeholderFound := false
		for _, v := range filtered {
			if isFilteredPlaceholder(v) {
				placeholderFound = true
			}
		}
		require.True(t, placeholderFound, "Should have at least one filtered placeholder")
	})

	t.Run("filters out undersized non-rp vote extensions", func(t *testing.T) {
		setupAppResult := SetupApp(t, 4)
		hApp := setupAppResult.App
		validatorPrivKeys := setupAppResult.ValidatorKeys
		ctx := hApp.BaseApp.NewContext(true)
		ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
		validators := hApp.StakeKeeper.GetAllValidators(ctx)

		valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
		require.NoError(t, err)

		maxTxBytes := int64(1048576) // 1MB

		normalVE := createVoteExtensionOfSize(1024)

		dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
		require.NoError(t, err)

		// Undersized NonRpVE (10 bytes - below minNonRpVoteExtensionSize)
		undersizedNonRpVE := make([]byte, 10)
		for i := range undersizedNonRpVE {
			undersizedNonRpVE[i] = byte(i)
		}

		reqHeight := int64(3)
		round := int32(1)
		var votes []abci.ExtendedVoteInfo

		// Validator 0: Normal NonRpVE (should pass)
		privKey0 := findPrivKeyForValidator(validators[0], validatorPrivKeys)
		require.NotNil(t, privKey0)
		vote0 := createSignedVoteInfo(t, ctx, validators[0], privKey0, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote0)

		// Validator 1: Undersized NonRpVE (10 bytes - below minNonRpVoteExtensionSize)
		privKey1 := findPrivKeyForValidator(validators[1], validatorPrivKeys)
		require.NotNil(t, privKey1)
		vote1 := createSignedVoteInfo(t, ctx, validators[1], privKey1, normalVE, undersizedNonRpVE, reqHeight, round)
		votes = append(votes, vote1)

		// Validator 2: Normal NonRpVE (should pass) - need majority voting power
		privKey2 := findPrivKeyForValidator(validators[2], validatorPrivKeys)
		require.NotNil(t, privKey2)
		vote2 := createSignedVoteInfo(t, ctx, validators[2], privKey2, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote2)

		// Validator 3: Normal NonRpVE (should pass) - need majority voting power
		privKey3 := findPrivKeyForValidator(validators[3], validatorPrivKeys)
		require.NotNil(t, privKey3)
		vote3 := createSignedVoteInfo(t, ctx, validators[3], privKey3, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote3)

		// Filter vote extensions
		filtered, err := filterVoteExtensions(ctx, 3, votes, 1, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
		require.NoError(t, err)

		// Validator 1 has undersized NonRpVE: should be a placeholder, not dropped.
		require.Equal(t, 4, len(filtered), "All entries should be preserved (1 as placeholder)")
		placeholderFound := false
		for _, v := range filtered {
			if isFilteredPlaceholder(v) {
				placeholderFound = true
			}
		}
		require.True(t, placeholderFound, "Should have at least one filtered placeholder")
	})
}

func TestValidateVoteExtensionsCompleteness(t *testing.T) {
	valA := []byte{0x01}
	valB := []byte{0x02}
	valC := []byte{0x03}

	tests := []struct {
		name           string
		canonicalVotes []abci.VoteInfo
		extCommitVotes []abci.ExtendedVoteInfo
		shouldError    bool
		expectedErr    string
	}{
		{
			name: "all canonical commit validators present with commit flag",
			canonicalVotes: []abci.VoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			extCommitVotes: []abci.ExtendedVoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			shouldError: false,
		},
		{
			name: "canonical commit validator missing from ext commit info",
			canonicalVotes: []abci.VoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			extCommitVotes: []abci.ExtendedVoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			shouldError: true,
			expectedErr: "missing from ExtendedCommitInfo",
		},
		{
			name: "canonical commit validator downgraded to absent in ext commit info",
			canonicalVotes: []abci.VoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			extCommitVotes: []abci.ExtendedVoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagAbsent},
			},
			shouldError: true,
			expectedErr: "has flag",
		},
		{
			name: "non-commit canonical validators are ignored",
			canonicalVotes: []abci.VoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagAbsent},
				{Validator: abci.Validator{Address: valC}, BlockIdFlag: cmtTypes.BlockIDFlagNil},
			},
			extCommitVotes: []abci.ExtendedVoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			shouldError: false,
		},
		{
			name:           "empty canonical votes and ext commit info",
			canonicalVotes: []abci.VoteInfo{},
			extCommitVotes: []abci.ExtendedVoteInfo{},
			shouldError:    false,
		},
		{
			name: "filtered placeholder with commit flag passes completeness",
			canonicalVotes: []abci.VoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			extCommitVotes: []abci.ExtendedVoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				// valB is a filtered placeholder: commit flag but no extension data
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			shouldError: false,
		},
		{
			name: "extra validators in ext commit info are allowed",
			canonicalVotes: []abci.VoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			extCommitVotes: []abci.ExtendedVoteInfo{
				{Validator: abci.Validator{Address: valA}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
				{Validator: abci.Validator{Address: valB}, BlockIdFlag: cmtTypes.BlockIDFlagCommit},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVoteExtensionsCompleteness(tt.canonicalVotes, tt.extCommitVotes)
			if tt.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFilterVoteExtensions_ContentValidationFailuresBecomePlaceholdersPostPhuket(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})

	setupBaseVotes := func(t *testing.T) (sdk.Context, *HeimdallApp, *stakeTypes.ValidatorSet, []*stakeTypes.Validator, []cmtcrypto.PrivKey, []abci.ExtendedVoteInfo) {
		t.Helper()

		setupAppResult := SetupApp(t, 4)
		hApp := setupAppResult.App
		validatorPrivKeys := setupAppResult.ValidatorKeys
		ctx := hApp.BaseApp.NewContext(true)
		ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
		validators := hApp.StakeKeeper.GetAllValidators(ctx)

		valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
		require.NoError(t, err)

		normalVE := createVoteExtensionOfSize(1024)
		dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
		require.NoError(t, err)

		reqHeight := int64(3)
		round := int32(1)
		votes := make([]abci.ExtendedVoteInfo, 0, len(validators))

		for i := range validators {
			privKey := findPrivKeyForValidator(validators[i], validatorPrivKeys)
			require.NotNil(t, privKey)
			vote := createSignedVoteInfo(t, ctx, validators[i], privKey, normalVE, dummyNonRpVE, reqHeight, round)
			votes = append(votes, vote)
		}

		return ctx, hApp, &valSet, validators, validatorPrivKeys, votes
	}

	testCases := []struct {
		name   string
		mutate func(t *testing.T, ctx sdk.Context, app *HeimdallApp, validators []*stakeTypes.Validator, validatorPrivKeys []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo)
	}{
		{
			name: "unknown vote extension fields",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				votes[0].VoteExtension = appendProtobufPadding(votes[0].VoteExtension, 256)
			},
		},
		{
			name: "vote extension unmarshal failure",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				votes[0].VoteExtension = []byte{0x01, 0x02, 0x03}
			},
		},
		{
			name: "vote extension height mismatch",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				ve := new(sidetxs.VoteExtension)
				require.NoError(t, ve.Unmarshal(votes[0].VoteExtension))
				ve.Height++
				bz, err := ve.Marshal()
				require.NoError(t, err)
				votes[0].VoteExtension = bz
			},
		},
		{
			name: "invalid side tx responses",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				ve := new(sidetxs.VoteExtension)
				require.NoError(t, ve.Unmarshal(votes[0].VoteExtension))
				ve.SideTxResponses[0].TxHash = []byte{0x01}
				bz, err := ve.Marshal()
				require.NoError(t, err)
				votes[0].VoteExtension = bz
			},
		},
		{
			name: "invalid milestone proposition",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				ve := new(sidetxs.VoteExtension)
				require.NoError(t, ve.Unmarshal(votes[0].VoteExtension))
				ve.MilestoneProposition = &milestoneTypes.MilestoneProposition{
					StartBlockNumber: 10,
					BlockHashes:      [][]byte{},
					BlockTds:         []uint64{},
				}
				bz, err := ve.Marshal()
				require.NoError(t, err)
				votes[0].VoteExtension = bz
			},
		},
		{
			name: "validator not found in canonical set",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, validatorPrivKeys []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				unknownAddr := append([]byte(nil), votes[0].Validator.Address...)
				unknownAddr[0] ^= 0xFF
				votes[0].Validator.Address = unknownAddr
			},
		},
		{
			name: "validator power mismatch",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				votes[0].Validator.Power++
			},
		},
		{
			name: "empty extension signature",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				votes[0].ExtensionSignature = nil
			},
		},
		{
			name: "invalid extension signature",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				votes[0].ExtensionSignature = []byte{0x01, 0x02, 0x03}
			},
		},
		{
			name: "invalid non-rp extension signature",
			mutate: func(t *testing.T, _ sdk.Context, _ *HeimdallApp, _ []*stakeTypes.Validator, _ []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				votes[0].NonRpExtensionSignature = []byte{0x01, 0x02, 0x03}
			},
		},
		{
			name: "block hash mismatch with majority",
			mutate: func(t *testing.T, ctx sdk.Context, _ *HeimdallApp, validators []*stakeTypes.Validator, validatorPrivKeys []cmtcrypto.PrivKey, votes []abci.ExtendedVoteInfo) {
				// Create a VE with a different block hash, then re-sign it
				ve := new(sidetxs.VoteExtension)
				require.NoError(t, ve.Unmarshal(votes[0].VoteExtension))
				differentBlockHash := make([]byte, 32)
				for i := range differentBlockHash {
					differentBlockHash[i] = byte(0xFF - i) // clearly different from the default
				}
				ve.BlockHash = differentBlockHash
				bz, err := ve.Marshal()
				require.NoError(t, err)
				votes[0].VoteExtension = bz

				// Re-sign the modified VE
				privKey := findPrivKeyForValidator(validators[0], validatorPrivKeys)
				require.NotNil(t, privKey)
				cve := cmtTypes.CanonicalVoteExtension{
					Extension: bz,
					Height:    2, // reqHeight - 1
					Round:     1,
					ChainId:   ctx.ChainID(),
				}
				marshalDelimitedFn := func(msg proto.Message) ([]byte, error) {
					var buf bytes.Buffer
					if _, errW := protoio.NewDelimitedWriter(&buf).WriteMsg(msg); errW != nil {
						return nil, errW
					}
					return buf.Bytes(), nil
				}
				extSignBytes, err := marshalDelimitedFn(&cve)
				require.NoError(t, err)
				sig, err := privKey.Sign(extSignBytes)
				require.NoError(t, err)
				votes[0].ExtensionSignature = sig
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, hApp, valSet, validators, validatorPrivKeys, votes := setupBaseVotes(t)
			tc.mutate(t, ctx, hApp, validators, validatorPrivKeys, votes)

			filtered, err := filterVoteExtensions(ctx, 3, votes, 1, valSet, hApp.MilestoneKeeper, 1_048_576, sdklog.NewTestLogger(t))
			require.NoError(t, err)
			require.Len(t, filtered, len(votes), "invalid vote should be preserved as placeholder")

			placeholderCount := 0
			for _, vote := range filtered {
				if isFilteredPlaceholder(vote) {
					placeholderCount++
				}
			}
			require.Equal(t, 1, placeholderCount, "exactly one placeholder expected")
		})
	}
}

// findPrivKeyForValidator finds the private key that matches the validator's signer address
func findPrivKeyForValidator(validator *stakeTypes.Validator, privKeys []cmtcrypto.PrivKey) cmtcrypto.PrivKey {
	for _, privKey := range privKeys {
		addr := common.Bytes2Hex(privKey.PubKey().Address())
		// Compare both with and without 0x prefix, case-insensitive
		if strings.EqualFold(addr, validator.Signer) ||
			strings.EqualFold("0x"+addr, validator.Signer) ||
			strings.EqualFold(addr, strings.TrimPrefix(validator.Signer, "0x")) {
			return privKey
		}
	}
	return nil
}

// createVoteExtensionOfSize creates a vote extension of approximately the specified size
func createVoteExtensionOfSize(sizeBytes int) *sidetxs.VoteExtension {
	// Use a consistent block hash for all VEs so they can form the majority of consensus
	blockHash := make([]byte, 32)
	for i := range blockHash {
		blockHash[i] = byte(i)
	}

	ve := &sidetxs.VoteExtension{
		Height:    VoteExtBlockHeight,
		BlockHash: blockHash,
	}

	// Add side tx responses to reach the desired size
	// Each SideTxResponse is approximately 40 bytes
	responsesNeeded := sizeBytes / 40
	maxResponses := 50 // maxSideTxResponsesCount

	// If the size is too large for maxResponses, add more responses with larger tx data
	if responsesNeeded > maxResponses {
		responsesNeeded = maxResponses
	}

	for i := 0; i < responsesNeeded; i++ {
		txHash := make([]byte, 32)
		// Create some unique txs hashes for each response to avoid filtering due to duplicated votes
		// Use the index i in the first 4 bytes to ensure uniqueness
		txHash[0] = byte(i >> 24)
		txHash[1] = byte(i >> 16)
		txHash[2] = byte(i >> 8)
		txHash[3] = byte(i)
		// Fill rest with pattern based on i
		for j := 4; j < len(txHash); j++ {
			txHash[j] = byte((i*7 + j) % 256)
		}

		sideTxResp := sidetxs.SideTxResponse{
			TxHash: txHash,
			Result: sidetxs.Vote_VOTE_YES,
		}

		ve.SideTxResponses = append(ve.SideTxResponses, sideTxResp)
	}

	return ve
}

// createSignedVoteInfo creates a signed ExtendedVoteInfo with custom VoteExtension
func createSignedVoteInfo(
	t *testing.T,
	ctx sdk.Context,
	validator *stakeTypes.Validator,
	privKey cmtcrypto.PrivKey,
	voteExtension *sidetxs.VoteExtension,
	nonRpVoteExtension []byte,
	reqHeight int64,
	round int32,
) abci.ExtendedVoteInfo {
	t.Helper()

	cometVal := abci.Validator{
		Address: common.FromHex(validator.Signer),
		Power:   validator.VotingPower,
	}

	// Marshal the vote extension
	voteExtensionBytes, err := voteExtension.Marshal()
	require.NoError(t, err)

	// Sign the vote extension
	cve := cmtTypes.CanonicalVoteExtension{
		Extension: voteExtensionBytes,
		Height:    reqHeight - 1, // the vote extension was signed in the previous height
		Round:     int64(round),
		ChainId:   ctx.ChainID(),
	}

	marshalDelimitedFn := func(msg proto.Message) ([]byte, error) {
		var buf bytes.Buffer
		if _, err := protoio.NewDelimitedWriter(&buf).WriteMsg(msg); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	extSignBytes, err := marshalDelimitedFn(&cve)
	require.NoError(t, err)

	signature, err := privKey.Sign(extSignBytes)
	require.NoError(t, err)

	// Sign the non-rp vote extension
	nonRpSignature, err := privKey.Sign(nonRpVoteExtension)
	require.NoError(t, err)

	return abci.ExtendedVoteInfo{
		BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
		VoteExtension:           voteExtensionBytes,
		ExtensionSignature:      signature,
		Validator:               cometVal,
		NonRpVoteExtension:      nonRpVoteExtension,
		NonRpExtensionSignature: nonRpSignature,
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
	// Add multiple side tx responses to meet the minimum vote extension size of 10 bytes
	voteExtensionProto := sidetxs.VoteExtension{
		SideTxResponses: []sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
			{
				TxHash: make([]byte, 32), // Add a second tx response
				Result: sidetxs.Vote_VOTE_YES,
			},
			{
				TxHash: append([]byte{0x01}, txHashBytes[1:]...), // Add a third tx response with a different hash
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
	// Add multiple side tx responses to meet the minimum vote extension size of 10 bytes
	voteExtensionProto := sidetxs.VoteExtension{
		SideTxResponses: []sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
			{
				TxHash: make([]byte, 32), // Add a second tx response
				Result: sidetxs.Vote_VOTE_YES,
			},
			{
				TxHash: append([]byte{0x01}, txHashBytes[1:]...), // Add a third tx response with a different hash
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
	// Add multiple side tx responses to meet the minimum vote extension size of 10 bytes
	voteExtensionProto := sidetxs.VoteExtension{
		SideTxResponses: []sidetxs.SideTxResponse{
			{
				TxHash: txHashBytes,
				Result: sidetxs.Vote_VOTE_YES,
			},
			{
				TxHash: make([]byte, 32), // Add a second tx response
				Result: sidetxs.Vote_VOTE_YES,
			},
			{
				TxHash: append([]byte{0x01}, txHashBytes[1:]...), // Add a third tx response with a different hash
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

func TestIsFilteredPlaceholder(t *testing.T) {
	tests := []struct {
		name     string
		vote     abci.ExtendedVoteInfo
		expected bool
	}{
		{
			name: "placeholder: commit flag with all empty fields",
			vote: abci.ExtendedVoteInfo{
				Validator:   abci.Validator{Address: []byte{0x01}},
				BlockIdFlag: cmtTypes.BlockIDFlagCommit,
			},
			expected: true,
		},
		{
			name: "not placeholder: has VoteExtension",
			vote: abci.ExtendedVoteInfo{
				Validator:     abci.Validator{Address: []byte{0x01}},
				BlockIdFlag:   cmtTypes.BlockIDFlagCommit,
				VoteExtension: []byte("data"),
			},
			expected: false,
		},
		{
			name: "not placeholder: absent flag with empty fields",
			vote: abci.ExtendedVoteInfo{
				Validator:   abci.Validator{Address: []byte{0x01}},
				BlockIdFlag: cmtTypes.BlockIDFlagAbsent,
			},
			expected: false,
		},
		{
			name: "not placeholder: has ExtensionSignature only",
			vote: abci.ExtendedVoteInfo{
				Validator:          abci.Validator{Address: []byte{0x01}},
				BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				ExtensionSignature: []byte("sig"),
			},
			expected: false,
		},
		{
			name: "not placeholder: has NonRpVoteExtension only",
			vote: abci.ExtendedVoteInfo{
				Validator:          abci.Validator{Address: []byte{0x01}},
				BlockIdFlag:        cmtTypes.BlockIDFlagCommit,
				NonRpVoteExtension: []byte("nonrp"),
			},
			expected: false,
		},
		{
			name: "not placeholder: has NonRpExtensionSignature only",
			vote: abci.ExtendedVoteInfo{
				Validator:               abci.Validator{Address: []byte{0x01}},
				BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
				NonRpExtensionSignature: []byte("sig"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFilteredPlaceholder(tt.vote)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestFilterVoteExtensions_PlaceholderPassesCompleteness verifies that the full
// PrepareProposal -> ProcessProposal path works when one canonical committer
// has an oversized VE that gets filtered to a placeholder.
func TestFilterVoteExtensions_PlaceholderPassesCompleteness(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})

	setupAppResult := SetupApp(t, 4)
	hApp := setupAppResult.App
	validatorPrivKeys := setupAppResult.ValidatorKeys
	ctx := hApp.BaseApp.NewContext(true)
	ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)
	validators := hApp.StakeKeeper.GetAllValidators(ctx)

	valSet, err := hApp.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(t, err)

	maxTxBytes := int64(1048576)

	normalVE := createVoteExtensionOfSize(2000)
	dummyNonRpVE, err := GetDummyNonRpVoteExtension(2, ctx.ChainID())
	require.NoError(t, err)

	reqHeight := int64(3)
	round := int32(1)
	var votes []abci.ExtendedVoteInfo

	// Validators 0, 2, 3: normal VEs
	for _, i := range []int{0, 2, 3} {
		privKey := findPrivKeyForValidator(validators[i], validatorPrivKeys)
		require.NotNil(t, privKey)
		vote := createSignedVoteInfo(t, ctx, validators[i], privKey, normalVE, dummyNonRpVE, reqHeight, round)
		votes = append(votes, vote)
	}

	// Validator 1: oversized NonRpVE (should become placeholder)
	oversizedNonRpVE := make([]byte, 600) // exceeds maxNonRpVoteExtensionSize of 500
	for i := range oversizedNonRpVE {
		oversizedNonRpVE[i] = byte(i)
	}
	privKey1 := findPrivKeyForValidator(validators[1], validatorPrivKeys)
	require.NotNil(t, privKey1)
	vote1 := createSignedVoteInfo(t, ctx, validators[1], privKey1, normalVE, oversizedNonRpVE, reqHeight, round)
	votes = append(votes, vote1)

	// Step 1: filterVoteExtensions (PrepareProposal path)
	filtered, err := filterVoteExtensions(ctx, reqHeight, votes, round, &valSet, hApp.MilestoneKeeper, maxTxBytes, sdklog.NewTestLogger(t))
	require.NoError(t, err)
	require.Equal(t, 4, len(filtered), "All entries preserved including placeholder")

	// Step 2: Build canonical commit (as CometBFT would provide in ProcessProposal)
	canonicalVotes := make([]abci.VoteInfo, len(validators))
	for i, v := range validators {
		canonicalVotes[i] = abci.VoteInfo{
			Validator: abci.Validator{
				Address: common.FromHex(v.Signer),
				Power:   v.VotingPower,
			},
			BlockIdFlag: cmtTypes.BlockIDFlagCommit,
		}
	}

	// Step 3: ValidateVoteExtensionsCompleteness (ProcessProposal path)
	err = ValidateVoteExtensionsCompleteness(canonicalVotes, filtered)
	require.NoError(t, err, "Completeness check must pass with placeholder entries")

	// Step 4: ValidateVoteExtensions should also pass (placeholder is skipped, 3/4 VP > 2/3)
	err = ValidateVoteExtensions(ctx, reqHeight, filtered, round, &valSet, hApp.MilestoneKeeper)
	require.NoError(t, err, "VE validation must pass with placeholder when remaining VP > 2/3")
}

func setupEmptyExtendedVoteInfo(
	t *testing.T,
	flag cmtTypes.BlockIDFlag,
	blockHashBytes []byte,
	validator abci.Validator,
	privKey cmtcrypto.PrivKey,
	height int64,
	app *HeimdallApp,
) abci.ExtendedVoteInfo {
	t.Helper()

	nonRpDummyVoteExt, err := GetDummyNonRpVoteExtension(height, app.ChainID())
	require.NoErrorf(t, err, "failed to get dummy nonRpVoteExtension: %v", err)

	// create a protobuf msg for ConsolidatedSideTxResponse
	voteExtensionProto := sidetxs.VoteExtension{
		BlockHash: blockHashBytes,
		Height:    VoteExtBlockHeight,
	}

	// marshal it into Protobuf bytes
	voteExtensionBytes, err := voteExtensionProto.Marshal()
	require.NoErrorf(t, err, "failed to marshal voteExtensionProto: %v", err)

	voteInfo := abci.ExtendedVoteInfo{
		BlockIdFlag:        flag,
		VoteExtension:      voteExtensionBytes,
		Validator:          validator,
		NonRpVoteExtension: nonRpDummyVoteExt,
	}

	createSignatureForVoteExtension(t, height, privKey, voteExtensionBytes, nonRpDummyVoteExt, &voteInfo)

	return voteInfo
}

func createSignatureForVoteExtension(
	t *testing.T,
	height int64,
	privKey cmtcrypto.PrivKey,
	voteExtensionBytes,
	nonRpVoteExtensionBytes []byte,
	voteInfo *abci.ExtendedVoteInfo,
) {
	cve := cmtTypes.CanonicalVoteExtension{
		Extension: voteExtensionBytes,
		Height:    height,
		Round:     int64(0),
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
	signatureNonRpVE, err := privKey.Sign(nonRpVoteExtensionBytes)
	require.NoErrorf(t, err, "failed to sign nonRpVoteExtensionBytes: %v", err)

	voteInfo.ExtensionSignature = signature
	voteInfo.NonRpExtensionSignature = signatureNonRpVE
}
