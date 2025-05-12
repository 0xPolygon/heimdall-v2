package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	stakinginfo "github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"

	// chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	chainmanagerKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topupKeeper "github.com/0xPolygon/heimdall-v2/x/topup/keeper"
	topUpTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"

	gogoproto "github.com/gogo/protobuf/proto"

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

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

func buildSignedTx2(
	msg sdk.Msg,
	ctx sdk.Context,
	priv cryptotypes.PrivKey,
	app *HeimdallApp,
) ([]byte, error) {
	// 1) derive the fee-payer address (also your only signer)
	feePayerAddr := sdk.AccAddress(priv.PubKey().Address())

	// 2) create & register the account in state
	acct := authTypes.NewBaseAccount(feePayerAddr, priv.PubKey(), 1337, 0)
	app.AccountKeeper.SetAccount(ctx, acct)

	// 3) fund it so it can actually pay fees
	testutil.FundAccount(
		ctx,
		app.BankKeeper,
		feePayerAddr,
		sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
	)

	// 4) set up the TxBuilder
	txConfig := authtx.NewTxConfig(app.AppCodec(), authtx.DefaultSignModes)
	defaultSignMode, _ := authsigning.APISignModeToInternal(
		txConfig.SignModeHandler().DefaultMode(),
	)
	app.SetTxDecoder(txConfig.TxDecoder())

	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	txBuilder.SetMsgs(msg)

	// 5) force this account to be the explicit fee-payer
	txBuilder.SetFeePayer(feePayerAddr)

	// 6) now tell the SDK “I’m going to sign two slots”
	emptySig := signing.SignatureV2{
		PubKey:   priv.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: defaultSignMode},
		Sequence: 0,
	}
	txBuilder.SetSignatures(emptySig, emptySig) // ← two placeholders

	// 7) prepare your signer metadata
	signerData := authsigning.SignerData{
		ChainID:       "test-chain", // use your actual chain ID
		AccountNumber: 1337,
		Sequence:      0,
		PubKey:        priv.PubKey(),
	}

	// 8) sign slot #0 (the “message” signer)
	sigMsg, err := tx.SignWithPrivKey(
		context.TODO(),
		defaultSignMode,
		signerData,
		txBuilder,
		priv,
		txConfig,
		0, // index 0
	)
	if err != nil {
		return nil, err
	}
	// re-apply with slot 0 filled
	txBuilder.SetSignatures(sigMsg, emptySig)

	// 9) sign slot #1 (the “fee-payer” signer)
	sigFee, err := tx.SignWithPrivKey(
		context.TODO(),
		defaultSignMode,
		signerData,
		txBuilder,
		priv,
		txConfig,
		1, // index 1
	)
	if err != nil {
		return nil, err
	}
	// now we have both
	txBuilder.SetSignatures(sigMsg, sigFee)

	// 10) finally encode
	return txConfig.TxEncoder()(txBuilder.GetTx())
}

func genTestValidators() (stakeTypes.ValidatorSet, []stakeTypes.Validator) {
	var TestValidators = []stakeTypes.Validator{
		{
			ValId:       3,
			StartEpoch:  0,
			EndEpoch:    0,
			VotingPower: 10000,
			PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
			Signer:      "0x1c4f0f054a0d6a1415382dc0fd83c6535188b220",
			LastUpdated: "0",
		},
		{
			ValId:       4,
			StartEpoch:  0,
			EndEpoch:    0,
			VotingPower: 10000,
			PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
			Signer:      "0x461295d3d9249215e758e939a150ab180950720b",
			LastUpdated: "0",
		},
		{
			ValId:       5,
			StartEpoch:  0,
			EndEpoch:    0,
			VotingPower: 10000,
			PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
			Signer:      "0x836fe3e3dd0a5f77d9d5b0f67e48048aaafcd5a0",
			LastUpdated: "0",
		},
		{
			ValId:       1,
			StartEpoch:  0,
			EndEpoch:    0,
			VotingPower: 10000,
			PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
			Signer:      "0x925a91f8003aaeabea6037103123b93c50b86ca3",
			LastUpdated: "0",
		},
		{
			ValId:       2,
			StartEpoch:  0,
			EndEpoch:    0,
			VotingPower: 10000,
			PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
			Signer:      "0xc787af4624cb3e80ee23ae7faac0f2acea2be34c",
			LastUpdated: "0",
		},
	}

	validators := make([]*stakeTypes.Validator, 0, len(TestValidators))
	for _, v := range TestValidators {
		validators = append(validators, &v)
	}
	valSet := stakeTypes.ValidatorSet{
		Validators: validators,
	}

	vals := make([]stakeTypes.Validator, 0, len(validators))
	for _, v := range validators {
		vals = append(vals, *v)
	}

	return valSet, vals
}

func buildSignedTx(msg sdk.Msg, signer string, ctx sdk.Context, priv cryptotypes.PrivKey, app HeimdallApp) ([]byte, error) {
	propBytes := common.FromHex(signer)
	propAddr := sdk.AccAddress(propBytes)
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)

	testutil.FundAccount(ctx, app.BankKeeper, propAddr,
		sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
	)

	// Build and sign the tx
	txConfig := authtx.NewTxConfig(app.AppCodec(), authtx.DefaultSignModes)
	defaultSignMode, err := authsigning.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())
	app.SetTxDecoder(txConfig.TxDecoder())

	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	txBuilder.SetMsgs(msg)

	sigV2 := signing.SignatureV2{PubKey: priv.PubKey(), Data: &signing.SingleSignatureData{
		SignMode:  defaultSignMode,
		Signature: nil,
	}, Sequence: 0}
	txBuilder.SetSignatures(sigV2)

	signerData := authsigning.SignerData{
		ChainID:       "",
		AccountNumber: 1337,
		Sequence:      0,
		PubKey:        priv.PubKey(),
	}
	sigV2, err = tx.SignWithPrivKey(context.TODO(), defaultSignMode, signerData,
		txBuilder, priv, txConfig, 0)

	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	return txBytes, err
}

