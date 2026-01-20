package app

import (
	"crypto/sha256"
	"math/big"
	"testing"

	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFullABCI(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)

	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(&ethTypes.Header{
			Number: big.NewInt(10),
		}, nil)
	mockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

	app.caller = mockCaller

	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        priv.PubKey().Address().String(),
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "1",
	}

	txBytes, err := buildSignedTx(msg, priv.PubKey().Address().String(), ctx, priv, app)
	require.NoError(t, err)

	_, extCommit, _, err := buildExtensionCommits(
		t,
		app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators,
		validatorPrivKeys,
		app.LastBlockHeight(),
	)
	require.NoError(t, err)

	// Prepare proposal
	reqPrepare := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          app.LastBlockHeight() + 1,
	}

	respPrepare, err := app.PrepareProposal(reqPrepare)
	require.NoError(t, err)
	require.NotEmpty(t, respPrepare.Txs)

	txHash := sha256.Sum256(txBytes)
	hash := common.BytesToHash(txHash[:])

	// Process proposal
	reqProcess := &abci.RequestProcessProposal{
		Txs:                respPrepare.Txs,
		ProposedLastCommit: abci.CommitInfo{Round: reqPrepare.LocalLastCommit.Round},
		ProposerAddress:    common.FromHex(validators[0].Signer),
		Height:             app.LastBlockHeight() + 1,
		Hash:               hash.Bytes(),
	}

	resProcess, err := app.ProcessProposal(reqProcess)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, resProcess.Status)

	// Extend vote
	reqExtend := &abci.RequestExtendVote{
		Height:          app.LastBlockHeight() + 1,
		Hash:            reqProcess.Hash,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Txs:             respPrepare.Txs,
	}

	respExtend, err := app.ExtendVote(t.Context(), reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)
	require.NotNil(t, respExtend.NonRpExtension)

	// Verify vote extension
	reqVerifyExt := &abci.RequestVerifyVoteExtension{
		Height:             app.LastBlockHeight() + 1,
		Hash:               reqProcess.Hash,
		ValidatorAddress:   common.FromHex(validators[0].Signer),
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
	}
	respVerifyExt, err := app.VerifyVoteExtension(reqVerifyExt)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, respVerifyExt.Status)

	// PreBlocker
	reqPreBlocker := &abci.RequestFinalizeBlock{
		Height:          app.LastBlockHeight() + 1,
		Hash:            reqProcess.Hash,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Txs:             respPrepare.Txs,
	}

	_, err = app.PreBlocker(ctx, reqPreBlocker)
	require.NoError(t, err)
}
