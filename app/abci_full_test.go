package app

import (
	"crypto/sha256"
	"math/big"
	"testing"

	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testInfo struct {
	txBytes    [][]byte
	mockCaller *helpermocks.IContractCaller
}

func getTest(t *testing.T, testIdx int, priv cryptotypes.PrivKey, app *HeimdallApp, ctx sdk.Context) *testInfo {
	tests := []testInfo{
		{
			txBytes: func() [][]byte {
				msgs := []sdk.Msg{
					&types.MsgCheckpoint{
						Proposer:        priv.PubKey().Address().String(),
						StartBlock:      100,
						EndBlock:        200,
						RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
						AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
						BorChainId:      "1",
					},
				}
				txBytes := make([][]byte, len(msgs))
				for i, msg := range msgs {
					tx, err := buildSignedTx(msg, priv.PubKey().Address().String(), ctx, priv, app)
					require.NoError(t, err)
					txBytes[i] = tx
				}
				return txBytes
			}(),
			mockCaller: func() *helpermocks.IContractCaller {
				mockCaller := new(helpermocks.IContractCaller)
				mockCaller.
					On("GetBorChainBlock", mock.Anything, mock.Anything).
					Return(&ethTypes.Header{
						Number: big.NewInt(10),
					}, nil)
				mockCaller.
					On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
					Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)
				return mockCaller
			}(),
		},
	}

	if testIdx < 0 || testIdx >= len(tests) {
		return nil
	}

	return &tests[testIdx]
}

func TestFullABCI(t *testing.T) {
	for i := 0; i < 1; i++ {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		testInfo := getTest(t, i, priv, app, ctx)
		if testInfo == nil {
			break
		}

		app.caller = testInfo.mockCaller

		t.Run("execute test", func(t *testing.T) {
			executeTest(t, priv, app, ctx, validatorPrivKeys, testInfo.txBytes)
		})
	}
}

func executeTest(
	t *testing.T,
	priv cryptotypes.PrivKey,
	app *HeimdallApp,
	ctx sdk.Context,
	validatorPrivKeys []secp256k1.PrivKey,
	txBytes [][]byte,
) {
	validators := app.StakeKeeper.GetAllValidators(ctx)

	_, extCommit, _, err := buildExtensionCommits(
		t,
		app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators,
		validatorPrivKeys,
		app.LastBlockHeight(),
		nil,
	)
	require.NoError(t, err)

	voteExtensions := executeHeight(t, ctx, app, priv, *extCommit, txBytes)
	require.NotNil(t, voteExtensions)

	cometVal1 := abci.Validator{
		Address: common.FromHex(validators[0].Signer),
		Power:   validators[0].VotingPower,
	}

	voteInfo := abci.ExtendedVoteInfo{
		BlockIdFlag:        cmtproto.BlockIDFlagCommit,
		Validator:          cometVal1,
		VoteExtension:      voteExtensions.VoteExtension,
		NonRpVoteExtension: voteExtensions.NonRpExtension,
	}

	createSignatureForVoteExtension(
		t,
		app.LastBlockHeight(),
		validatorPrivKeys[0],
		voteInfo.VoteExtension,
		voteInfo.NonRpVoteExtension,
		&voteInfo,
	)

	_, extCommit, _, err = buildExtensionCommits(
		t,
		app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators,
		validatorPrivKeys,
		app.LastBlockHeight(),
		&voteInfo,
	)
	require.NoError(t, err)

	voteExtensions = executeHeight(t, ctx, app, priv, *extCommit, [][]byte{})
	require.NotNil(t, voteExtensions)
}

func executeHeight(
	t *testing.T,
	ctx sdk.Context,
	app *HeimdallApp,
	priv cryptotypes.PrivKey,
	extCommit abci.ExtendedCommitInfo,
	txBytes [][]byte,
) *abci.ResponseExtendVote {

	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Prepare proposal
	reqPrepare := &abci.RequestPrepareProposal{
		Txs:             txBytes,
		MaxTxBytes:      1_000_000,
		LocalLastCommit: extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          app.LastBlockHeight() + 1,
	}

	respPrepare, err := app.PrepareProposal(reqPrepare)
	require.NoError(t, err)
	require.NotEmpty(t, respPrepare.Txs)

	txHash := sha256.Sum256(respPrepare.GetBlob())
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

	reqFinalizeBlock := &abci.RequestFinalizeBlock{
		Height:          app.LastBlockHeight() + 1,
		Hash:            reqProcess.Hash,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Txs:             respPrepare.Txs,
	}
	_, err = app.FinalizeBlock(reqFinalizeBlock)
	require.NoError(t, err)

	_, err = app.Commit()
	require.NoError(t, err)

	return respExtend
}
