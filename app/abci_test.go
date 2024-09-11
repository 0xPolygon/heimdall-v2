package app

import (
	"errors"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

func TestNewPrepareProposalHandler(t *testing.T) {
	t.Skip("TODO HV2: fix and enable this test")
	hApp, _, _ := SetupApp(t, 1)
	// finalize block so we have CheckTx state set
	_, err := hApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
	})
	require.NoError(t, err)
	ctx := cosmostestutil.DefaultContextWithKeys(hApp.keys, hApp.tKeys, nil)

	tests := []struct {
		name           string
		req            *abci.RequestPrepareProposal
		expectedError  error
		expectedTxSize int
	}{
		{
			name: "No transactions in request",
			req: &abci.RequestPrepareProposal{
				Height: CurrentHeight,
				Txs:    [][]byte{},
			},
			expectedError: errors.New("no txs found in the request to prepare the proposal"),
		},
		{
			name: "Invalid transaction during checkTx",
			req: &abci.RequestPrepareProposal{
				Height: CurrentHeight,
				Txs:    [][]byte{common.Hex2Bytes(TxHash1), {2}}, // Second transaction will fail
			},
			expectedError: errors.New("invalid transaction"),
		},
		{
			name: "Successful proposal preparation",
			req: &abci.RequestPrepareProposal{
				Height: CurrentHeight,
				Txs:    [][]byte{common.Hex2Bytes(TxHash1), []byte(TxHash2)}, // All valid transactions
				LocalLastCommit: abci.ExtendedCommitInfo{
					Votes: []abci.ExtendedVoteInfo{},
				},
			},
			expectedError:  nil,
			expectedTxSize: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := hApp.NewPrepareProposalHandler()

			resp, err := handler(setupContextWithVoteExtensionsEnableHeight(ctx, 1), tt.req)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Len(t, resp.Txs, tt.expectedTxSize)
			}
		})
	}
}

func TestNewProcessProposalHandler(t *testing.T) {
	t.Skip("TODO HV2: fix and enable this test")
	hApp, _, _ := SetupApp(t, 1)
	// finalize block so we have CheckTx state set
	_, err := hApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
	})
	require.NoError(t, err)
	ctx := cosmostestutil.DefaultContextWithKeys(hApp.keys, hApp.tKeys, nil)

	tests := []struct {
		name           string
		req            *abci.RequestProcessProposal
		expectedStatus abci.ResponseProcessProposal_ProposalStatus
	}{
		{
			name: "No transactions in request",
			req: &abci.RequestProcessProposal{
				Height: CurrentHeight,
				Txs:    [][]byte{},
			},
			expectedStatus: abci.ResponseProcessProposal_REJECT,
		},
		{
			name: "Invalid transaction during checkTx",
			req: &abci.RequestProcessProposal{
				Height: CurrentHeight,
				Txs:    [][]byte{common.Hex2Bytes(TxHash1), {2}}, // Second transaction will fail
			},
			expectedStatus: abci.ResponseProcessProposal_REJECT,
		},
		{
			name: "Valid proposal with majority",
			req: &abci.RequestProcessProposal{
				Height: CurrentHeight,
				Txs:    [][]byte{common.Hex2Bytes(TxHash1), []byte(TxHash2)}, // All valid transactions
			},
			expectedStatus: abci.ResponseProcessProposal_ACCEPT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := hApp.NewProcessProposalHandler()

			resp, err := handler(setupContextWithVoteExtensionsEnableHeight(ctx, 1), tt.req)
			require.NoError(t, err)
			require.Equal(t, tt.expectedStatus, resp.Status)
		})
	}
}

