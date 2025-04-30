package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	testutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

func TestPrepareProposal(t *testing.T) {

	priv, _, _ := testdata.KeyTestPubAddr()
	// Setup test app with 3 validators
	setupResult := SetupApp(t, 1)
	app := setupResult.App

	genState := app.DefaultGenesis()
	genBytes, err := json.Marshal(genState)
	require.NoError(t, err)
	app.InitChain(&abci.RequestInitChain{
		Validators:    []abci.ValidatorUpdate{},
		AppStateBytes: genBytes,
	})

	// Initialize the application state
	ctx := app.BaseApp.NewContext(true)

	// Set up consensus params
	params := cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.
		On("GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, nil)
	// stub any other calls used in GenMilestoneProposition:
	// mockCaller.
	// 	On("GetBlockNumberFromTxHash", mock.Anything).
	// 	Return(big.NewInt(0), nil).Times(2)

	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		app.AppCodec(),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)),
		mockCaller,
	)
	// If you also hit checkpoint or bor RPCs in your test, stub those too:
	app.CheckpointKeeper = checkpointKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(checkpointTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		&app.StakeKeeper,
		app.ChainManagerKeeper,
		&app.TopupKeeper,
		mockCaller,
	)
	app.BorKeeper = borKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ChainManagerKeeper,
		&app.StakeKeeper,
		nil,
	)
	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller

	app.caller = mockCaller

	validatorPrivKeys := setupResult.ValidatorKeys
	validators := app.StakeKeeper.GetAllValidators(ctx)

	cometVal1 := abci.Validator{
		Address: common.FromHex(validators[0].Signer),
		Power:   validators[0].VotingPower,
	}

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "test",
	}

	propBytes := common.FromHex(validators[0].Signer) // or use FromBech32 if it’s bech32‑encoded
	propAddr := sdk.AccAddress(propBytes)

	// create & set a BaseAccount for that address
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)

	// fund it so it can pay fees
	require.NoError(t,
		testutil.FundAccount(
			ctx,
			app.BankKeeper,
			propAddr,
			sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
		),
	)

	// addr := sdk.AccAddress(priv.PubKey().Address())
	// acc := authTypes.NewBaseAccount(addr, priv.PubKey(), 1337, 0)
	// require.NoError(t, testutil.FundAccount(ctx, app.BankKeeper, addr, sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount))))

	// app.AccountKeeper.SetAccount(ctx, acc)

	txConfig := authtx.NewTxConfig(app.AppCodec(), authtx.DefaultSignModes)
	defaultSignMode, err := authsigning.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())

	app.SetTxDecoder(txConfig.TxDecoder())

	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	err = txBuilder.SetMsgs(msg)
	require.NoError(t, err)

	sigV2 := signing.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  defaultSignMode,
			Signature: nil,
		},
		Sequence: 0,
	}

	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	// Second round: all signer infos are set, so each signer can sign.
	signerData := authsigning.SignerData{
		ChainID:       "",
		AccountNumber: 1337,
		Sequence:      0,
		PubKey:        priv.PubKey(),
	}
	sigV2, err = tx.SignWithPrivKey(
		context.TODO(), defaultSignMode, signerData,
		txBuilder, priv, txConfig, 0)
	require.NoError(t, err)
	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	// Send the tx to the app
	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	cmtPubKey, err := validators[0].CmtConsPublicKey()
	voteInfo1 := setupExtendedVoteInfoWithNonRp(
		t,
		cmtproto.BlockIDFlagCommit,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal1,
		validatorPrivKeys[0],
		2,
		app,
		cmtPubKey.GetEd25519(),
	)

	extCommit := &abci.ExtendedCommitInfo{
		Round: 1,
		Votes: []abci.ExtendedVoteInfo{
			voteInfo1,
		},
	}
	// Marshal the extended commit info
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)
	// Initialize state with FinalizeBlock
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 3,
		Txs:    [][]byte{extCommitBytes, txBytes},
	})
	require.NoError(t, err)

	// Commit the block
	_, err = app.Commit()
	require.NoError(t, err)

	// Prepare proposal request
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1000000, // Arbitrary large value for test
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          3,
	}
	_, err = app.PrepareProposal(reqPrep)
	require.NoError(t, err)

	// Test PrepareProposal handler
	respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)

	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)

	// Test ProcessProposal handler
	reqProcess := &abci.RequestProcessProposal{
		Txs:                respPrep.Txs,
		Height:             reqPrep.Height,
		ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
	}

	respProc, err := app.NewProcessProposalHandler()(ctx, reqProcess)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, respProc.Status)

	fmt.Println("----------------------------------------------------------------------------", app.caller)

	// Test ExtendVote handler
	reqExtend := abci.RequestExtendVote{
		Txs:    respPrep.Txs,
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))
	mockCaller.AssertExpectations(t)
	require.NotNil(t, respExtend.VoteExtension)

	fmt.Println("finally!")

	// Test VerifyVoteExtension handler
	// reqVerify := abci.RequestVerifyVoteExtension{
	// 	VoteExtension:      respExtend.VoteExtension,
	// 	NonRpVoteExtension: respExtend.NonRpExtension,
	// 	ValidatorAddress:   []byte("validator-1"),
	// 	Height:             3,
	// 	Hash:               []byte("test-hash"),
	// }
	// respVerify, err := app.VerifyVoteExtensionHandler()(ctx, &reqVerify)
	// fmt.Println("Hello world")
	// require.NoError(t, err)
	// fmt.Println("Helloworld")
	// require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, respVerify.Status)

	// // Test FinalizeBlock handler
	// finalizeReq := abci.RequestFinalizeBlock{
	// 	Txs:    respPrep.Txs,
	// 	Height: 4,
	// 	Time:   ctx.BlockTime(),
	// }
	// _, err = app.PreBlocker(ctx, &finalizeReq)
	// require.NoError(t, err)

	// // Commit the block
	// _, err = app.Commit()
	// require.NoError(t, err)
}

var defaultFeeAmount = big.NewInt(10).Exp(big.NewInt(10), big.NewInt(15), nil).Int64()