func buildExtensionCommits(t *testing.T, app *HeimdallApp, txHashBytes []byte, validators []*stakeTypes.Validator, validatorPrivKeys []secp256k1.PrivKey) ([]byte, *abci.ExtendedCommitInfo, *abci.ExtendedVoteInfo, error) {

	cometVal := abci.Validator{
		Address: common.FromHex(validators[0].Signer),
		Power:   validators[0].VotingPower,
	}

	cmtPubKey, err := validators[0].CmtConsPublicKey()

	voteInfo := setupExtendedVoteInfoWithNonRp(
		t,
		cmtproto.BlockIDFlagCommit,
		txHashBytes,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal,
		validatorPrivKeys[0],
		2,
		app,
		cmtPubKey.GetEd25519(),
	)

	extCommit := &abci.ExtendedCommitInfo{
		Round: 1,
		Votes: []abci.ExtendedVoteInfo{voteInfo},
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)
	return extCommitBytes, extCommit, &voteInfo, err
}

func SetupAppWithABCIctx(t *testing.T) (cryptotypes.PrivKey, HeimdallApp, sdk.Context, []secp256k1.PrivKey) {
	priv, _, _ := testdata.KeyTestPubAddr()

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

	validatorPrivKeys := setupResult.ValidatorKeys
	return priv, *app, ctx, validatorPrivKeys
}

func TestPrepareProposalHandler(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "test",
	}

	txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 3,
		Txs:    [][]byte{extCommitBytes, txBytes},
	})
	require.NoError(t, err)

	// Prepare/Process proposal
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          3,
	}

	_, err = app.PrepareProposal(reqPrep)
	require.NoError(t, err)

	respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)
}

func TestProcessProposalHandler(t *testing.T) {

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "test",
	}

	txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 3,
		Txs:    [][]byte{extCommitBytes, txBytes},
	})
	require.NoError(t, err)

	// Prepare/Process proposal
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          3,
	}

	_, err = app.PrepareProposal(reqPrep)
	require.NoError(t, err)

	respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)

	reqProcess := &abci.RequestProcessProposal{
		Txs:                respPrep.Txs,
		Height:             3,
		ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
	}
	respProc, err := app.NewProcessProposalHandler()(ctx, reqProcess)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, respProc.Status)

	// Table-driven tests for ProcessProposalHandler
	testCases := []struct {
		name       string
		req        *abci.RequestProcessProposal
		wantStatus abci.ResponseProcessProposal_ProposalStatus
	}{
		{
			name: "valid transactions",
			req: &abci.RequestProcessProposal{
				Txs:                respPrep.Txs,
				Height:             3,
				ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
			},
			wantStatus: abci.ResponseProcessProposal_ACCEPT,
		},
		{
			name: "no transactions",
			req: &abci.RequestProcessProposal{
				Txs:                [][]byte{},
				Height:             3,
				ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
			},
			wantStatus: abci.ResponseProcessProposal_REJECT,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			respProc, err := app.NewProcessProposalHandler()(ctx, tc.req)
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, respProc.Status)
		})
	}
}

func TestExtendVoteHandler(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "test",
	}

	txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 3,
		Txs:    [][]byte{extCommitBytes, txBytes},
	})
	require.NoError(t, err)

	// Prepare/Process proposal
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          3,
	}

	_, err = app.PrepareProposal(reqPrep)
	require.NoError(t, err)

	respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)

	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.
		On("GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, nil)

	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		app.AppCodec(),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)),
		mockCaller,
	)
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

	reqExtend := abci.RequestExtendVote{
		Txs:    respPrep.Txs,
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)
	mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	terrUnmarshal := "error occurred while decoding ExtendedCommitInfo"
	terrTxDecode := "error occurred while decoding tx bytes in ExtendVoteHandler"
	testCases := []struct {
		name        string
		req         abci.RequestExtendVote
		wantErr     bool
		errContains string
	}{
		{
			name: "valid extend vote",
			req: abci.RequestExtendVote{
				Txs:    respPrep.Txs,
				Hash:   []byte("test-hash"),
				Height: 3,
			},
			wantErr: false,
		},
		{
			name: "unmarshal failure",
			req: abci.RequestExtendVote{
				Txs:    [][]byte{{0x01, 0x02, 0x03}},
				Hash:   []byte("test-hash"),
				Height: 3,
			},
			wantErr:     true,
			errContains: terrUnmarshal,
		},
		{
			name: "tx decode failure",
			req: abci.RequestExtendVote{
				Txs:    [][]byte{respPrep.Txs[0], {0x01, 0x02, 0x03}},
				Hash:   []byte("test-hash"),
				Height: 3,
			},
			wantErr:     true,
			errContains: terrTxDecode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			respExtend, err := app.ExtendVoteHandler()(ctx, &tc.req)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, respExtend)
			} else {
				require.NoError(t, err)
				require.NotNil(t, respExtend)
				mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))
			}
		})
	}
}