func TestExtendVote(t *testing.T) {
	t.Skip("TODO HV2: fix and enable this test")
	hApp, _, _ := SetupApp(t, 1)
	// finalize block so we have CheckTx state set
	_, err := hApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
	})
	require.NoError(t, err)
	ctx := cosmostestutil.DefaultContextWithKeys(hApp.keys, hApp.tKeys, nil)

	tests := []struct {
		name          string
		req           *abci.RequestExtendVote
		expectedError error
	}{
		{
			name: "Invalid ExtendedVoteInfo decoding",
			req: &abci.RequestExtendVote{
				Height: CurrentHeight,
				Txs:    [][]byte{{0x01}}, // Invalid data
			},
			expectedError: errors.New("error occurred while decoding ExtendedVoteInfos"),
		},
		{
			name: "Valid ExtendVoteHandler",
			req: &abci.RequestExtendVote{
				Height: CurrentHeight,
				Txs:    [][]byte{common.Hex2Bytes(TxHash1)},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hApp.sideTxCfg = sidetxs.NewSideTxConfigurator()
			voteExtProcessor := NewVoteExtensionProcessor()
			handler := voteExtProcessor.ExtendVoteHandler()

			_, err := handler(setupContextWithVoteExtensionsEnableHeight(ctx, 1), tt.req)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestVerifyVoteExtension(t *testing.T) {
	t.Skip("TODO HV2: fix and enable this test")
	hApp, _, _ := SetupApp(t, 1)
	// finalize block so we have CheckTx state set
	_, err := hApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
	})
	require.NoError(t, err)
	ctx := cosmostestutil.DefaultContextWithKeys(hApp.keys, hApp.tKeys, nil)

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

	tests := []struct {
		name           string
		req            *abci.RequestVerifyVoteExtension
		expectedStatus abci.ResponseVerifyVoteExtension_VerifyStatus
	}{
		{
			name: "Invalid VoteExtension decoding",
			req: &abci.RequestVerifyVoteExtension{
				Height:        CurrentHeight,
				VoteExtension: []byte{0x01}, // Invalid data
			},
			expectedStatus: abci.ResponseVerifyVoteExtension_REJECT,
		},
		{
			name: "Mismatched block height",
			req: &abci.RequestVerifyVoteExtension{
				Height:        CurrentHeight,
				VoteExtension: voteExtensionBytes,
			},
			expectedStatus: abci.ResponseVerifyVoteExtension_REJECT,
		},
		{
			name: "Valid VoteExtension",
			req: &abci.RequestVerifyVoteExtension{
				VoteExtension: voteExtensionBytes,
				Height:        CurrentHeight,
				Hash:          common.Hex2Bytes(TxHash1),
			},
			expectedStatus: abci.ResponseVerifyVoteExtension_ACCEPT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hApp.sideTxCfg = sidetxs.NewSideTxConfigurator()
			voteExtProcessor := NewVoteExtensionProcessor()
			handler := voteExtProcessor.VerifyVoteExtensionHandler()

			resp, err := handler(setupContextWithVoteExtensionsEnableHeight(ctx, 1), tt.req)
			require.NoError(t, err)
			require.Equal(t, tt.expectedStatus, resp.Status)
		})
	}
}

func TestPreBlocker(t *testing.T) {
	t.Skip("TODO HV2: fix and enable this test")
	hApp, _, _ := SetupApp(t, 1)
	// finalize block so we have CheckTx state set
	_, err := hApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
	})
	require.NoError(t, err)
	ctx := cosmostestutil.DefaultContextWithKeys(hApp.keys, hApp.tKeys, nil)

	tests := []struct {
		name          string
		req           *abci.RequestFinalizeBlock
		expectedError error
	}{
		{
			name: "Invalid ExtendedVoteInfo decoding",
			req: &abci.RequestFinalizeBlock{
				Height: CurrentHeight,
				Txs:    [][]byte{{0x01}}, // Invalid data
			},
			expectedError: errors.New("error occurred while unmarshalling ExtendedVoteInfo"),
		},
		{
			name: "No Validators found",
			req: &abci.RequestFinalizeBlock{
				Height: CurrentHeight,
				Txs:    [][]byte{common.Hex2Bytes(TxHash1)},
			},
			expectedError: errors.New("no validators found"),
		},
		{
			name: "Successful PreBlock execution",
			req: &abci.RequestFinalizeBlock{
				Height: CurrentHeight,
				Txs:    [][]byte{common.Hex2Bytes(TxHash1)},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := hApp.PreBlocker(setupContextWithVoteExtensionsEnableHeight(ctx, 1), tt.req)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
