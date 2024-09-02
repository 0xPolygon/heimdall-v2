package app

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const (
	TxHash1  = "0x000000000000000000000000000000000000000000000000000000000001dead"
	TxHash2  = "0x000000000000000000000000000000000000000000000000000000000002dead"
	TxHash3  = "0x000000000000000000000000000000000000000000000000000000000003dead"
	ValAddr1 = "0x000000000000000000000000000000000001dEaD"
	ValAddr2 = "0x000000000000000000000000000000000002dEaD"
	ValAddr3 = "0x000000000000000000000000000000000003dEaD"
)

// MockKeeper is a mock of stakingKeeper.Keeper
type MockKeeper struct {
	mock.Mock
}

func (m *MockKeeper) GetValidatorSet(ctx sdk.Context) (stakeTypes.ValidatorSet, error) {
	args := m.Called(ctx)
	return args.Get(0).(stakeTypes.ValidatorSet), args.Error(1)
}

func (m *MockKeeper) GetValidatorInfo(ctx sdk.Context, addr string) (stakeTypes.Validator, error) {
	args := m.Called(ctx, addr)
	return args.Get(0).(stakeTypes.Validator), args.Error(1)
}

func TestValidateVoteExtensions(t *testing.T) {
	// TODO HV2: Implement this test
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
		validators      []*stakeTypes.Validator
		extVoteInfo     []abci.ExtendedVoteInfo
		expectedApprove [][]byte
		expectedReject  [][]byte
		expectedSkip    [][]byte
	}{
		{
			name: "single tx approved with 2/3+1 majority",
			validators: []*stakeTypes.Validator{
				{VotingPower: 10},
				{VotingPower: 20},
				{VotingPower: 1},
			},
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator: abci.Validator{
						Address: val1,
						Power:   10,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val2,
						Power:   20,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val3,
						Power:   1,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
			},
			expectedApprove: [][]byte{[]byte(TxHash1)},
			expectedReject:  make([][]byte, 0, 3),
			expectedSkip:    make([][]byte, 0, 3),
		},
		{
			name: "one tx approved one rejected one skipped",
			validators: []*stakeTypes.Validator{
				{VotingPower: 40},
				{VotingPower: 30},
				{VotingPower: 5},
			},
			extVoteInfo: []abci.ExtendedVoteInfo{
				{
					Validator: abci.Validator{
						Address: val1,
						Power:   40,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val1,
						Power:   40,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash2),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},

				{
					Validator: abci.Validator{
						Address: val1,
						Power:   40,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash3),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val2,
						Power:   30,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash1),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val2,
						Power:   30,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash2),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val2,
						Power:   30,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash3),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val3,
						Power:   5,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_NO, TxHash1),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
				{
					Validator: abci.Validator{
						Address: val3,
						Power:   5,
					},
					VoteExtension:      mustMarshalSideTxResponses(t, sidetxs.Vote_VOTE_YES, TxHash2),
					ExtensionSignature: []byte("signature"),
					BlockIdFlag:        types.BlockIDFlagCommit,
				},
			},
			expectedApprove: [][]byte{[]byte(TxHash1)},
			expectedReject:  [][]byte{[]byte(TxHash2)},
			expectedSkip:    [][]byte{[]byte(TxHash3)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			approvedTxs, rejectedTxs, skippedTxs, err := tallyVotes(tc.extVoteInfo, log.NewTestLogger(t), tc.validators)
			require.NoError(t, err)
			require.Equal(t, tc.expectedApprove, approvedTxs)
			require.Equal(t, tc.expectedReject, rejectedTxs)
			require.Equal(t, tc.expectedSkip, skippedTxs)
		})
	}
}

func TestAggregateVotes(t *testing.T) {
	txHashStr := "000000000000000000000000000000000000000000000000000000000001dead"
	hashStr := "000000000000000000000000000000000000000000000000000000000000dead"

	// Convert hex strings to byte slices
	txHashBytes, err := hex.DecodeString(txHashStr)
	require.NoError(t, err)
	hashBytes, err := hex.DecodeString(hashStr)
	require.NoError(t, err)

	// Prepare a valid JSON for VoteExtension with base64 encoding for bytes
	voteExtension := `{
		"side_tx_responses": [
			{
				"tx_hash": "` + base64.StdEncoding.EncodeToString(txHashBytes) + `",
				"result": 1
			}
		],
		"hash": "` + base64.StdEncoding.EncodeToString(hashBytes) + `",
		"height": 100
	}`

	val1, err := address.NewHexCodec().StringToBytes(ValAddr1)
	require.NoError(t, err)

	extVoteInfo := []abci.ExtendedVoteInfo{
		{
			Validator: abci.Validator{
				Address: val1,
				Power:   10,
			},
			VoteExtension:      []byte(voteExtension),
			ExtensionSignature: []byte("signature"),
			BlockIdFlag:        types.BlockIDFlagCommit,
		},
	}

	expectedVotes := map[string]map[sidetxs.Vote]int64{
		txHashStr: {
			sidetxs.Vote_VOTE_YES: 10,
		},
	}

	actualVotes, err := aggregateVotes(extVoteInfo)
	require.NoError(t, err)
	require.NotEmpty(t, actualVotes)
	require.Equal(t, expectedVotes, actualVotes)
}

func TestCheckDuplicateVotes(t *testing.T) {
	sideTxResponses := []*sidetxs.SideTxResponse{
		{TxHash: []byte(TxHash1)},
		{TxHash: []byte(TxHash2)},
		{TxHash: []byte(TxHash1)},
	}

	duplicate, txHash := checkDuplicateVotes(sideTxResponses)
	require.True(t, duplicate)
	require.Equal(t, []byte(TxHash1), txHash)
}

func TestIsVoteValid(t *testing.T) {
	require.True(t, isVoteValid(sidetxs.Vote_UNSPECIFIED))
	require.True(t, isVoteValid(sidetxs.Vote_VOTE_YES))
	require.True(t, isVoteValid(sidetxs.Vote_VOTE_NO))
	require.False(t, isVoteValid(100))
}

func TestMustAddSpecialTransaction(t *testing.T) {
	// TODO HV2: Implement this test
}

func mustMarshalSideTxResponses(t *testing.T, vote sidetxs.Vote, txHashes ...string) []byte {
	responses := make([]*sidetxs.SideTxResponse, len(txHashes))
	for i, txHash := range txHashes {
		responses[i] = &sidetxs.SideTxResponse{
			TxHash: []byte(txHash),
			Result: vote,
		}
	}

	sideTxResponses := sidetxs.ConsolidatedSideTxResponse{
		SideTxResponses: responses,
	}

	voteExtension, err := json.Marshal(sideTxResponses)
	require.NoError(t, err)
	return voteExtension
}