func TestVerifyVoteExtensionHandler(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "test",
	}

	txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)

	extCommitBytes, extCommit, voteInfo, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 3,
		Txs:    [][]byte{extCommitBytes, txBytes},
	})
	require.NoError(t, err)

	// Prepare/Process proposal
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          3,
	}

	_, err = app.PrepareProposal(reqPrep)
	require.NoError(t, err)

	respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)

	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.
		On("GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, nil)

	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		app.AppCodec(),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)),
		mockCaller,
	)
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

	reqExtend := abci.RequestExtendVote{
		Txs:    respPrep.Txs,
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)
	mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	reqVerify := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo.Validator.Address, // <<< use the real consensus addr
		Height:             3,
		Hash:               []byte("test-hash"),
	}
	respVerify, err := app.VerifyVoteExtensionHandler()(ctx, &reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, respVerify.Status)

	// Table-driven cases for VerifyVoteExtensionHandler
	testCases := []struct {
		name       string
		req        abci.RequestVerifyVoteExtension
		wantStatus abci.ResponseVerifyVoteExtension_VerifyStatus
	}{
		{
			name:       "valid extension",
			req:        abci.RequestVerifyVoteExtension{VoteExtension: respExtend.VoteExtension, NonRpVoteExtension: respExtend.NonRpExtension, ValidatorAddress: voteInfo.Validator.Address, Height: 3, Hash: []byte("test-hash")},
			wantStatus: abci.ResponseVerifyVoteExtension_ACCEPT,
		},
		{
			name:       "unmarshal fail",
			req:        abci.RequestVerifyVoteExtension{VoteExtension: []byte{0x01, 0x02, 0x03}, NonRpVoteExtension: respExtend.NonRpExtension, ValidatorAddress: voteInfo.Validator.Address, Height: 3, Hash: []byte("test-hash")},
			wantStatus: abci.ResponseVerifyVoteExtension_REJECT,
		},
		{
			name:       "height mismatch",
			req:        abci.RequestVerifyVoteExtension{VoteExtension: respExtend.VoteExtension, NonRpVoteExtension: respExtend.NonRpExtension, ValidatorAddress: voteInfo.Validator.Address, Height: 4, Hash: []byte("test-hash")},
			wantStatus: abci.ResponseVerifyVoteExtension_REJECT,
		},
		{
			name:       "hash mismatch",
			req:        abci.RequestVerifyVoteExtension{VoteExtension: respExtend.VoteExtension, NonRpVoteExtension: respExtend.NonRpExtension, ValidatorAddress: voteInfo.Validator.Address, Height: 3, Hash: []byte("wrong-hash")},
			wantStatus: abci.ResponseVerifyVoteExtension_REJECT,
		},
		{
			name: "side-tx validation failure",
			// construct invalid side extension bytes
			req: func() abci.RequestVerifyVoteExtension {
				fake := &sidetxs.VoteExtension{BlockHash: respExtend.VoteExtension, Height: 3, SideTxResponses: nil}
				bz, _ := gogoproto.Marshal(fake)
				return abci.RequestVerifyVoteExtension{VoteExtension: bz, NonRpVoteExtension: respExtend.NonRpExtension, ValidatorAddress: voteInfo.Validator.Address, Height: 3, Hash: []byte("test-hash")}
			}(),
			wantStatus: abci.ResponseVerifyVoteExtension_REJECT,
		},
		{
			name:       "non-rp validation error",
			req:        abci.RequestVerifyVoteExtension{VoteExtension: respExtend.VoteExtension, NonRpVoteExtension: []byte{0x01, 0x02, 0x03, 0xFF}, ValidatorAddress: voteInfo.Validator.Address, Height: 3, Hash: []byte("test-hash")},
			wantStatus: abci.ResponseVerifyVoteExtension_ACCEPT,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := app.VerifyVoteExtensionHandler()(ctx, &tc.req)
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, resp.Status)
		})
	}
}

func TestPreBlocker(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	msg := &borTypes.MsgProposeSpan{
		// SpanId:     2,
		Proposer:   validators[0].Signer,
		StartBlock: 26657,
		EndBlock:   30000,
		ChainId:    "testChainParams.ChainParams.BorChainId",
		Seed:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		SeedAuthor: "val1Addr.Hex()",
	}

	txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)
	var txBytesCmt cmtTypes.Tx = txBytes

	extCommitBytes, _, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys)

	app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes})

	finalizeReq := abci.RequestFinalizeBlock{
		Txs:    [][]byte{extCommitBytes, txBytes},
		Height: 3,
	}
	_, err = app.PreBlocker(ctx, &finalizeReq)
	require.NoError(t, err)

}

func TestSidetxsHappyPath(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// logIndex := uint64(10)
	blockNumber := uint64(599)

	_, _, addr2 := testdata.KeyTestPubAddr()

	txReceipt := &ethTypes.Receipt{
		BlockNumber: new(big.Int).SetUint64(blockNumber),
	}

	event := &stakinginfo.StakinginfoTopUpFee{
		User: common.Address(sdk.AccAddress(addr2.String())),
		Fee:  new(big.Int).SetUint64(1),
	}

	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.
		On("GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, nil)

	mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.AnythingOfType("int64")).Return(txReceipt, nil)
	mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)
	app.TopupKeeper = topupKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.BankKeeper,
		app.ChainManagerKeeper,
		mockCaller,
	)
	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		app.AppCodec(),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)),
		mockCaller,
	)
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

	propBytes := common.FromHex(validators[0].Signer)
	propAddr := sdk.AccAddress(propBytes)
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)
	require.NoError(t,
		testutil.FundAccount(ctx, app.BankKeeper, propAddr,
			sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
		),
	)

	coins, _ := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})

	testCases := []struct {
		name string
		msg  sdk.Msg
	}{
		{
			name: "bor [MsgProposeSpan]] happy path",
			msg: &borTypes.MsgProposeSpan{
				SpanId:     2,
				Proposer:   validators[0].Signer,
				StartBlock: 26657,
				EndBlock:   30000,
				ChainId:    "testChainParams.ChainParams.BorChainId",
				Seed:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
				SeedAuthor: "val1Addr.Hex()",
			},
		},
		// {
		// 	name: "Clerk Module Happy Path",
		// 	msg: func() *clerkTypes.MsgEventRecord {
		// 		rec := clerkTypes.NewMsgEventRecord(
		// 			validators[0].Signer,
		// 			TxHash1,
		// 			1,
		// 			50,
		// 			1,
		// 			propAddr,
		// 			make([]byte, 0),
		// 			"0",
		// 		)
		// 		return &rec
		// 	}(),
		// },
		{
			name: "topup [MsgProposeSpan]] happy path",
			msg: func() *topUpTypes.MsgTopupTx {
				rec := topUpTypes.NewMsgTopupTx(
					validators[0].Signer,
					validators[0].Signer,
					coins.AmountOf(authTypes.FeeToken),
					[]byte(TxHash1),
					1,
					1,
				)
				return rec
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			txBytes, err := buildSignedTx(tc.msg, validators[0].Signer, ctx, priv, app)
			var txBytesCmt cmtTypes.Tx = txBytes

			extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys)
			_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
				Height: 3,
				Txs:    [][]byte{extCommitBytes, txBytes},
			})
			require.NoError(t, err)

			// Prepare/Process proposal
			reqPrep := &abci.RequestPrepareProposal{
				Txs:             [][]byte{txBytes},
				MaxTxBytes:      1_000_000,
				LocalLastCommit: *extCommit,
				ProposerAddress: common.FromHex(validators[0].Signer),
				Height:          3,
			}

			_, err = app.PrepareProposal(reqPrep)
			require.NoError(t, err)

			respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
			require.NoError(t, err)
			require.NotEmpty(t, respPrep.Txs)

			reqExtend := abci.RequestExtendVote{
				Txs:    [][]byte{extCommitBytes, txBytes},
				Hash:   []byte("test-hash"),
				Height: 3,
			}
			respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
			require.NoError(t, err)
			require.NotNil(t, respExtend.VoteExtension)
			mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

			app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes})

			extCommitBytes2, _, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys)

			finalizeReq := abci.RequestFinalizeBlock{
				Txs:    [][]byte{extCommitBytes2, txBytes},
				Height: 3,
			}
			_, err = app.PreBlocker(ctx, &finalizeReq)
			require.NoError(t, err)

		})
	}

}

func TestAllUnhappyPathBorSideTxs(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	valSet, vals := genTestValidators()

	mockCaller := new(helpermocks.IContractCaller)

	app.ChainManagerKeeper = chainmanagerKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.TopupKeeper = topupKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.BankKeeper,
		app.ChainManagerKeeper,
		mockCaller,
	)
	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		app.AppCodec(),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)),
		mockCaller,
	)
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
		mockCaller,
	)
	app.BorKeeper.SetContractCaller(mockCaller)
	// app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	// ——— wire up side-msg server ———
	borKeeper.NewSideMsgServerImpl(&app.BorKeeper)
	app.sideTxCfg = sidetxs.NewSideTxConfigurator()
	app.RegisterSideMsgServices(app.sideTxCfg)

	// ◀── **move this here**, after your server is registered:
	app.BorKeeper.SetContractCaller(mockCaller)

	propBytes := common.FromHex(validators[0].Signer)
	propAddr := sdk.AccAddress(propBytes)
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)
	require.NoError(t,
		testutil.FundAccount(ctx, app.BankKeeper, propAddr,
			sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
		),
	)
	spans := []borTypes.Span{
		{
			Id:                0,
			StartBlock:        0,
			EndBlock:          256,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                1,
			StartBlock:        257,
			EndBlock:          6656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                2,
			StartBlock:        6657,
			EndBlock:          16656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                3,
			StartBlock:        16657,
			EndBlock:          26656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
	}

	seedBlock1 := spans[3].EndBlock
	val1Addr := common.HexToAddress(vals[0].GetOperator())
	blockHeader1 := ethTypes.Header{Number: big.NewInt(int64(seedBlock1))}
	blockHash1 := blockHeader1.Hash()

	mockCaller.On("GetBorChainBlockAuthor", mock.Anything).Return(&val1Addr, nil).Times(100)

	mockCaller.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(&blockHeader1, nil).Times(100)
	mockCaller.
		On("GetBorChainBlocksInBatch", mock.Anything, mock.Anything, mock.Anything).
		Return([]*ethTypes.Header{&blockHeader1}, nil).Times(100)

	for _, span := range spans {
		err := app.BorKeeper.AddNewSpan(ctx, &span)
		require.NoError(t, err)
		err = app.BorKeeper.StoreSeedProducer(ctx, span.Id, &val1Addr)
	}
	testChainParams := chainmanagertypes.DefaultParams()

	t.Run("seed mismatch", func(t *testing.T) {

		msg := &borTypes.MsgProposeSpan{
			SpanId:     4,
			Proposer:   validators[0].Signer,
			StartBlock: 26657,
			EndBlock:   30000,
			ChainId:    testChainParams.ChainParams.BorChainId,
			Seed:       []byte("someWrongSeed"),
		}

		txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: 3,
			Txs:    [][]byte{extCommitBytes, txBytes},
		})
		require.NoError(t, err)

		// Prepare/Process proposal
		reqPrep := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1_000_000,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
			Height:          3,
		}

		_, err = app.PrepareProposal(reqPrep)
		require.NoError(t, err)

		respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
		require.NoError(t, err)
		require.NotEmpty(t, respPrep.Txs)

		reqExtend := abci.RequestExtendVote{
			Txs:    [][]byte{extCommitBytes, txBytes},
			Hash:   []byte("test-hash"),
			Height: 3,
		}
		respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
		require.NoError(t, err)
		require.NotNil(t, respExtend.VoteExtension)
		mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	})

	t.Run("span is not in turn", func(t *testing.T) {
		msg := &borTypes.MsgProposeSpan{
			SpanId:     4,
			Proposer:   val1Addr.String(),
			StartBlock: 26657,
			EndBlock:   30000,
			ChainId:    testChainParams.ChainParams.BorChainId,
			Seed:       blockHash1.Bytes(),
		}

		txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: 3,
			Txs:    [][]byte{extCommitBytes, txBytes},
		})
		require.NoError(t, err)

		// Prepare/Process proposal
		reqPrep := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1_000_000,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
			Height:          3,
		}

		_, err = app.PrepareProposal(reqPrep)
		require.NoError(t, err)

		respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
		require.NoError(t, err)
		require.NotEmpty(t, respPrep.Txs)

		reqExtend := abci.RequestExtendVote{
			Txs:    [][]byte{extCommitBytes, txBytes},
			Hash:   []byte("test-hash"),
			Height: 3,
		}
		respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
		require.NoError(t, err)
		require.NotNil(t, respExtend.VoteExtension)
		mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	})

	t.Run("correct span is proposed", func(t *testing.T) {
		msg := &borTypes.MsgProposeSpan{
			SpanId:     4,
			Proposer:   val1Addr.String(),
			StartBlock: 26657,
			EndBlock:   30000,
			ChainId:    testChainParams.ChainParams.BorChainId,
			Seed:       blockHash1.Bytes(),
		}

		txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: 3,
			Txs:    [][]byte{extCommitBytes, txBytes},
		})
		require.NoError(t, err)

		// Prepare/Process proposal
		reqPrep := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1_000_000,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
			Height:          3,
		}

		_, err = app.PrepareProposal(reqPrep)
		require.NoError(t, err)

		respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
		require.NoError(t, err)
		require.NotEmpty(t, respPrep.Txs)

		reqExtend := abci.RequestExtendVote{
			Txs:    [][]byte{extCommitBytes, txBytes},
			Hash:   []byte("test-hash"),
			Height: 3,
		}
		respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
		require.NoError(t, err)
		require.NotNil(t, respExtend.VoteExtension)
		mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	})

}

func TestSomething(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Create a checkpoint message
	msg := &borTypes.MsgProposeSpan{
		SpanId:     2,
		Proposer:   validators[0].Signer,
		StartBlock: 26657,
		EndBlock:   30000,
		ChainId:    "testChainParams.ChainParams.BorChainId",
		Seed:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		SeedAuthor: "val1Addr.Hex()",
	}

	txBytes, err := buildSignedTx2(msg, ctx, priv, &app)
	var txBytesCmt cmtTypes.Tx = txBytes

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys)

	app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes})
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 3,
		Txs:    [][]byte{extCommitBytes, txBytes},
	})
	require.NoError(t, err)

	// Prepare/Process proposal
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          3,
	}

	_, err = app.PrepareProposal(reqPrep)
	require.NoError(t, err)

	respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)
}

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

	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		app.AppCodec(),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)),
		mockCaller,
	)
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

	// Prepare proposer account
	propBytes := common.FromHex(validators[0].Signer)
	propAddr := sdk.AccAddress(propBytes)
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)
	require.NoError(t,
		testutil.FundAccount(ctx, app.BankKeeper, propAddr,
			sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
		),
	)

	// Build and sign the tx
	txConfig := authtx.NewTxConfig(app.AppCodec(), authtx.DefaultSignModes)
	defaultSignMode, err := authsigning.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())
	require.NoError(t, err)
	app.SetTxDecoder(txConfig.TxDecoder())

	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	require.NoError(t, txBuilder.SetMsgs(msg))

	sigV2 := signing.SignatureV2{PubKey: priv.PubKey(), Data: &signing.SingleSignatureData{
		SignMode:  defaultSignMode,
		Signature: nil,
	}, Sequence: 0}
	require.NoError(t, txBuilder.SetSignatures(sigV2))

	signerData := authsigning.SignerData{
		ChainID:       "",
		AccountNumber: 1337,
		Sequence:      0,
		PubKey:        priv.PubKey(),
	}
	sigV2, err = tx.SignWithPrivKey(context.TODO(), defaultSignMode, signerData,
		txBuilder, priv, txConfig, 0)
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(sigV2))

	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	// Build a fake commit for height=3
	cmtPubKey, err := validators[0].CmtConsPublicKey()
	require.NoError(t, err)
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
		Votes: []abci.ExtendedVoteInfo{voteInfo1},
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 3,
		Txs:    [][]byte{extCommitBytes, txBytes},
	})
	require.NoError(t, err)
	// _, err = app.Commit()
	// require.NoError(t, err)

	// Prepare/Process proposal
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          3,
	}

	_, err = app.PrepareProposal(reqPrep)
	require.NoError(t, err)

	respPrep, err := app.NewPrepareProposalHandler()(ctx, reqPrep)
	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)

	// ---------------------- Prepare Proposal for fake No extCommitInfo

	reqProcess := &abci.RequestProcessProposal{
		Txs:                respPrep.Txs,
		Height:             3,
		ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
	}
	respProc, err := app.NewProcessProposalHandler()(ctx, reqProcess)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, respProc.Status)

	// ---------------------------- No transaction test case PrepareProposal --------------------------------------
	reqPrepNoTx := &abci.RequestProcessProposal{
		Txs:                [][]byte{},
		ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
		Height:             3,
	}
	respPrepNoTx, err := app.NewProcessProposalHandler()(ctx, reqPrepNoTx)
	require.NoError(t, err) // handler itself should not error
	require.Equal(t,
		abci.ResponseProcessProposal_REJECT,
		respPrepNoTx.Status,
		"expected a REJECT status when no txs are provided",
	)

	// ---------------------------- No transaction test case PrepareProposal --------------------------------------

	// ---------------------------- Excommit info round mismatch  --------------------------------------

	req := &abci.RequestProcessProposal{
		Txs: [][]byte{
			{0x01, 0x02, 0x03},
		},
		Height:             3,
		ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
	}

	respExCommit, err := app.NewProcessProposalHandler()(ctx, req)
	require.NoError(t, err, "handler itself should not error")
	require.Equal(
		t,
		abci.ResponseProcessProposal_REJECT,
		respExCommit.Status,
		"expected REJECT when ExtendedCommitInfo.Unmarshal fails",
	)

	// ---------------------------- Excommit info round mismatch  --------------------------------------
	// ---------------------------- Excommit Round mismatch  --------------------------------------

	reqExcommitRoundMismatch := &abci.RequestProcessProposal{
		Txs:                respPrep.Txs,
		Height:             3,
		ProposedLastCommit: abci.CommitInfo{Round: 30},
	}
	respExCommitRountMismatch, err := app.NewProcessProposalHandler()(ctx, reqExcommitRoundMismatch)
	require.NoError(t, err, "handler itself should not error")
	require.Equal(
		t,
		abci.ResponseProcessProposal_REJECT,
		respExCommitRountMismatch.Status,
		"expected REJECT when ExtendedCommitInfo Round mismatches",
	)

	// ---------------------------- Excommit Round mismatch  --------------------------------------

	// ------------------------------- Bad transaction ------------------------------------------
	badTx := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	reqBadTx := &abci.RequestProcessProposal{
		Txs: [][]byte{
			respPrep.Txs[0], // valid commit
			badTx,           // decode error here
		},
		Height:             3,
		ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
	}
	respBadTx, err := app.NewProcessProposalHandler()(ctx, reqBadTx)
	require.NoError(t, err, "handler itself should not error")
	require.Equal(
		t,
		abci.ResponseProcessProposal_REJECT,
		respBadTx.Status,
		"expected REJECT when Transaction decoding fails",
	)

	// ------------------------------------------------------------------------------------------
	// --------------------------------- Process Proposal Verify --------------------------------

	// msgBadTx := &checkpointTypes.MsgCheckpoint{
	// 	Proposer:        validators[0].Signer,
	// 	StartBlock:      1,
	// 	EndBlock:        2,
	// 	RootHash:        common.Hex2Bytes("aa"),
	// 	AccountRootHash: common.Hex2Bytes("bb"),
	// 	BorChainId:      "test",
	// }
	// txBuilderBadTx := txConfig.NewTxBuilder()
	// require.NoError(t, txBuilderBadTx.SetMsgs(msgBadTx))
	// require.NoError(t, txBuilderBadTx.SetSignatures(sigV2))

	// txBytesBadTx, err := txConfig.TxEncoder()(txBuilderBadTx.GetTx())
	// require.NoError(t, err)

	// reqBadTxMsg := &abci.RequestProcessProposal{
	// 	Txs: [][]byte{
	// 		respPrep.Txs[0],
	// 		txBytesBadTx, // decode error here
	// 	},
	// 	Height:             3,
	// 	ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
	// }

	// respBadTxMsg, err := app.NewProcessProposalHandler()(ctx, reqBadTxMsg)
	// require.NoError(t, err, "handler itself should not error")
	// require.Equal(
	// 	t,
	// 	abci.ResponseProcessProposal_REJECT,
	// 	respBadTxMsg.Status,
	// 	"expected REJECT when Transaction decoding fails",
	// )

	// ------------------------------------------------------------------------------------------

	// ExtendVote
	reqExtend := abci.RequestExtendVote{
		Txs:    respPrep.Txs,
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)
	mockCaller.AssertCalled(t, "GetBorChainBlocksInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	// ------------------------------- Extend Vote Handler throws error when Unmarshalling of extCommit fails------------------------
	reqExtendUnmarshalFail := abci.RequestExtendVote{
		Txs: [][]byte{
			{0x01, 0x02, 0x03},
		},
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtendUnmarshalFail, err := app.ExtendVoteHandler()(ctx, &reqExtendUnmarshalFail)
	require.Nil(t, respExtendUnmarshalFail, "should not return a response when TxDecode fails")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error occurred while decoding ExtendedCommitInfo, they should have be encoded in the beginning of txs slice")
	// ------------------------------- Extend Vote Handler throws error when Unmarshalling of extCommit fails------------------------

	reqExtendTxFail := abci.RequestExtendVote{
		Txs: [][]byte{
			respPrep.Txs[0],
			{0x01, 0x02, 0x03},
		},
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtendTxFail, err := app.ExtendVoteHandler()(ctx, &reqExtendTxFail)
	require.Nil(t, respExtendTxFail, "should not return a response when TxDecode fails")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error occurred while decoding tx bytes in ExtendVoteHandler")

	// ------------------------------- Extend Vote Handler throws error when Unmarshalling of extCommit fails------------------------

	// VerifyVoteExtension — **here’s the fix: pass the consensus address** 🎉
	reqVerify := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address, // <<< use the real consensus addr
		Height:             3,
		Hash:               []byte("test-hash"),
	}
	respVerify, err := app.VerifyVoteExtensionHandler()(ctx, &reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, respVerify.Status)

	// --------------------------------------------------------------------------------------
	// Test VerifyVoteExtension handler: unmarshal failure
	badReq := abci.RequestVerifyVoteExtension{
		VoteExtension:      []byte{0x01, 0x02, 0x03},    // invalid protobuf
		NonRpVoteExtension: respExtend.NonRpExtension,   // whatever the handler expects
		ValidatorAddress:   voteInfo1.Validator.Address, // real consensus addr
		Height:             3,
		Hash:               []byte("test-hash"),
	}
	respBad, err := app.VerifyVoteExtensionHandler()(ctx, &badReq)
	require.NoError(t, err, "handler should swallow unmarshal errors and return a response")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_REJECT,
		respBad.Status,
		"expected REJECT when VoteExtension protobuf unmarshal fails",
	)
	// --------------------------------------------------------------------------------------

	// ————————————— height-mismatch branch —————————————
	badReqHeight := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address,
		Height:             reqExtend.Height + 1, // deliberately wrong (was 3)
		Hash:               []byte("test-hash"),
	}
	respBadHeight, err := app.VerifyVoteExtensionHandler()(ctx, &badReqHeight)
	require.NoError(t, err, "handler should swallow height-mismatch and return a response")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_REJECT,
		respBadHeight.Status,
		"expected REJECT when req.Height (%d) != VoteExtension.Height (%d)",
		badReqHeight.Height, reqExtend.Height,
	)
	// ————————————————————————————————————————————————————————

	// ---------------------- block‐hash mismatch branch ----------------------
	badReqHash := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address,
		Height:             reqExtend.Height,     // same as before
		Hash:               []byte("wrong-hash"), // deliberately different
	}
	respBadHash, err := app.VerifyVoteExtensionHandler()(ctx, &badReqHash)
	require.NoError(t, err, "handler should swallow hash‐mismatch and return a response")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_REJECT,
		respBadHash.Status,
		"expected REJECT when req.Hash (%x) != blockHash in VoteExtension", badReqHash.Hash,
	)
	// -------------------------------------------------------------------------

	// ---------------------- side-tx responses validation failure branch ----------------------
	// Unmarshal the good VoteExtension so we can mutate its SideTxResponses
	// var badExt sidetxs.VoteExtension
	// require.NoError(t, proto.Unmarshal(respExtend.VoteExtension, &badExt), "should unmarshal existing VoteExtension")

	// // Corrupt SideTxResponses to trigger a validation error
	// badExt.SideTxResponses = nil // or set to a slice with invalid entries

	// badVoteBytes, err := proto.Marshal(&badExt)
	// require.NoError(t, err, "should marshal corrupted VoteExtension")
	fakeExt := &sidetxs.VoteExtension{
		BlockHash:       []byte("whatever"), // keep or change as needed
		Height:          reqExtend.Height,   // so height‐check passes
		SideTxResponses: nil,                // nil to force validateSideTxResponses error
	}
	fakeBz, err := gogoproto.Marshal(fakeExt)
	require.NoError(t, err, "gogo-Marshal should work on a Gogo type")

	badReqSide := abci.RequestVerifyVoteExtension{
		VoteExtension:      fakeBz,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address,
		Height:             reqExtend.Height,
		Hash:               []byte("test-hash"),
	}

	respBadSide, err := app.VerifyVoteExtensionHandler()(ctx, &badReqSide)
	require.NoError(t, err, "handler should swallow side-tx validation errors and return a response")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_REJECT,
		respBadSide.Status,
		"expected REJECT when validateSideTxResponses returns an error",
	)
	// -----------------------------------------------------------------------------------------

	// ------------------ height-mismatch branch ------------------
	fakeExtHeight := &sidetxs.VoteExtension{
		BlockHash:            []byte("whatever"),     // you can reuse the real block hash or respExtend data
		Height:               reqExtend.Height + 100, // deliberately off by +1
		SideTxResponses:      nil,                    // you can copy the real side-txs so only height trips
		MilestoneProposition: nil,                    // optional
	}

	fakeExtBzHeight, err := gogoproto.Marshal(fakeExtHeight)
	require.NoError(t, err, "gogo‐Marshal should succeed on our fake extension")
	badReqSideHeight := abci.RequestVerifyVoteExtension{
		VoteExtension:      fakeExtBzHeight,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address,
		Height:             reqExtend.Height,
		Hash:               []byte("test-hash"),
	}
	respBadHeight, err = app.VerifyVoteExtensionHandler()(ctx, &badReqSideHeight)
	require.NoError(t, err, "handler should swallow height-mismatch and return a response")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_REJECT,
		respBadHeight.Status,
		"expected REJECT when req.Height (%d) != VoteExtension.Height (%d)",
		badReqHeight.Height, fakeExtHeight.Height,
	)

	badSide := []sidetxs.SideTxResponse{
		{
			// pick any txHash—this is what validateSideTxResponses will return
			TxHash: []byte("deadbeef"),
			// leave other fields nil/zero so validation fails
		},
	}
	var goodExt sidetxs.VoteExtension
	require.NoError(t,
		gogoproto.Unmarshal(respExtend.VoteExtension, &goodExt),
		"should unmarshal the real VoteExtension",
	)
	// 3) Build a fake VoteExtension with the bad side‐txs
	fakeExt2 := &sidetxs.VoteExtension{
		BlockHash:       goodExt.BlockHash,
		Height:          goodExt.Height, // keep height correct
		SideTxResponses: badSide,        // invalid payload
		// MilestoneProposition: nil,        // optional
	}
	fakeBz2, err := gogoproto.Marshal(fakeExt2)
	require.NoError(t, err, "gogo‐Marshal should succeed")

	// 5) Call the verify handler
	badReqSide = abci.RequestVerifyVoteExtension{
		VoteExtension:      fakeBz2,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address,
		Height:             reqExtend.Height,
		Hash:               []byte("test-hash"),
	}
	respBadSide, err = app.VerifyVoteExtensionHandler()(ctx, &badReqSide)
	require.NoError(t, err, "handler should swallow side‐tx validation errors")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_REJECT,
		respBadSide.Status,
		"expected REJECT when validateSideTxResponses returns an error and txHash=%X",
		badSide[0].TxHash,
	)

	// ---------------------- Non-RP extension validation failure ----------------------
	badReqNonRp := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,       // use the good extension
		NonRpVoteExtension: []byte{0x01, 0x02, 0x03, 0xFF}, // invalid bytes to force an error
		ValidatorAddress:   voteInfo1.Validator.Address,    // correct consensus addr
		Height:             reqExtend.Height,               // keep height/hash correct
		Hash:               []byte("test-hash"),
	}

	respNonRp, err := app.VerifyVoteExtensionHandler()(ctx, &badReqNonRp)
	require.NoError(t, err, "handler should swallow non-RP validation errors and continue")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_ACCEPT,
		respNonRp.Status,
		"expected ACCEPT even if ValidateNonRpVoteExtension returns an error",
	)
	fmt.Println("finally!")

	// Test FinalizeBlock handler
	finalizeReq := abci.RequestFinalizeBlock{
		Txs:    [][]byte{extCommitBytes, txBytes},
		Height: 3,
	}
	_, err = app.PreBlocker(ctx, &finalizeReq)
	require.NoError(t, err)

	// _, err = app.Commit()
	// require.NoError(t, err)

	//-------------------------------- bor Preblock happy path ---------------------------------------------
	// flag_toget
	msgBor := &borTypes.MsgProposeSpan{
		// SpanId:     2,
		Proposer:   validators[0].Signer,
		StartBlock: 26657,
		EndBlock:   30000,
		ChainId:    "testChainParams.ChainParams.BorChainId",
		Seed:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		SeedAuthor: "val1Addr.Hex()",
	}

	require.NoError(t, txBuilder.SetMsgs(msgBor))
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(sigV2))

	txBytesBor, err := txConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytesBor})
	fmt.Println("#################################################################")
	fmt.Println(txBytesBor)
	fmt.Println(app.StakeKeeper.GetLastBlockTxs(ctx))

	// _, err = app.Commit()
	// require.NoError(t, err)
	var txBytesBorcmt cmtTypes.Tx = txBytesBor

	voteInfo2 := setupExtendedVoteInfoWithNonRp(
		t,
		cmtproto.BlockIDFlagCommit,
		txBytesBorcmt.Hash(),
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal1,
		validatorPrivKeys[0],
		2,
		app,
		cmtPubKey.GetEd25519(),
	)

	extCommit2 := &abci.ExtendedCommitInfo{
		Round: 1,
		Votes: []abci.ExtendedVoteInfo{voteInfo2},
	}
	extCommitBytes2, err := extCommit2.Marshal()
	require.NoError(t, err)

	// ---------------------- unhappy path of bor side transaction ------------------------

	// Test FinalizeBlock handler
	finalizeReqBorSidetx := abci.RequestFinalizeBlock{
		Txs:    [][]byte{extCommitBytes2, txBytesBor},
		Height: 3,
	}
	_, err = app.PreBlocker(ctx, &finalizeReqBorSidetx)
	require.NoError(t, err)

	// --------------------- bor unhappy path 1 ----------------------------------------
	// Commit the block
	// -------------------------- PreBlocker for cleck module ------------------

	msgClerk := clerkTypes.NewMsgEventRecord(
		validators[0].Signer,
		TxHash1,
		1,
		50,
		1,
		propAddr,
		make([]byte, 0),
		"0",
	)
	require.NoError(t, txBuilder.SetMsgs(&msgClerk))
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(sigV2))

	txBytesClerk, err := txConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytesClerk})
	fmt.Println("#################################################################")
	fmt.Println(txBytesBor)
	fmt.Println(app.StakeKeeper.GetLastBlockTxs(ctx))

	// _, err = app.Commit()
	// require.NoError(t, err)
	var txBytesClerkcmt cmtTypes.Tx = txBytesBor

	voteInfo3 := setupExtendedVoteInfoWithNonRp(
		t,
		cmtproto.BlockIDFlagCommit,
		txBytesClerkcmt.Hash(),
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal1,
		validatorPrivKeys[0],
		2,
		app,
		cmtPubKey.GetEd25519(),
	)

	extCommit3 := &abci.ExtendedCommitInfo{
		Round: 1,
		Votes: []abci.ExtendedVoteInfo{voteInfo3},
	}
	extCommitBytes3, err := extCommit3.Marshal()
	require.NoError(t, err)

	// Test FinalizeBlock handler
	finalizeReqClerkSidetx := abci.RequestFinalizeBlock{
		Txs:    [][]byte{extCommitBytes3, txBytesClerk},
		Height: 3,
	}
	_, err = app.PreBlocker(ctx, &finalizeReqClerkSidetx)
	require.NoError(t, err)

	//----------------------------------------------------------------

	// -------------------------- Happy path for topup module ------------------
	coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
	msgTopUp := topUpTypes.NewMsgTopupTx(
		validators[0].Signer,
		validators[0].Signer,
		coins.AmountOf(authTypes.FeeToken),
		[]byte(TxHash1),
		1,
		1,
	)

	require.NoError(t, txBuilder.SetMsgs(msgTopUp))
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(sigV2))

	txBytesTopUp, err := txConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytesTopUp})

	_, err = app.Commit()
	require.NoError(t, err)
	var txBytesTopUpcmt cmtTypes.Tx = txBytesTopUp

	voteInfo4 := setupExtendedVoteInfoWithNonRp(
		t,
		cmtproto.BlockIDFlagCommit,
		txBytesTopUpcmt.Hash(),
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal1,
		validatorPrivKeys[0],
		2,
		app,
		cmtPubKey.GetEd25519(),
	)

	extCommit4 := &abci.ExtendedCommitInfo{
		Round: 1,
		Votes: []abci.ExtendedVoteInfo{voteInfo4},
	}
	extCommitBytes4, err := extCommit4.Marshal()
	require.NoError(t, err)

	// Test FinalizeBlock handler
	finalizeReqTopUpSidetx := abci.RequestFinalizeBlock{
		Txs:    [][]byte{extCommitBytes4, txBytesTopUp},
		Height: 3,
	}
	_, err = app.PreBlocker(ctx, &finalizeReqTopUpSidetx)
	require.NoError(t, err)

	//---------------------------------------------------------------------------
	//--------------------happy path for

}

var defaultFeeAmount = big.NewInt(10).Exp(big.NewInt(10), big.NewInt(15), nil).Int64()
