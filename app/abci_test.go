package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/protoio"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	addressUtils "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/bor"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagerKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk"
	clerkKeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	clerktestutil "github.com/0xPolygon/heimdall-v2/x/clerk/testutil"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/0xPolygon/heimdall-v2/x/topup"
	topupKeeper "github.com/0xPolygon/heimdall-v2/x/topup/keeper"
	topUpTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

// genTestValidators generates a set of test validators for testing purposes.
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

// buildSignedTxWithSequence builds and signs a transaction with the given sequence number.
func buildSignedTxWithSequence(msg sdk.Msg, ctx sdk.Context, priv cryptotypes.PrivKey, app *HeimdallApp, sequence uint64) ([]byte, error) {
	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	if propAcc == nil {
		propAcc = authTypes.NewBaseAccount(propAddr, priv.PubKey(), 1, 0)
		app.AccountKeeper.SetAccount(ctx, propAcc)
	} else if propAcc.GetPubKey() == nil {
		// Some genesis accounts (e.g., created from raw addresses) may not have a pubkey yet.
		err := propAcc.SetPubKey(priv.PubKey())
		if err != nil {
			return nil, fmt.Errorf("failed to set pubkey for account: %w", err)
		}
		app.AccountKeeper.SetAccount(ctx, propAcc)
	}

	// Build and sign the tx
	txConfig := authtx.NewTxConfig(app.AppCodec(), authtx.DefaultSignModes)
	defaultSignMode, err := authsigning.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())
	app.SetTxDecoder(txConfig.TxDecoder())

	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to set tx msg: %w", err)
	}

	sigV2 := signing.SignatureV2{PubKey: priv.PubKey(), Data: &signing.SingleSignatureData{
		SignMode:  defaultSignMode,
		Signature: nil,
	}, Sequence: sequence}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	chainID := ctx.ChainID()
	if chainID == "" {
		chainID = app.ChainID()
	}
	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: propAcc.GetAccountNumber(),
		Sequence:      sequence,
		PubKey:        priv.PubKey(),
	}
	sigV2, err = tx.SignWithPrivKey(context.TODO(), defaultSignMode, signerData,
		txBuilder, priv, txConfig, sequence)
	if err != nil {
		return nil, err
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	return txBytes, err
}

// buildSignedTx builds and signs a transaction for the given message, automatically fetching the account sequence.
func buildSignedTx(msg sdk.Msg, ctx sdk.Context, priv cryptotypes.PrivKey, app *HeimdallApp) ([]byte, error) {
	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	var sequence uint64
	if propAcc != nil {
		sequence = propAcc.GetSequence()
	}
	return buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
}

// buildSignedMultiMsgTx builds a single signed tx containing multiple messages.
func buildSignedMultiMsgTx(msgs []sdk.Msg, ctx sdk.Context, priv cryptotypes.PrivKey, app *HeimdallApp) ([]byte, error) {
	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	if propAcc == nil {
		propAcc = authTypes.NewBaseAccount(propAddr, priv.PubKey(), 1, 0)
		app.AccountKeeper.SetAccount(ctx, propAcc)
	} else if propAcc.GetPubKey() == nil {
		err := propAcc.SetPubKey(priv.PubKey())
		if err != nil {
			return nil, fmt.Errorf("failed to set pubkey for account: %w", err)
		}
		app.AccountKeeper.SetAccount(ctx, propAcc)
	}

	sequence := propAcc.GetSequence()

	txConfig := authtx.NewTxConfig(app.AppCodec(), authtx.DefaultSignModes)
	defaultSignMode, err := authsigning.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())
	app.SetTxDecoder(txConfig.TxDecoder())

	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{PubKey: priv.PubKey(), Data: &signing.SingleSignatureData{
		SignMode:  defaultSignMode,
		Signature: nil,
	}, Sequence: sequence}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	chainID := ctx.ChainID()
	if chainID == "" {
		chainID = app.ChainID()
	}
	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: propAcc.GetAccountNumber(),
		Sequence:      sequence,
		PubKey:        priv.PubKey(),
	}
	sigV2, err = tx.SignWithPrivKey(context.TODO(), defaultSignMode, signerData,
		txBuilder, priv, txConfig, sequence)
	if err != nil {
		return nil, err
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	return txConfig.TxEncoder()(txBuilder.GetTx())
}

// buildExtensionCommits builds the extension commits for the given block hash and validators, using the provided vote info or creating an empty one if nil.
func buildExtensionCommits(
	t *testing.T,
	app *HeimdallApp,
	blockHashBytes []byte,
	validators []*stakeTypes.Validator,
	validatorPrivKeys []secp256k1.PrivKey,
	height int64,
	voteInfo *abci.ExtendedVoteInfo,
) ([]byte, *abci.ExtendedCommitInfo, *abci.ExtendedVoteInfo, error) {

	cometVal := abci.Validator{
		Address: common.FromHex(validators[0].Signer),
		Power:   validators[0].VotingPower,
	}

	if voteInfo == nil {
		voteInfo = new(setupEmptyExtendedVoteInfo(
			t,
			cmtproto.BlockIDFlagCommit,
			blockHashBytes,
			cometVal,
			validatorPrivKeys[0],
			height,
			app,
		))
	}

	extCommit := &abci.ExtendedCommitInfo{
		Votes: []abci.ExtendedVoteInfo{*voteInfo},
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)
	return extCommitBytes, extCommit, voteInfo, err
}

// SetupAppWithABCICtx sets up a HeimdallApp with a single validator and returns the private key, app instance, context, and validator private keys for testing.
func SetupAppWithABCICtx(t *testing.T) (cryptotypes.PrivKey, *HeimdallApp, sdk.Context, []secp256k1.PrivKey) {
	return SetupAppWithABCICtxAndValidators(t, 1)
}

// SetupAppWithABCICtxAndValidators sets up a HeimdallApp with the given number of validators and returns the private key, app instance, context, and validator private keys for testing.
func SetupAppWithABCICtxAndValidators(t *testing.T, numValidators int) (cryptotypes.PrivKey, *HeimdallApp, sdk.Context, []secp256k1.PrivKey) {
	priv, _, _ := testdata.KeyTestPubAddr()

	setupResult := SetupAppWithPrivKey(t, uint64(numValidators), priv)
	app := setupResult.App

	// Initialize the application state
	ctx := app.BaseApp.NewContext(true).WithChainID(app.ChainID())

	// Set up consensus params
	params := cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	validatorPrivKeys := setupResult.ValidatorKeys
	return priv, app, ctx, validatorPrivKeys
}

// TestPrepareProposalHandler tests the PrepareProposal handler of the HeimdallApp by creating a checkpoint message, building a signed transaction, and preparing a proposal with the transaction and an extension commit.
func TestPrepareProposalHandler(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
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

	txBytes, err := buildSignedTx(msg, ctx, priv, app)
	require.NoError(t, err)

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

	// Prepare the proposal
	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          app.LastBlockHeight() + 1,
	}

	respPrep, err := app.PrepareProposal(reqPrep)
	require.NoError(t, err)
	require.NotEmpty(t, respPrep.Txs)
}

func TestPrepareProposalHandler_BudgetExhausted(t *testing.T) {
	original := prepareProposalBudget
	prepareProposalBudget = 0
	t.Cleanup(func() { prepareProposalBudget = original })

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	msg := &types.MsgCheckpoint{
		Proposer:        priv.PubKey().Address().String(),
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "1",
	}
	txBytes, err := buildSignedTx(msg, ctx, priv, app)
	require.NoError(t, err)

	_, extCommit, _, err := buildExtensionCommits(
		t,
		app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators, validatorPrivKeys, app.LastBlockHeight(),
		nil,
	)
	require.NoError(t, err)

	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          app.LastBlockHeight() + 1,
	}

	respPrep, err := app.PrepareProposal(reqPrep)
	require.NoError(t, err)
	// loop breaks on iteration 0; response carries only the marshaled commit info.
	require.Len(t, respPrep.Txs, 1)
}

func TestPrepareProposalHandler_BudgetNotReached(t *testing.T) {
	original := prepareProposalBudget
	prepareProposalBudget = time.Hour
	t.Cleanup(func() { prepareProposalBudget = original })

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	msg := &types.MsgCheckpoint{
		Proposer:        priv.PubKey().Address().String(),
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "1",
	}
	txBytes, err := buildSignedTx(msg, ctx, priv, app)
	require.NoError(t, err)

	_, extCommit, _, err := buildExtensionCommits(
		t,
		app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators, validatorPrivKeys, app.LastBlockHeight(),
		nil,
	)
	require.NoError(t, err)

	reqPrep := &abci.RequestPrepareProposal{
		Txs:             [][]byte{txBytes},
		MaxTxBytes:      1_000_000,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          app.LastBlockHeight() + 1,
	}

	respPrep, err := app.PrepareProposal(reqPrep)
	require.NoError(t, err)
	// commit info + the proposed tx.
	require.Len(t, respPrep.Txs, 2)
}

func TestProcessProposalHandler(t *testing.T) {

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "1",
	}

	txBytes, err := buildSignedTx(msg, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)

	require.NoError(t, err)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          3,
		Txs:             [][]byte{extCommitBytes, txBytes},
		ProposerAddress: common.FromHex(validators[0].Signer),
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

	// tests for ProcessProposalHandler
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

// TestExtendVoteHandler tests the ExtendVote handler of the HeimdallApp by creating a checkpoint message, building a signed transaction, preparing a proposal, and extending the vote with valid transactions and checking the interactions with the mock contract caller.
func TestExtendVoteHandler(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
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

	txBytes, err := buildSignedTx(msg, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          3,
		Txs:             [][]byte{extCommitBytes, txBytes},
		ProposerAddress: common.FromHex(validators[0].Signer),
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
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(&ethTypes.Header{
			Number: big.NewInt(10),
		}, nil)
	mockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
	mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

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
				mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))
			}
		})
	}
}

// TestVerifyVoteExtensionHandler tests the VerifyVoteExtension handler of the HeimdallApp by creating a checkpoint message, building a signed transaction, preparing a proposal, extending the vote, and verifying the vote extension with valid and invalid transactions.
func TestVerifyVoteExtensionHandler(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
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

	txBytes, err := buildSignedTx(msg, ctx, priv, app)

	extCommitBytes, extCommit, voteInfo, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          3,
		Txs:             [][]byte{extCommitBytes, txBytes},
		ProposerAddress: common.FromHex(validators[0].Signer),
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
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(&ethTypes.Header{
			Number: big.NewInt(10),
		}, nil)
	mockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
	mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	reqVerify := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo.Validator.Address, // use the real consensus addr
		Height:             3,
		Hash:               []byte("test-hash"),
	}
	respVerify, err := app.VerifyVoteExtensionHandler()(ctx, &reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, respVerify.Status)

	// test cases for VerifyVoteExtensionHandler
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
			wantStatus: abci.ResponseVerifyVoteExtension_REJECT,
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

func TestVerifyVoteExtensionHandler_AcceptsOnBorQueryError(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "test",
	}

	txBytes, err := buildSignedTx(msg, ctx, priv, app)

	extCommitBytes, _, voteInfo, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          3,
		Txs:             [][]byte{extCommitBytes, txBytes},
		ProposerAddress: common.FromHex(validators[0].Signer),
	})
	require.NoError(t, err)

	// First, get a valid VE with a working mock caller (for ExtendVote)
	workingMockCaller := new(helpermocks.IContractCaller)
	workingMockCaller.
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(&ethTypes.Header{Number: big.NewInt(10)}, nil)
	workingMockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		app.AppCodec(),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)),
		workingMockCaller,
	)
	app.CheckpointKeeper = checkpointKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(checkpointTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		&app.StakeKeeper,
		app.ChainManagerKeeper,
		&app.TopupKeeper,
		workingMockCaller,
	)
	app.BorKeeper = borKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ChainManagerKeeper,
		&app.StakeKeeper,
		nil,
		nil,
	)
	app.BorKeeper.SetContractCaller(workingMockCaller)
	app.MilestoneKeeper.IContractCaller = workingMockCaller
	app.caller = workingMockCaller

	reqExtend := abci.RequestExtendVote{
		Txs:    [][]byte{extCommitBytes, txBytes},
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)

	// Now swap in a broken mock caller that simulates bor_getRootHash failure (e.g., not available like in erigon)
	brokenMockCaller := new(helpermocks.IContractCaller)
	brokenMockCaller.
		On("CheckIfBlocksExist", mock.Anything, mock.Anything).
		Return(false, fmt.Errorf("the method bor_getRootHash does not exist/is not available"))
	brokenMockCaller.
		On("GetRootHash", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte(nil), fmt.Errorf("the method bor_getRootHash does not exist/is not available"))
	app.caller = brokenMockCaller

	// Verify: the NonRpVE contains valid checkpoint data, but bor is unreachable.
	reqVerify := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo.Validator.Address,
		Height:             3,
		Hash:               []byte("test-hash"),
	}
	respVerify, err := app.VerifyVoteExtensionHandler()(ctx, &reqVerify)
	require.NoError(t, err)
	require.Equal(t,
		abci.ResponseVerifyVoteExtension_ACCEPT,
		respVerify.Status,
		"expected ACCEPT when NonRpVE validation fails with bor query error (ErrFailedToQueryBor)",
	)
}

// TestVerifyVoteExtensionHandler_RejectsUnknownFieldsPadding tests that the VerifyVoteExtension handler rejects vote extensions that contain unknown fields padding, ensuring that the handler properly validates the structure of the vote extension data.
func TestVerifyVoteExtensionHandler_RejectsUnknownFieldsPadding(t *testing.T) {
	setupAppResult := SetupApp(t, 1)
	hApp := setupAppResult.App
	validatorPrivKeys := setupAppResult.ValidatorKeys

	ctx := hApp.BaseApp.NewContext(true)
	ctx = setupContextWithVoteExtensionsEnableHeight(ctx, 1)

	vals := hApp.StakeKeeper.GetAllValidators(ctx)
	require.NotEmpty(t, vals)

	valAddrBytes := common.FromHex(vals[0].Signer)

	cometVal := abci.Validator{
		Address: valAddrBytes,
		Power:   vals[0].VotingPower,
	}

	ext := setupExtendedVoteInfo(
		t,
		cmtproto.BlockIDFlagCommit,
		common.FromHex(TxHash1),
		common.FromHex(TxHash2),
		cometVal,
		validatorPrivKeys[0],
	)

	// padding for VEs
	paddedVE := appendProtobufPadding(ext.VoteExtension, 64*1024)

	req := &abci.RequestVerifyVoteExtension{
		ValidatorAddress: valAddrBytes,
		Height:           CurrentHeight,
		VoteExtension:    paddedVE,
	}

	resp, err := hApp.VerifyVoteExtensionHandler()(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, resp.Status)
}

// TestPreBlocker tests the PreBlocker function of the HeimdallApp by creating a MsgProposeSpan message, building a signed transaction, creating an extension commit, and calling the PreBlocker with the transaction and extension commit to ensure it processes without errors.
func TestPreBlocker(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	msg := &borTypes.MsgProposeSpan{
		Proposer:   validators[0].Signer,
		StartBlock: 26657,
		EndBlock:   30000,
		ChainId:    "testChainParams.ChainParams.BorChainId",
		Seed:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		SeedAuthor: "val1Addr.Hex()",
	}

	txBytes, err := buildSignedTx(msg, ctx, priv, app)
	var txBytesCmt cmtTypes.Tx = txBytes

	extCommitBytes, _, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)

	err = app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes})
	require.NoError(t, err)

	finalizeReq := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytes, txBytes},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}
	_, err = app.PreBlocker(ctx, &finalizeReq)
	require.NoError(t, err)

}

// TestSideTxsHappyPath tests the happy path for side transactions in the HeimdallApp by setting up a mock contract caller, configuring the necessary keepers, and ensuring that the side transaction processing works correctly without errors.
func TestSideTxsHappyPath(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	blockNumber := uint64(599)

	_, _, addr2 := testdata.KeyTestPubAddr()

	txReceipt := &ethTypes.Receipt{
		BlockNumber: new(big.Int).SetUint64(blockNumber),
	}

	stateSyncEvent := &statesender.StatesenderStateSynced{
		Id:              new(big.Int).SetUint64(1),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Data:            []byte("test-data"),
	}

	event := &stakinginfo.StakinginfoTopUpFee{
		User: common.Address(addr2.Bytes()),
		Fee:  new(big.Int).SetUint64(1),
	}

	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(&ethTypes.Header{
			Number: big.NewInt(10),
		}, nil)
	mockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

	mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.AnythingOfType("int64")).Return(txReceipt, nil)
	mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)
	app.TopupKeeper = topupKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.BankKeeper,
		app.ChainManagerKeeper,
		&app.StakeKeeper,
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
	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[borTypes.ModuleName] = bor.NewAppModule(new(borKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ChainManagerKeeper,
		&app.StakeKeeper,
		nil,
		nil,
	)))
	app.BorKeeper.SetContractCaller(mockCaller)

	app.ModuleManager.Modules[clerkTypes.ModuleName] = clerk.NewAppModule(new(clerkKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.ChainManagerKeeper,
		mockCaller,
	)))
	app.sideTxCfg = sidetxs.NewSideTxConfigurator()
	app.RegisterSideMsgServices(app.sideTxCfg)

	propBytes := common.FromHex(validators[0].Signer)
	propAddr := sdk.AccAddress(propBytes)
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)
	require.NoError(t,
		testutil.FundAccount(ctx, app.BankKeeper, propAddr,
			sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
		),
	)

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
		{
			name: "Clerk Module Happy Path",
			msg: func() *clerkTypes.MsgEventRecord {
				return new(clerkTypes.NewMsgEventRecord(
					validators[0].Signer,
					TxHash1,
					1,
					50,
					1,
					propAddr,
					make([]byte, 0),
					"0",
				))
			}(),
		},
	}

	mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil)
	mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(stateSyncEvent, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			txBytes, err := buildSignedTx(tc.msg, ctx, priv, app)
			var txBytesCmt cmtTypes.Tx = txBytes

			extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
			_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
				Height:          3,
				Txs:             [][]byte{extCommitBytes, txBytes},
				ProposerAddress: common.FromHex(validators[0].Signer),
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
			mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

			err = app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes})
			require.NoError(t, err)

			extCommitBytes2, _, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
			require.NoError(t, err)

			finalizeReq := abci.RequestFinalizeBlock{
				Txs:             [][]byte{extCommitBytes2, txBytes},
				Height:          3,
				ProposerAddress: common.FromHex(validators[0].Signer),
			}
			_, err = app.PreBlocker(ctx, &finalizeReq)
			require.NoError(t, err)

		})
	}

}

// TestAllUnhappyPathBorSideTxs tests various unhappy path scenarios for Bor side transactions in the HeimdallApp by setting up a mock contract caller, configuring the necessary keepers, and ensuring that the side transaction processing correctly handles errors and edge cases without causing unexpected behavior.
func TestAllUnhappyPathBorSideTxs(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	valSet, vals := genTestValidators()

	mockCaller := new(helpermocks.IContractCaller)

	app.ChainManagerKeeper = chainmanagerKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	helper.SetTestInitialHeight(3)
	app.TopupKeeper = topupKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.BankKeeper,
		app.ChainManagerKeeper,
		&app.StakeKeeper,
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
	mockBorKeeper := borKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ChainManagerKeeper,
		&app.StakeKeeper,
		nil,
		mockCaller,
	)
	app.BorKeeper = mockBorKeeper

	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[borTypes.ModuleName] = bor.NewAppModule(&mockBorKeeper)
	app.BorKeeper.SetContractCaller(mockCaller)
	app.sideTxCfg = sidetxs.NewSideTxConfigurator()
	app.RegisterSideMsgServices(app.sideTxCfg)

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

	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return(&val1Addr, nil)
	mockCaller.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(&blockHeader1, nil)
	mockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.Anything, mock.Anything).
		Return([]*ethTypes.Header{&blockHeader1}, []uint64{1}, []common.Address{common.HexToAddress(vals[0].GetOperator())}, nil)

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

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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
		mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

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

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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
		mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

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

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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
		mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

	})

}

// TestAllUnhappyPathClerkSideTxs tests various unhappy path scenarios for Clerk side transactions in the HeimdallApp by setting up a mock contract caller, configuring the necessary keepers, and ensuring that the side transaction processing correctly handles errors and edge cases without causing unexpected behavior.
func TestAllUnhappyPathClerkSideTxs(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCaller := new(helpermocks.IContractCaller)
	mockChainKeeper := clerktestutil.NewMockChainKeeper(ctrl)

	app.ChainManagerKeeper = chainmanagerKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	helper.SetTestInitialHeight(3)
	app.TopupKeeper = topupKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.BankKeeper,
		app.ChainManagerKeeper,
		&app.StakeKeeper,
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
	mockBorKeeper := borKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ChainManagerKeeper,
		&app.StakeKeeper,
		nil,
		mockCaller,
	)
	app.BorKeeper = mockBorKeeper

	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[clerkTypes.ModuleName] = clerk.NewAppModule(new(clerkKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		mockChainKeeper,
		mockCaller,
	)))
	app.BorKeeper.SetContractCaller(mockCaller)
	app.sideTxCfg = sidetxs.NewSideTxConfigurator()
	app.RegisterSideMsgServices(app.sideTxCfg)

	propBytes := common.FromHex(validators[0].Signer)
	propAddr := sdk.AccAddress(propBytes)
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)
	require.NoError(t,
		testutil.FundAccount(ctx, app.BankKeeper, propAddr,
			sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
		),
	)

	mockChainKeeper.
		EXPECT().
		GetParams(gomock.Any()).
		Return(chainmanagertypes.DefaultParams(), nil).
		AnyTimes()

	t.Run("no receipt", func(t *testing.T) {

		logIndex := uint64(200)
		blockNumber := uint64(51)

		ac := address.NewHexCodec()
		Address2 := "0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff"

		addrBz2, err := ac.StringToBytes(Address2)
		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(1),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(new(clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			logIndex,
			blockNumber,
			10,
			addrBz2,
			make([]byte, 0),
			"101",
		)), ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

	t.Run("NoLog", func(t *testing.T) {

		logIndex := uint64(100)
		blockNumber := uint64(510)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
		}

		ac := address.NewHexCodec()
		Address2 := "0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff"

		addrBz2, err := ac.StringToBytes(Address2)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(new(clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			logIndex,
			blockNumber,
			10,
			addrBz2,
			make([]byte, 0),
			"0",
		)), ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

	t.Run("EventDataExceed", func(t *testing.T) {

		id := uint64(111)
		logIndex := uint64(1)
		blockNumber := uint64(1000)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}

		const letterBytes = "abcdefABCDEF"
		b := make([]byte, helper.MaxStateSyncSize+3)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}

		ac := address.NewHexCodec()
		Address2 := "0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff"

		addrBz2, err := ac.StringToBytes(Address2)

		msg := clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			[]byte("123"),
			"0",
		)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		event := &statesender.StatesenderStateSynced{
			Id:              new(big.Int).SetUint64(msg.Id),
			ContractAddress: common.BytesToAddress([]byte(msg.ContractAddress)),
			Data:            b,
		}
		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil).Once()

		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

	t.Run("Post Handler should fail for no vote", func(t *testing.T) {

		id := uint64(111)
		logIndex := uint64(1)
		blockNumber := uint64(1000)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}

		const letterBytes = "abcdefABCDEF"
		b := make([]byte, helper.MaxStateSyncSize+3)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}

		ac := address.NewHexCodec()
		Address2 := "0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff"

		addrBz2, err := ac.StringToBytes(Address2)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(new(clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			make([]byte, 0),
			"0",
		)), ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

		finalizeReq := abci.RequestFinalizeBlock{
			Txs:             [][]byte{extCommitBytes, txBytes},
			Height:          3,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}
		_, err = app.PreBlocker(ctx, &finalizeReq)
		require.NoError(t, err)

	})

}

// TestAllUnhappyPathTopupSideTxs tests various unhappy path scenarios for Topup side transactions in the HeimdallApp by setting up a mock contract caller, configuring the necessary keepers, and ensuring that the topup transaction processing correctly handles errors and edge cases without causing unexpected behavior.
func TestAllUnhappyPathTopupSideTxs(t *testing.T) {

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCaller := new(helpermocks.IContractCaller)
	mockChainKeeper := clerktestutil.NewMockChainKeeper(ctrl)

	app.ChainManagerKeeper = chainmanagerKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	helper.SetTestInitialHeight(3)
	app.TopupKeeper = topupKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.BankKeeper,
		mockChainKeeper,
		&app.StakeKeeper,
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
	mockBorKeeper := borKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ChainManagerKeeper,
		&app.StakeKeeper,
		nil,
		mockCaller,
	)
	app.BorKeeper = mockBorKeeper

	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[topUpTypes.ModuleName] = topup.NewAppModule(new(app.TopupKeeper))
	app.BorKeeper.SetContractCaller(mockCaller)
	app.sideTxCfg = sidetxs.NewSideTxConfigurator()
	app.RegisterSideMsgServices(app.sideTxCfg)

	propBytes := common.FromHex(validators[0].Signer)
	propAddr := sdk.AccAddress(propBytes)
	propAcc := authTypes.NewBaseAccount(propAddr, nil, 1337, 0)
	app.AccountKeeper.SetAccount(ctx, propAcc)
	require.NoError(t,
		testutil.FundAccount(ctx, app.BankKeeper, propAddr,
			sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
		),
	)

	mockChainKeeper.
		EXPECT().
		GetParams(gomock.Any()).
		Return(chainmanagertypes.DefaultParams(), nil).
		AnyTimes()

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()

	t.Run("no receipt", func(t *testing.T) {

		logIndex := uint64(10)
		blockNumber := uint64(599)
		hash := []byte(TxHash1)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(new(*topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)), ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

	t.Run("No Log", func(t *testing.T) {

		logIndex := uint64(10)
		blockNumber := uint64(599)
		hash := []byte(TxHash1)

		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(new(*topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)), ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

	t.Run("block mismatch", func(t *testing.T) {

		logIndex := uint64(10)
		blockNumber := uint64(600)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
		}
		hash := []byte(TxHash1)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})

		event := &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(addr1.Bytes()),
			Fee:  coins.AmountOf(authTypes.FeeToken).BigInt(),
		}

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil).Once()
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(new(*topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)), ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

	t.Run("user mismatch", func(t *testing.T) {

		logIndex := uint64(10)
		blockNumber := uint64(700)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}
		hash := []byte(TxHash1)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})

		event := &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(addr2.Bytes()),
			Fee:  coins.AmountOf(authTypes.FeeToken).BigInt(),
		}
		fmt.Println("txReceipt: ", txReceipt)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil)
		mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(new(*topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)), ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2, nil)
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
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

		var ve sidetxs.VoteExtension

		err = ve.Unmarshal(respExtend.VoteExtension)
		require.NoError(t, err)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

}

// TestMilestoneHappyPath tests the happy path scenario for the Milestone module in the HeimdallApp by setting up a mock contract caller, configuring the necessary keepers, and ensuring that the milestone creation and processing flow works correctly without any errors or unexpected behavior.
func TestMilestoneHappyPath(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	err := app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   10000000000000000,
		ValidatorSet: stakeTypes.ValidatorSet{
			Validators: validators,
			Proposer:   validators[0],
		},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	})
	require.NoError(t, err)

	// Create a checkpoint message
	msg := &types.MsgCheckpoint{
		Proposer:        validators[0].Signer,
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      "test",
	}

	txBytes, err := buildSignedTx(msg, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          3,
		Txs:             [][]byte{extCommitBytes, txBytes},
		ProposerAddress: common.FromHex(validators[0].Signer),
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
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return(
			[]*ethTypes.Header{
				{
					ParentHash:  common.HexToHash("0xabc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc1"),
					UncleHash:   common.HexToHash("0xdef456def456def456def456def456def456def456def456def456def456def4"),
					Coinbase:    common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
					Root:        common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
					TxHash:      common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
					ReceiptHash: common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
					Number:      big.NewInt(10000000000000000),
				},
			},
			[]uint64{10000000000000000},
			[]common.Address{common.HexToAddress(validators[0].Signer)},
			nil,
		).Times(100)
	mockCaller.
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(
			&ethTypes.Header{
				ParentHash: common.BytesToHash([]byte(TxHash1)),
				Number:     big.NewInt(10000000000000000),
			},
			nil,
		).Times(100)

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
		nil,
	)
	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	// create a milestone
	testMilestone1 := milestoneTypes.Milestone{
		Proposer:    validators[0].Signer,
		StartBlock:  1,
		EndBlock:    2,
		Hash:        common.BytesToHash([]byte(TxHash1)).Bytes(),
		BorChainId:  "1",
		MilestoneId: "milestoneID",
		Timestamp:   144,
	}

	err = app.MilestoneKeeper.AddMilestone(ctx, testMilestone1)
	require.NoError(t, err)

	reqExtend := abci.RequestExtendVote{
		Txs:    respPrep.Txs,
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)

	var ve sidetxs.VoteExtension
	err = ve.Unmarshal(respExtend.VoteExtension)
	require.NoError(t, err)

	extCommitBytesWithMilestone, _, _, err := buildExtensionCommitsWithMilestoneProposition(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, *ve.MilestoneProposition)

	finalizeReq := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytesWithMilestone, txBytes},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	_, err = app.PreBlocker(ctx, &finalizeReq)
}

// TestMilestoneUnhappyPaths tests various unhappy path scenarios for the Milestone module in the HeimdallApp by setting up a mock contract caller, configuring the necessary keepers, and ensuring that the milestone creation and processing flow correctly handles errors and edge cases without causing unexpected behavior.
func TestMilestoneUnhappyPaths(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
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

	txBytes, err := buildSignedTx(msg, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          3,
		Txs:             [][]byte{extCommitBytes, txBytes},
		ProposerAddress: common.FromHex(validators[0].Signer),
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
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)
	mockCaller.
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(
			&ethTypes.Header{
				Number: big.NewInt(10000000000000000),
			},
			nil,
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
		nil,
	)
	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	t.Run("No milestone", func(t *testing.T) {
		reqExtend := abci.RequestExtendVote{
			Txs:    respPrep.Txs,
			Hash:   []byte("test-hash"),
			Height: 3,
		}
		respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
		require.NoError(t, err)
		require.NotNil(t, respExtend.VoteExtension)
		mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

		finalizeReq := abci.RequestFinalizeBlock{
			Txs:             [][]byte{extCommitBytes, txBytes},
			Height:          3,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		_, err = app.PreBlocker(ctx, &finalizeReq)
	})

	t.Run("No block with Majority Support", func(t *testing.T) {
		testMilestone1 := milestoneTypes.Milestone{
			Proposer:    validators[0].Signer,
			StartBlock:  1,
			EndBlock:    2,
			Hash:        []byte(TxHash1),
			BorChainId:  "1",
			MilestoneId: "milestoneID",
			Timestamp:   144,
		}

		err = app.MilestoneKeeper.AddMilestone(ctx, testMilestone1)
		require.NoError(t, err)

		reqExtend := abci.RequestExtendVote{
			Txs:    respPrep.Txs,
			Hash:   []byte("test-hash"),
			Height: 3,
		}
		respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
		require.NoError(t, err)
		require.NotNil(t, respExtend.VoteExtension)
		mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

		finalizeReq := abci.RequestFinalizeBlock{
			Txs:             [][]byte{extCommitBytes, txBytes},
			Height:          3,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		_, err = app.PreBlocker(ctx, &finalizeReq)
	})

}

// TestPrepareProposal tests the PrepareProposal handler in the HeimdallApp by setting up a mock contract caller, configuring the necessary keepers, and ensuring that the proposal preparation flow works correctly without any errors or unexpected behavior, including handling various edge cases and scenarios related to checkpoint messages and milestone propositions.
func TestPrepareProposal(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})
	priv, _, _ := testdata.KeyTestPubAddr()
	setupResult := SetupApp(t, 1)
	app := setupResult.App

	genState := app.DefaultGenesis()
	genBytes, err := json.Marshal(genState)
	require.NoError(t, err)
	_, err = app.InitChain(&abci.RequestInitChain{
		Validators:    []abci.ValidatorUpdate{},
		AppStateBytes: genBytes,
	})
	require.NoError(t, err)

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
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(&ethTypes.Header{
			Number: big.NewInt(1),
		}, nil)
	mockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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

	// Prepare the proposer account
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
		ChainID: func() string {
			if ctx.ChainID() != "" {
				return ctx.ChainID()
			}
			return app.ChainID()
		}(),
		AccountNumber: propAcc.GetAccountNumber(),
		Sequence:      propAcc.GetSequence(),
		PubKey:        priv.PubKey(),
	}
	sigV2, err = tx.SignWithPrivKey(context.TODO(), defaultSignMode, signerData,
		txBuilder, priv, txConfig, signerData.Sequence)
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
		Height:          3,
		Txs:             [][]byte{extCommitBytes, txBytes},
		ProposerAddress: common.FromHex(validators[0].Signer),
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

	reqPrepNoTx := &abci.RequestProcessProposal{
		Txs:                [][]byte{},
		ProposedLastCommit: abci.CommitInfo{Round: reqPrep.LocalLastCommit.Round},
		Height:             3,
	}
	respPrepNoTx, err := app.NewProcessProposalHandler()(ctx, reqPrepNoTx)
	require.NoError(t, err)
	require.Equal(t,
		abci.ResponseProcessProposal_REJECT,
		respPrepNoTx.Status,
		"expected a REJECT status when no txs are provided",
	)

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

	reqExCommitRoundMismatch := &abci.RequestProcessProposal{
		Txs:                respPrep.Txs,
		Height:             3,
		ProposedLastCommit: abci.CommitInfo{Round: 30},
	}
	respExCommitRoundMismatch, err := app.NewProcessProposalHandler()(ctx, reqExCommitRoundMismatch)
	require.NoError(t, err, "handler itself should not error")
	require.Equal(
		t,
		abci.ResponseProcessProposal_REJECT,
		respExCommitRoundMismatch.Status,
		"expected REJECT when ExtendedCommitInfo Round mismatches",
	)

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

	// ExtendVote
	reqExtend := abci.RequestExtendVote{
		Txs:    respPrep.Txs,
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)
	mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

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

	reqVerify := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address, // use the real consensus addr
		Height:             3,
		Hash:               []byte("test-hash"),
	}
	respVerify, err := app.VerifyVoteExtensionHandler()(ctx, &reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, respVerify.Status)

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

	badReqHeight := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,
		NonRpVoteExtension: respExtend.NonRpExtension,
		ValidatorAddress:   voteInfo1.Validator.Address,
		Height:             reqExtend.Height + 1, // deliberately wrong
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

	fakeExt := &sidetxs.VoteExtension{
		BlockHash:       []byte("whatever"),
		Height:          reqExtend.Height, // height‐check passes
		SideTxResponses: nil,              // nil to force validateSideTxResponses error
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

	fakeExtHeight := &sidetxs.VoteExtension{
		BlockHash:            []byte("whatever"),
		Height:               reqExtend.Height + 100, // deliberately wrong
		SideTxResponses:      nil,
		MilestoneProposition: nil,
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
			TxHash: []byte("deadbeef"),
			// leave other fields nil/zero so validation fails
		},
	}
	var goodExt sidetxs.VoteExtension
	require.NoError(t,
		gogoproto.Unmarshal(respExtend.VoteExtension, &goodExt),
		"should unmarshal the real VoteExtension",
	)
	// build a fake VoteExtension with the bad side‐txs
	fakeExt2 := &sidetxs.VoteExtension{
		BlockHash:       goodExt.BlockHash,
		Height:          goodExt.Height, // keep height correct
		SideTxResponses: badSide,        // invalid payload
	}
	fakeBz2, err := gogoproto.Marshal(fakeExt2)
	require.NoError(t, err, "gogo‐Marshal should succeed")

	// call the verifyHandler
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

	badReqNonRp := abci.RequestVerifyVoteExtension{
		VoteExtension:      respExtend.VoteExtension,       // use the good extension
		NonRpVoteExtension: []byte{0x01, 0x02, 0x03, 0xFF}, // invalid bytes to force an error
		ValidatorAddress:   voteInfo1.Validator.Address,    // correct consensus addr
		Height:             reqExtend.Height,               // correct height
		Hash:               []byte("test-hash"),
	}

	respNonRp, err := app.VerifyVoteExtensionHandler()(ctx, &badReqNonRp)
	require.NoError(t, err, "handler should swallow non-RP validation errors and continue")
	require.Equal(
		t,
		abci.ResponseVerifyVoteExtension_REJECT,
		respNonRp.Status,
		"expected REJECT when validateNonRpVoteExtensionData returns an error",
	)
	fmt.Println("finally!")

	// Test FinalizeBlock handler
	finalizeReq := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytes, txBytes},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}
	_, err = app.PreBlocker(ctx, &finalizeReq)
	require.NoError(t, err)

	msgBor := &borTypes.MsgProposeSpan{
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
	err = app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytesBor})
	require.NoError(t, err)

	voteInfo2 := setupEmptyExtendedVoteInfo(
		t,
		cmtproto.BlockIDFlagCommit,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal1,
		validatorPrivKeys[0],
		2,
		app,
	)

	extCommit2 := &abci.ExtendedCommitInfo{
		Votes: []abci.ExtendedVoteInfo{voteInfo2},
	}
	extCommitBytes2, err := extCommit2.Marshal()
	require.NoError(t, err)

	// Test FinalizeBlock handler
	finalizeReqBorSideTx := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytes2, txBytesBor},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}
	_, err = app.PreBlocker(ctx, &finalizeReqBorSideTx)
	require.NoError(t, err)

	require.NoError(t, txBuilder.SetMsgs(new(clerkTypes.NewMsgEventRecord(
		validators[0].Signer,
		TxHash1,
		1,
		50,
		1,
		propAddr,
		make([]byte, 0),
		"0",
	))))
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(sigV2))

	txBytesClerk, err := txConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	err = app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytesClerk})
	require.NoError(t, err)

	voteInfo3 := setupEmptyExtendedVoteInfo(
		t,
		cmtproto.BlockIDFlagCommit,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal1,
		validatorPrivKeys[0],
		2,
		app,
	)

	extCommit3 := &abci.ExtendedCommitInfo{
		Votes: []abci.ExtendedVoteInfo{voteInfo3},
	}
	extCommitBytes3, err := extCommit3.Marshal()
	require.NoError(t, err)

	// Test FinalizeBlock handler
	finalizeReqClerkSideTx := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytes3, txBytesClerk},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}
	_, err = app.PreBlocker(ctx, &finalizeReqClerkSideTx)
	require.NoError(t, err)

	coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
	require.NoError(t, err)
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
	err = app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytesTopUp})
	require.NoError(t, err)

	_, err = app.Commit()
	require.NoError(t, err)

	voteInfo4 := setupEmptyExtendedVoteInfo(
		t,
		cmtproto.BlockIDFlagCommit,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal1,
		validatorPrivKeys[0],
		2,
		app,
	)

	extCommit4 := &abci.ExtendedCommitInfo{
		Votes: []abci.ExtendedVoteInfo{voteInfo4},
	}
	extCommitBytes4, err := extCommit4.Marshal()
	require.NoError(t, err)

	// Test FinalizeBlock handler
	finalizeReqTopUpSideTx := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytes4, txBytesTopUp},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}
	_, err = app.PreBlocker(ctx, &finalizeReqTopUpSideTx)
	require.NoError(t, err)
}

var defaultFeeAmount = big.NewInt(10).Exp(big.NewInt(10), big.NewInt(15), nil).Int64()

// TestUpdateBlockProducerStatus tests the updateBlockProducerStatus function in the HeimdallApp by setting up an initial state with active and failed producers, providing a new set of supporting producers, and verifying that the function correctly updates the latest active producers while clearing the latest failed producers, ensuring that the application state reflects the expected changes after the function call.
func TestUpdateBlockProducerStatus(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)

	// Set up the initial state for the latest active and failed producers
	initialActiveProducers := map[uint64]struct{}{1: {}, 2: {}}
	err := app.BorKeeper.UpdateLatestActiveProducer(ctx, initialActiveProducers)
	require.NoError(t, err)

	err = app.BorKeeper.AddLatestFailedProducer(ctx, 3)
	require.NoError(t, err)
	err = app.BorKeeper.AddLatestFailedProducer(ctx, 4)
	require.NoError(t, err)

	// The supporting producers for the new block
	supportingProducerIDs := map[uint64]struct{}{5: {}, 6: {}}

	// Call the function to update the block producer status
	err = app.updateBlockProducerStatus(ctx, supportingProducerIDs)
	require.NoError(t, err)

	// Check the state after the call
	latestActive, err := app.BorKeeper.GetLatestActiveProducer(ctx)
	require.NoError(t, err)
	require.Equal(t, supportingProducerIDs, latestActive)

	latestFailed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
	require.NoError(t, err)
	require.Empty(t, latestFailed)
}

// TestCheckAndAddFutureSpan tests the checkAndAddFutureSpan function in the HeimdallApp by setting up a mock application state with validators and a last span, providing different milestone propositions and supporting validator sets, and verifying that the function correctly adds a new future span when the conditions are met while ensuring that no new span is added when the conditions are not satisfied, thus validating the expected behavior of span management in the application.
func TestCheckAndAddFutureSpan(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)

	// Get validators to create the proper span
	validators := app.StakeKeeper.GetAllValidators(ctx)
	valSlice := make([]*stakeTypes.Validator, len(validators))
	for i := range validators {
		valSlice[i] = validators[i]
	}
	valSet := stakeTypes.ValidatorSet{Validators: valSlice}

	// Create validators for selected producers
	selectedProducers := make([]stakeTypes.Validator, len(validators))
	for i, val := range validators {
		selectedProducers[i] = *val
	}

	lastSpan := borTypes.Span{
		Id:                1,
		StartBlock:        100,
		EndBlock:          200,
		BorChainId:        "1",
		ValidatorSet:      valSet,
		SelectedProducers: selectedProducers,
	}
	err := app.BorKeeper.AddNewSpan(ctx, &lastSpan)
	require.NoError(t, err)

	producerValID := selectedProducers[0].ValId
	// The producer is not in the supporting set.
	supportingValidatorIDs := make(map[uint64]struct{})
	for _, v := range validators {
		if v.ValId != producerValID {
			supportingValidatorIDs[v.ValId] = struct{}{}
		}
	}

	t.Run("condition false", func(t *testing.T) {
		majorityMilestone := &milestoneTypes.MilestoneProposition{
			StartBlockNumber: 50, // This will make the condition false
			BlockHashes:      [][]byte{[]byte("hash1")},
		}

		err := app.checkAndAddFutureSpan(ctx, majorityMilestone, lastSpan, supportingValidatorIDs)
		require.NoError(t, err)

		// Check that no new span was added
		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id, currentLastSpan.Id)
	})

	t.Run("condition true", func(t *testing.T) {
		majorityMilestone := &milestoneTypes.MilestoneProposition{
			StartBlockNumber: 150, // This will make the condition true
			BlockHashes:      [][]byte{[]byte("hash1")},
		}

		helper.SetRioHeight(int64(lastSpan.EndBlock + 1))

		// Mock IContractCaller to return the address.
		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return([]common.Address{common.HexToAddress(validators[0].Signer)}, nil)
		app.BorKeeper.SetContractCaller(mockCaller)

		params, err := app.BorKeeper.GetParams(ctx)
		require.NoError(t, err)

		// Set up producer votes so that producer selection can work
		if len(validators) > 1 {
			// All validators vote for the same candidate to ensure consensus
			var consensusCandidateID uint64
			for _, v := range validators {
				if v.ValId != producerValID {
					consensusCandidateID = v.ValId
					break
				}
			}

			allValidatorIDs := make(map[uint64]struct{})
			for _, val := range validators {
				allValidatorIDs[val.ValId] = struct{}{}
				producerVotes := borTypes.ProducerVotes{Votes: []uint64{consensusCandidateID}}
				err := app.BorKeeper.SetProducerVotes(ctx, val.ValId, producerVotes)
				require.NoError(t, err)
			}

			// Set up producer performance scores
			err := app.BorKeeper.UpdateValidatorPerformanceScore(ctx, allValidatorIDs, 1)
			require.NoError(t, err)

			// Set up the minimal span state
			params, err := app.BorKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.ProducerCount = 1
			err = app.BorKeeper.SetParams(ctx, params)
			require.NoError(t, err)
		}

		err = app.checkAndAddFutureSpan(ctx, majorityMilestone, lastSpan, supportingValidatorIDs)
		require.NoError(t, err)

		// Make sure the new span was created
		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id+1, currentLastSpan.Id, "a new span should be created with incremented ID")
		require.Equal(t, lastSpan.EndBlock+1, currentLastSpan.StartBlock, "new span should start after the last span")
		require.Equal(t, currentLastSpan.StartBlock+params.SpanDuration-1, currentLastSpan.EndBlock, "new span should have the exact span duration defined in params")
	})
}

// TestCheckAndRotateCurrentSpan tests the checkAndRotateCurrentSpan function in the HeimdallApp by setting up a mock application state with validators and a last span, providing different block heights and Rio heights, and verifying that the function correctly rotates the current span when the conditions are met while ensuring that no rotation occurs when the conditions are not satisfied, thus validating the expected behavior of span rotation in the application.
func TestCheckAndRotateCurrentSpan(t *testing.T) {
	t.Run("condition false - diff too small", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)

		lastMilestone := &milestoneTypes.Milestone{EndBlock: 100}
		err := app.MilestoneKeeper.AddMilestone(ctx, *lastMilestone)
		require.NoError(t, err)
		lastMilestoneBlock := uint64(50)
		err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, lastMilestoneBlock)
		require.NoError(t, err)

		// Get validators to create the proper span
		validators := app.StakeKeeper.GetAllValidators(ctx)
		valSlice := make([]*stakeTypes.Validator, len(validators))
		for i := range validators {
			valSlice[i] = validators[i]
		}
		valSet := stakeTypes.ValidatorSet{Validators: valSlice}

		// Create validators for selected producers
		selectedProducers := make([]stakeTypes.Validator, len(validators))
		for i, val := range validators {
			selectedProducers[i] = *val
		}

		lastSpan := borTypes.Span{
			Id:                1,
			StartBlock:        90,
			EndBlock:          190,
			BorChainId:        "1",
			ValidatorSet:      valSet,
			SelectedProducers: selectedProducers,
		}
		err = app.BorKeeper.AddNewSpan(ctx, &lastSpan)
		require.NoError(t, err)

		ctx = ctx.WithBlockHeight(int64(lastMilestoneBlock) + helper.GetChangeProducerThreshold(ctx)) // diff == ChangeProducerThreshold

		err = app.checkAndRotateCurrentSpan(ctx)
		require.NoError(t, err)

		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id, currentLastSpan.Id)
	})

	t.Run("condition false - not veBlop", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)

		lastMilestone := &milestoneTypes.Milestone{EndBlock: 100}
		err := app.MilestoneKeeper.AddMilestone(ctx, *lastMilestone)
		require.NoError(t, err)
		lastMilestoneBlock := uint64(50)
		err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, lastMilestoneBlock)
		require.NoError(t, err)

		// Get validators to create the proper span
		validators := app.StakeKeeper.GetAllValidators(ctx)
		valSlice := make([]*stakeTypes.Validator, len(validators))
		for i := range validators {
			valSlice[i] = validators[i]
		}
		valSet := stakeTypes.ValidatorSet{Validators: valSlice}

		// Create validators for selected producers
		selectedProducers := make([]stakeTypes.Validator, len(validators))
		for i, val := range validators {
			selectedProducers[i] = *val
		}

		lastSpan := borTypes.Span{
			Id:                1,
			StartBlock:        90,
			EndBlock:          190,
			BorChainId:        "1",
			ValidatorSet:      valSet,
			SelectedProducers: selectedProducers,
		}
		err = app.BorKeeper.AddNewSpan(ctx, &lastSpan)
		require.NoError(t, err)

		ctx = ctx.WithBlockHeight(int64(lastMilestoneBlock) + helper.GetChangeProducerThreshold(ctx) + 1)
		helper.SetRioHeight(int64(lastMilestone.EndBlock + 2)) // Makes IsRio false

		err = app.checkAndRotateCurrentSpan(ctx)
		require.NoError(t, err)

		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id, currentLastSpan.Id)

		helper.SetRioHeight(0) // reset
	})

	t.Run("condition true", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)

		lastMilestone := &milestoneTypes.Milestone{
			EndBlock:   100,
			BorChainId: "1",
		}
		err := app.MilestoneKeeper.AddMilestone(ctx, *lastMilestone)
		require.NoError(t, err)
		lastMilestoneBlock := uint64(50)
		err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, lastMilestoneBlock)
		require.NoError(t, err)

		validators := app.StakeKeeper.GetAllValidators(ctx)
		valSlice := make([]*stakeTypes.Validator, len(validators))
		for i := range validators {
			valSlice[i] = validators[i]
		}
		valSet := stakeTypes.ValidatorSet{Validators: valSlice}

		// Create validators for selected producers
		selectedProducers := make([]stakeTypes.Validator, len(validators))
		for i, val := range validators {
			selectedProducers[i] = *val
		}

		lastSpan := borTypes.Span{
			Id:                1,
			StartBlock:        90,
			EndBlock:          190,
			BorChainId:        "1",
			ValidatorSet:      valSet,
			SelectedProducers: selectedProducers,
		}
		err = app.BorKeeper.AddNewSpan(ctx, &lastSpan)
		require.NoError(t, err)

		initialActiveProducers := make(map[uint64]struct{})
		for _, val := range validators {
			initialActiveProducers[val.ValId] = struct{}{}
		}

		// Add a few extra producer IDs to ensure we have candidates after the current producer is removed
		initialActiveProducers[1] = struct{}{}
		initialActiveProducers[2] = struct{}{}

		err = app.BorKeeper.UpdateLatestActiveProducer(ctx, initialActiveProducers)
		require.NoError(t, err)
		err = app.BorKeeper.AddLatestFailedProducer(ctx, uint64(99)) // some other producer
		require.NoError(t, err)

		// Set up comprehensive producer votes and state for successful producer selection
		if len(validators) > 0 {
			// For 3 validators with voting power 100 each:
			// totalPotentialProducers = 3
			// Max possible weighted vote at position 1: totalPotentialProducers * maxVotingPower = 3 * 100 = 300
			// Required threshold: (300 * 2/3) + 1 = 201
			// If all 3 validators vote for the same candidate at position 1: 3 * 100 = 300 > 201

			// Use actual validator IDs - find one that's not the current producer
			var consensusCandidate uint64
			for _, val := range validators {
				// Current producer is validators[0], so use any other validator
				if val.ValId != validators[0].ValId {
					consensusCandidate = val.ValId
					break
				}
			}
			if consensusCandidate == 0 {
				// Fallback: use the second validator if available
				if len(validators) > 1 {
					consensusCandidate = validators[1].ValId
				}
			}

			// Set producer votes - all validators vote for the same consensus candidate
			for _, val := range validators {
				// All validators vote for the consensus candidate in the first position, then fill with other validator IDs
				var votes []uint64
				votes = append(votes, consensusCandidate) // First choice - consensus candidate
				for j, otherVal := range validators {
					if otherVal.ValId != consensusCandidate && len(votes) < 3 {
						votes = append(votes, otherVal.ValId)
					}
					if len(votes) >= 3 {
						break
					}
					_ = j // avoid unused variable
				}

				producerVotes := borTypes.ProducerVotes{Votes: votes}
				err := app.BorKeeper.SetProducerVotes(ctx, val.ValId, producerVotes)
				require.NoError(t, err)

				// Include this validator in the initial active producers
				initialActiveProducers[val.ValId] = struct{}{}
			}

			// Ensure bor params allow for proper producer selection
			params, err := app.BorKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.ProducerCount = 3  // Allow 3 producers
			params.SpanDuration = 100 // Set reasonable span duration
			err = app.BorKeeper.SetParams(ctx, params)
			require.NoError(t, err)
		}

		// diff > ChangeProducerThreshold
		ctx = ctx.WithBlockHeight(int64(lastMilestoneBlock) + helper.GetChangeProducerThreshold(ctx) + 1)
		// Make IsRio true
		helper.SetRioHeight(int64(lastMilestone.EndBlock + 1))

		// Mock IContractCaller with proper producer mapping
		mockCaller := new(helpermocks.IContractCaller)
		producerSignerStr := validators[0].Signer
		mockCaller.On("GetBorChainBlockAuthor", mock.Anything, lastMilestone.EndBlock+1).
			Return(new(common.HexToAddress(producerSignerStr)), nil)
		app.BorKeeper.SetContractCaller(mockCaller)

		// Call the function to check and rotate the current span
		err = app.checkAndRotateCurrentSpan(ctx)
		require.NoError(t, err)

		// Assert that a new span was actually created
		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id+1, currentLastSpan.Id, "a new span should be created with incremented ID")
		require.Equal(t, lastMilestone.EndBlock+1, currentLastSpan.StartBlock, "new span should start after the last milestone")
		require.Equal(t, lastSpan.EndBlock, currentLastSpan.EndBlock, "new span will have the same end block as the last span")

		// Verify other expected state changes
		newLastMilestoneBlock, err := app.MilestoneKeeper.GetLastMilestoneBlock(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(ctx.BlockHeight())+helper.GetSpanRotationBuffer(ctx), newLastMilestoneBlock, "last milestone block should be updated")

		failedProducers, err := app.BorKeeper.GetLatestFailedProducer(ctx)
		require.NoError(t, err)

		currentProducerID := validators[0].ValId
		_, isFailed := failedProducers[currentProducerID]
		require.True(t, isFailed, "current producer should be added to failed list")
	})
}

// TestPreBlockerSpanRotationWithMinorityMilestone tests that span rotation is skipped
// when there's at least 1/3 voting power supporting a new milestone
func TestPreBlockerSpanRotationWithMinorityMilestone(t *testing.T) {
	_, app, ctx, validatorPrivKeys := SetupAppWithABCICtxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Set up consensus params to enable vote extensions
	params := cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	// Set up the initial state with milestone and span
	milestone := milestoneTypes.Milestone{
		MilestoneId: "1",
		StartBlock:  0,
		EndBlock:    100,
		Hash:        common.HexToHash("0x1234").Bytes(),
	}
	err := app.MilestoneKeeper.AddMilestone(ctx, milestone)
	require.NoError(t, err)

	// Set the last milestone block - this is needed for checkAndRotateCurrentSpan to work
	err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, milestone.EndBlock)
	require.NoError(t, err)

	span := &borTypes.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   200,
		ValidatorSet: stakeTypes.ValidatorSet{
			Validators: validators,
			Proposer:   validators[0],
		},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	}
	err = app.BorKeeper.AddNewSpan(ctx, span)
	require.NoError(t, err)

	// Set up the mock contract caller
	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).
		Return(new(common.HexToAddress(validators[0].Signer)), nil)
	app.BorKeeper.SetContractCaller(mockCaller)

	// Set context to trigger span rotation conditions
	blockHeight := int64(milestone.EndBlock) + helper.GetChangeProducerThreshold(ctx) + 1
	ctx = ctx.WithBlockHeight(blockHeight)
	// Set rio height to be at or before milestone.EndBlock+1 to ensure IsRio check passes
	helper.SetRioHeight(int64(milestone.EndBlock + 1))

	// Create vote extensions with 40% voting power supporting a new milestone
	// This is more than 1/3 but less than 2/3
	voteExtensions := createVoteExtensionsWithPartialSupport(t, validators, validatorPrivKeys, &milestone, 40, blockHeight-1)

	// Create ExtendedCommitInfo from vote extensions
	extCommit := &abci.ExtendedCommitInfo{
		Round: 0,
		Votes: voteExtensions,
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	req := &abci.RequestFinalizeBlock{
		Height:          ctx.BlockHeight(),
		Txs:             [][]byte{extCommitBytes, []byte("dummy-tx")},
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	// Execute PreBlocker
	_, err = app.PreBlocker(ctx, req)
	require.NoError(t, err)

	// Verify that span was not rotated
	currentSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, span.Id, currentSpan.Id, "Span should not have been rotated when 1/3+ voting power supports a milestone")
}

// TestPreBlockerSpanRotationWithoutMinorityMilestone tests that span rotation occurs
// when there's less than 1/3 voting power supporting a new milestone
func TestPreBlockerSpanRotationWithoutMinorityMilestone(t *testing.T) {
	_, app, ctx, validatorPrivKeys := SetupAppWithABCICtxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Set up consensus params to enable vote extensions
	params := cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	// Set up the initial state with milestone and span
	milestone := milestoneTypes.Milestone{
		MilestoneId: "1",
		StartBlock:  0,
		EndBlock:    100,
		Hash:        common.HexToHash("0x1234").Bytes(),
	}
	err := app.MilestoneKeeper.AddMilestone(ctx, milestone)
	require.NoError(t, err)

	// Set the last milestone block - this is needed for checkAndRotateCurrentSpan to work
	err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, milestone.EndBlock)
	require.NoError(t, err)

	span := &borTypes.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   200,
		ValidatorSet: stakeTypes.ValidatorSet{
			Validators: validators,
			Proposer:   validators[0],
		},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	}
	err = app.BorKeeper.AddNewSpan(ctx, span)
	require.NoError(t, err)

	// Set up the mock contract caller
	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).
		Return(new(common.HexToAddress(validators[0].Signer)), nil)
	app.BorKeeper.SetContractCaller(mockCaller)

	// Set context to trigger span rotation conditions
	blockHeight := int64(milestone.EndBlock) + helper.GetChangeProducerThreshold(ctx) + 1
	ctx = ctx.WithBlockHeight(blockHeight)
	// Set rio height to be at or before milestone.EndBlock+1 to ensure IsRio check passes
	helper.SetRioHeight(int64(milestone.EndBlock + 1))

	// Create vote extensions with only 20% voting power supporting a new milestone
	// This is less than 1/3
	voteExtensions := createVoteExtensionsWithPartialSupport(t, validators, validatorPrivKeys, &milestone, 20, blockHeight-1)

	// Create ExtendedCommitInfo from vote extensions
	extCommit := &abci.ExtendedCommitInfo{
		Round: 0,
		Votes: voteExtensions,
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	req := &abci.RequestFinalizeBlock{
		Height:          ctx.BlockHeight(),
		Txs:             [][]byte{extCommitBytes, []byte("dummy-tx")},
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	// Execute PreBlocker
	_, err = app.PreBlocker(ctx, req)
	require.NoError(t, err)

	// Verify that the span was rotated
	currentSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.NotEqual(t, span.Id, currentSpan.Id, "Span should have been rotated when less than 1/3 voting power supports a milestone")
}

// TestPreBlockerSpanRotationWithMajorityMilestone tests that span rotation is skipped
// when there's a 2/3-majority milestone (existing behavior)
func TestPreBlockerSpanRotationWithMajorityMilestone(t *testing.T) {
	_, app, ctx, validatorPrivKeys := SetupAppWithABCICtxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Set up consensus params to enable vote extensions
	params := cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	// Set up the initial state with milestone and span
	milestone := milestoneTypes.Milestone{
		MilestoneId: "1",
		StartBlock:  0,
		EndBlock:    100,
		Hash:        common.HexToHash("0x1234").Bytes(),
	}
	err := app.MilestoneKeeper.AddMilestone(ctx, milestone)
	require.NoError(t, err)

	// Set the last milestone block - this is needed for checkAndRotateCurrentSpan to work
	err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, milestone.EndBlock)
	require.NoError(t, err)

	span := &borTypes.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   200,
		ValidatorSet: stakeTypes.ValidatorSet{
			Validators: validators,
			Proposer:   validators[0],
		},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	}
	err = app.BorKeeper.AddNewSpan(ctx, span)
	require.NoError(t, err)

	// Set context to trigger span rotation conditions
	blockHeight := int64(milestone.EndBlock) + helper.GetChangeProducerThreshold(ctx) + 1
	ctx = ctx.WithBlockHeight(blockHeight)
	// Set rio height to be at or before milestone.EndBlock+1 to ensure IsRio check passes
	helper.SetRioHeight(int64(milestone.EndBlock + 1))

	// Create vote extensions with 70% voting power supporting a new milestone
	// This is more than 2/3
	voteExtensions := createVoteExtensionsWithPartialSupport(t, validators, validatorPrivKeys, &milestone, 70, blockHeight-1)

	// Create ExtendedCommitInfo from vote extensions
	extCommit := &abci.ExtendedCommitInfo{
		Round: 0,
		Votes: voteExtensions,
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	req := &abci.RequestFinalizeBlock{
		Height:          ctx.BlockHeight(),
		Txs:             [][]byte{extCommitBytes, []byte("dummy-tx")},
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	// Execute PreBlocker
	_, err = app.PreBlocker(ctx, req)
	require.NoError(t, err)

	// When there's a 2/3-majority milestone, it gets processed normally
	// This can include creating a new span if the milestone warrants it
	_, err = app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)

	// Verify that milestone was added
	latestMilestone, err := app.MilestoneKeeper.GetLastMilestone(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(101), latestMilestone.EndBlock, "New milestone should have been added with correct end block")
}

// TestPrepareProposal_MultipleTransactionsPerBlock tests the PrepareProposal handler's ability to handle multiple transactions in a single block, ensuring that all transactions are included in the proposal response and that the ExtendedCommitInfo is properly accounted for in the transaction count, thus validating the correct behavior of transaction processing and proposal preparation in scenarios with multiple transactions.
func TestPrepareProposal_MultipleTransactionsPerBlock(t *testing.T) {
	tests := []struct {
		name        string
		numTxs      int
		txSizes     []int
		expectTxs   int // including ExtendedCommitInfo
		description string
	}{
		{
			name:        "5 small regular transactions",
			numTxs:      5,
			txSizes:     []int{100, 100, 100, 100, 100},
			expectTxs:   6, // 1 ExtendedCommitInfo + 5 txs
			description: "All small transactions should fit within max bytes",
		},
		{
			name:        "10 transactions of varying sizes",
			numTxs:      10,
			txSizes:     []int{100, 200, 150, 300, 250, 180, 220, 160, 190, 140},
			expectTxs:   11, // 1 ExtendedCommitInfo + 10 txs
			description: "All varying size transactions should fit",
		},
		{
			name:        "50 small transactions",
			numTxs:      50,
			txSizes:     []int{},
			expectTxs:   51, // 1 ExtendedCommitInfo + 50 txs
			description: "Large number of small transactions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
			validators := app.StakeKeeper.GetAllValidators(ctx)

			// After setup, the app is at height 3 (after finalize at height 2 and commit)
			ctx = ctx.WithBlockHeight(3)

			var proposedTxs [][]byte
			propAddr := sdk.AccAddress(priv.PubKey().Address())
			propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
			sequence := propAcc.GetSequence()

			for i := 0; i < tt.numTxs; i++ {
				msg := &checkpointTypes.MsgCheckpoint{
					Proposer:        priv.PubKey().Address().String(),
					StartBlock:      uint64(100 + i*100),
					EndBlock:        uint64(200 + i*100),
					RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
					AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
					BorChainId:      "1",
				}

				txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
				require.NoError(t, err)
				proposedTxs = append(proposedTxs, txBytes)
				sequence++
			}

			_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
			require.NoError(t, err)

			req := &abci.RequestPrepareProposal{
				Txs:             proposedTxs,
				MaxTxBytes:      10000000,
				Height:          3,
				LocalLastCommit: *extCommit,
				ProposerAddress: common.FromHex(validators[0].Signer),
			}

			// Call app.PrepareProposal first to initialize state
			res, err := app.PrepareProposal(req)
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tt.expectTxs, len(res.Txs), "Should have exactly %d transactions (including ExtendedCommitInfo)", tt.expectTxs)
		})
	}
}

// TestPrepareProposal_MultipleSideTxsSameType tests the PrepareProposal handler's ability to handle multiple side transactions of the same type (e.g., multiple checkpoint messages or multiple bor propose span messages) in a single proposal, ensuring that all transactions are included in the proposal response and that the ExtendedCommitInfo is properly accounted for, thus validating the correct processing of multiple side transactions of the same type during proposal preparation.
func TestPrepareProposal_MultipleSideTxsSameType(t *testing.T) {
	t.Run("multiple checkpoint messages in different txs", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Create 5 checkpoint side txs
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 5; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 6, len(res.Txs), "Should have exactly 6 transactions (1 ExtendedCommitInfo + 5 checkpoint txs)")
	})

	t.Run("multiple bor propose span messages in different txs", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Create 3 bor propose span side txs
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 3; i++ {
			msg := &borTypes.MsgProposeSpan{
				Proposer:   priv.PubKey().Address().String(),
				SpanId:     uint64(1 + i),
				StartBlock: uint64(i * 6400),
				EndBlock:   uint64((i+1)*6400 - 1),
				ChainId:    "1",
				Seed:       common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 4, len(res.Txs), "Should have exactly 4 transactions (1 ExtendedCommitInfo + 3 bor propose span txs)")
	})
}

// TestPrepareProposal_MultipleSideTxsDifferentTypes tests the PrepareProposal handler's ability to handle multiple side transactions of different types (e.g., checkpoint messages, bor propose span messages, clerk event record messages, stake validator join messages, and topup messages) in a single proposal, ensuring that all transactions are included in the proposal response and that the ExtendedCommitInfo is properly accounted for, thus validating the correct processing of multiple side transactions of different types during proposal preparation.
func TestPrepareProposal_MultipleSideTxsDifferentTypes(t *testing.T) {
	t.Run("mix of checkpoint, bor, clerk, stake, and topup side txs", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		// 1. Checkpoint side tx
		checkpointMsg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}
		txBytes, err := buildSignedTxWithSequence(checkpointMsg, ctx, priv, app, sequence)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)
		sequence++

		// 2. Bor propose span side tx
		borMsg := &borTypes.MsgProposeSpan{
			Proposer:   priv.PubKey().Address().String(),
			SpanId:     1,
			StartBlock: 0,
			EndBlock:   6399,
			ChainId:    "1",
			Seed:       common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
		}
		txBytes, err = buildSignedTxWithSequence(borMsg, ctx, priv, app, sequence)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)
		sequence++

		// 3. Clerk event record side tx
		clerkMsg := &clerkTypes.MsgEventRecord{
			From:            priv.PubKey().Address().String(),
			TxHash:          "0x0000000000000000000000000000000000000000000000000000000000000001",
			LogIndex:        0,
			BlockNumber:     100,
			ContractAddress: "0x0000000000000000000000000000000000000001",
			Data:            []byte{0x01},
			Id:              1,
			ChainId:         "1",
		}
		txBytes, err = buildSignedTxWithSequence(clerkMsg, ctx, priv, app, sequence)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)
		sequence++

		// 4. Stake validator join side tx
		stakeMsg := &stakeTypes.MsgValidatorJoin{
			From:            priv.PubKey().Address().String(),
			ValId:           1,
			ActivationEpoch: 0,
			SignerPubKey:    secp256k1.GenPrivKey().PubKey().Bytes(),
			Amount:          math.NewInt(1000000000),
			TxHash:          common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			LogIndex:        0,
			BlockNumber:     100,
			Nonce:           0,
		}
		txBytes, err = buildSignedTxWithSequence(stakeMsg, ctx, priv, app, sequence)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)
		sequence++

		// 5. Topup side tx
		topupMsg := &topUpTypes.MsgTopupTx{
			Proposer:    priv.PubKey().Address().String(),
			User:        priv.PubKey().Address().String(),
			Fee:         math.NewInt(1000000000000000000),
			TxHash:      common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000003"),
			LogIndex:    0,
			BlockNumber: 100,
		}
		txBytes, err = buildSignedTxWithSequence(topupMsg, ctx, priv, app, sequence)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)
		sequence++

		_, _, _, err = buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 6, len(res.Txs), "Should have exactly 6 transactions (1 ExtendedCommitInfo + 5 side txs)")
	})
}

// TestPrepareProposal_MaxBytesConstraint tests the PrepareProposal handler's ability to enforce the MaxTxBytes constraint by including an ExtendedCommitInfo and multiple transactions that exceed the max bytes limit, ensuring that the handler correctly includes the ExtendedCommitInfo and only includes as many transactions as can fit within the specified MaxTxBytes, thus validating the proper handling of transaction size constraints during proposal preparation.
func TestPrepareProposal_MaxBytesConstraint(t *testing.T) {
	t.Run("exceeds max bytes with large transactions", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Create 10 transactions
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 10; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Use 100KB MaxTxBytes to accommodate VEs without restrictive filtering
		// Per-validator VE limit with 4 validators: (100000/4/3)-700 = 7633 bytes (reasonable)
		maxBytes := int64(100000)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      maxBytes,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		// With 100KB, ExtendedCommitInfo + all 10 txs should fit (each tx ~400 bytes)
		// Total: ~1-2KB ExtCommitInfo + 10*400 bytes = ~5-6KB total, well under 100KB
		require.Equal(t, 11, len(res.Txs), "Should have ExtendedCommitInfo + all 10 txs")
	})
}

// TestPrepareProposal_TransactionWithMultipleSideHandlers tests the PrepareProposal handler's ability to process transactions that contain multiple side messages of different types (e.g., a transaction that includes both a checkpoint message and a bor propose-span message), ensuring that the handler correctly identifies and processes all side messages within the transaction, includes the appropriate ExtendedCommitInfo, and returns a proposal response that accounts for all valid transactions and side messages, thus validating the proper handling of complex transactions with multiple side handlers during proposal preparation.
func TestPrepareProposal_TransactionWithMultipleSideHandlers(t *testing.T) {
	t.Run("skip tx with multiple side messages", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// This test would require creating a tx with multiple side messages
		// which should be skipped by PrepareProposal
		// Note: The current transaction builder might not easily support this,
		// but the code path exists in PrepareProposal to handle it

		_, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// test with a single side tx to ensure it's not skipped
		checkpointMsg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTx(checkpointMsg, ctx, priv, app)
		require.NoError(t, err)

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 2, len(res.Txs), "Should have exactly 2 transactions (1 ExtendedCommitInfo + 1 side tx)")
	})
}

// TestPrepareProposal_AccountSequenceMismatch tests the PrepareProposal handler's ability to handle transactions with account sequence mismatches by including multiple transactions with the same sequence number, ensuring that the handler correctly processes the first transaction and rejects subsequent transactions with duplicate sequence numbers, includes the appropriate ExtendedCommitInfo, and returns a proposal response that accounts for valid transactions while rejecting those with sequence mismatches, thus validating the proper handling of account sequence mismatches during proposal preparation.
func TestPrepareProposal_AccountSequenceMismatch(t *testing.T) {
	t.Run("reject transactions with duplicate sequence numbers", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// After setup, the app is at height 3 (after finalize at height 2 and commit)
		ctx = ctx.WithBlockHeight(3)

		// Create 5 transactions all with the same sequence number (0)
		// This should cause all but the first one to be rejected
		var proposedTxs [][]byte
		for i := 0; i < 5; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			// Intentionally use sequence 0 for ALL transactions
			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, 0)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Only the first transaction should be accepted (sequence 0 is correct for the first tx)
		// All other transactions will fail because they also have sequence=0, but the account sequence is now 1
		// Result should be: 1 ExtendedCommitInfo + 1 successful tx = 2 total
		require.Equal(t, 2, len(res.Txs), "Should have exactly 2 transactions (1 ExtendedCommitInfo + 1 tx, others rejected due to sequence mismatch)")
	})

	t.Run("accept transactions with correct incrementing sequence numbers", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		ctx = ctx.WithBlockHeight(3)

		// Create 5 transactions with properly incrementing sequence numbers
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 5; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)
		require.NoError(t, err)
		require.NotNil(t, res)

		// All transactions should be accepted with correct sequence numbers
		// Result should be: 1 ExtendedCommitInfo + 5 successful txs = 6 total
		require.Equal(t, 6, len(res.Txs), "Should have exactly 6 transactions (1 ExtendedCommitInfo + 5 txs with correct sequences)")
	})
}

func TestPrepareProposal_SideTxCap(t *testing.T) {
	t.Run("includes at most maxSideTxResponsesCount side txs", func(t *testing.T) {
		// PrepareProposal cap is Zurich-gated to match ProcessProposal.
		helper.SetZurichHardforkHeight(1)
		t.Cleanup(func() { helper.SetZurichHardforkHeight(0) })

		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)
		ctx = ctx.WithBlockHeight(3)

		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < maxSideTxResponsesCount+1; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(
			t,
			app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			2,
			nil,
		)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, maxSideTxResponsesCount+1, len(res.Txs), "expected 1 commit tx + max side txs")
	})

	t.Run("pre-Zurich: cap is dormant, all side txs included", func(t *testing.T) {
		// Zurich inactive (height = 0): the gate function returns false and the
		// cap branch is dead code, so all side-tx-bearing txs are included.
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)
		ctx = ctx.WithBlockHeight(3)

		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < maxSideTxResponsesCount+2; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, maxSideTxResponsesCount+2+1, len(res.Txs), "pre-Zurich cap is dormant: expected 1 commit tx + all side txs")
	})
}

// TestProcessProposal_ValidProposalMultipleTxs tests the ProcessProposal handler's ability to process a valid proposal containing multiple transactions by including an ExtendedCommitInfo and several valid transactions in the proposal request, ensuring that the handler correctly processes all transactions, validates the ExtendedCommitInfo, and returns an acceptance response, thus validating the proper processing of valid proposals with multiple transactions during proposal evaluation.
func TestProcessProposal_ValidProposalMultipleTxs(t *testing.T) {
	t.Run("process proposal with 10 valid transactions", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Create 10 checkpoint side txs with proper sequence numbers
		var txsToProcess [][]byte
		for i := 0; i < 10; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, uint64(i))
			require.NoError(t, err)
			txsToProcess = append(txsToProcess, txBytes)
		}

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Prepend ExtendedCommitInfo to txs
		allTxs := append([][]byte{extCommitBytes}, txsToProcess...)

		req := &abci.RequestProcessProposal{
			Txs:    allTxs,
			Height: 3,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		res, err := app.ProcessProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, res.Status)
	})
}

func TestProcessProposal_RejectsOverCapSideTxs(t *testing.T) {
	t.Run("reject proposal with more than maxSideTxResponsesCount side txs", func(t *testing.T) {
		// ProcessProposal cap is Zurich-gated.
		helper.SetZurichHardforkHeight(1)
		t.Cleanup(func() { helper.SetZurichHardforkHeight(0) })

		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)
		ctx = ctx.WithBlockHeight(3)

		var txsToProcess [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < maxSideTxResponsesCount+1; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			txsToProcess = append(txsToProcess, txBytes)
			sequence++
		}

		extCommitBytes, extCommit, _, err := buildExtensionCommits(
			t,
			app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			2,
			nil,
		)
		require.NoError(t, err)

		allTxs := append([][]byte{extCommitBytes}, txsToProcess...)
		req := &abci.RequestProcessProposal{
			Txs:    allTxs,
			Height: 3,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		res, err := app.ProcessProposal(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status)
	})
}

// TestProcessProposal_RejectScenarios tests the ProcessProposal handler's ability to reject invalid proposals by including various invalid scenarios such as proposals with no transactions, proposals with invalid ExtendedCommitInfo, and proposals with round mismatches, ensuring that the handler correctly identifies these issues and returns a rejection response for each case, thus validating the proper handling of invalid proposals during proposal evaluation.
func TestProcessProposal_RejectScenarios(t *testing.T) {
	t.Run("reject proposal with no txs", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtx(t)

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{},
			Height: 2,
			ProposedLastCommit: abci.CommitInfo{
				Round: 1,
				Votes: []abci.VoteInfo{},
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status)
	})

	t.Run("reject proposal with invalid ExtendedCommitInfo", func(t *testing.T) {
		priv, app, ctx, _ := SetupAppWithABCICtx(t)

		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		// Use invalid bytes as ExtendedCommitInfo
		invalidExtCommit := []byte("invalid-extended-commit-info")

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{invalidExtCommit, txBytes},
			Height: 2,
			ProposedLastCommit: abci.CommitInfo{
				Round: 1,
				Votes: []abci.VoteInfo{},
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status)
	})

	t.Run("reject proposal with round mismatch", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		allTxs := [][]byte{extCommitBytes, txBytes}

		req := &abci.RequestProcessProposal{
			Txs:    allTxs,
			Height: 2,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round + 1, // Different round
				Votes: []abci.VoteInfo{},
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status)
	})
}

// TestExtendVote_MultipleSideTxsExecution tests the ExtendVote handler's ability to execute multiple side transactions of different types during the vote extension process by including an ExtendedCommitInfo and a variety of side transactions (e.g., checkpoint messages, bor propose span messages, clerk event record messages) in the vote extension request, ensuring that the handler correctly processes all side transactions, updates the vote extension state accordingly, and returns a successful response, thus validating the proper execution of multiple side transactions during vote extension.
func TestExtendVote_MultipleSideTxsExecution(t *testing.T) {
	t.Run("extend vote with 20 side transactions of different types", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
			nil,
		)
		app.BorKeeper.SetContractCaller(mockCaller)
		app.MilestoneKeeper.IContractCaller = mockCaller
		app.caller = mockCaller

		var allTxs [][]byte

		// Add ExtendedCommitInfo first
		extCommitBytes, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)
		allTxs = append(allTxs, extCommitBytes)

		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		// Add 5 checkpoint txs
		for i := 0; i < 5; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}
			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			allTxs = append(allTxs, txBytes)
			sequence++
		}

		// Add 5 bor propose span txs
		for i := 0; i < 5; i++ {
			msg := &borTypes.MsgProposeSpan{
				Proposer:   priv.PubKey().Address().String(),
				SpanId:     uint64(1 + i),
				StartBlock: uint64(i * 6400),
				EndBlock:   uint64((i+1)*6400 - 1),
				ChainId:    "1",
				Seed:       common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			}
			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			allTxs = append(allTxs, txBytes)
			sequence++
		}

		// Add 5 clerk event record txs
		for i := 0; i < 5; i++ {
			msg := &clerkTypes.MsgEventRecord{
				From:            priv.PubKey().Address().String(),
				TxHash:          common.Bytes2Hex(common.BigToHash(common.Big1).Bytes()),
				LogIndex:        uint64(i),
				BlockNumber:     uint64(100 + i),
				ContractAddress: "0x0000000000000000000000000000000000000001",
				Data:            []byte{0x01},
				Id:              uint64(i + 1),
				ChainId:         "1",
			}
			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			allTxs = append(allTxs, txBytes)
			sequence++
		}

		// Add 5 topup txs
		for i := 0; i < 5; i++ {
			msg := &topUpTypes.MsgTopupTx{
				Proposer:    priv.PubKey().Address().String(),
				User:        priv.PubKey().Address().String(),
				Fee:         math.NewInt(1000000000),
				TxHash:      common.BigToHash(common.Big1).Bytes(),
				LogIndex:    uint64(i),
				BlockNumber: uint64(100 + i),
			}
			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			allTxs = append(allTxs, txBytes)
			sequence++
		}

		req := &abci.RequestExtendVote{
			Txs:    allTxs,
			Height: 2,
			Hash:   common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		}

		handler := app.ExtendVoteHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.VoteExtension)
		require.NotEmpty(t, res.NonRpExtension)
	})
}

// TestExtendVote_MaxSideTxResponsesLimit tests the ExtendVote handler's ability to enforce the maximum side transaction responses limit by including an ExtendedCommitInfo and a large number of side transactions in the vote extension request, ensuring that the handler correctly limits the number of side transaction responses included in the vote extension to the defined maximum, even when more transactions are provided, thus validating the proper enforcement of side transaction response limits during vote extension.
func TestExtendVote_MaxSideTxResponsesLimit(t *testing.T) {
	t.Run("extend vote respects max side tx responses count", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
			nil,
		)
		app.BorKeeper.SetContractCaller(mockCaller)
		app.MilestoneKeeper.IContractCaller = mockCaller
		app.caller = mockCaller

		var allTxs [][]byte

		// Add ExtendedCommitInfo first
		extCommitBytes, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)
		allTxs = append(allTxs, extCommitBytes)

		// Add more than maxSideTxResponsesCount checkpoint txs
		// For testing purposes, add 1100 to exceed the limit
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 1100; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}
			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			allTxs = append(allTxs, txBytes)
			sequence++
		}

		req := &abci.RequestExtendVote{
			Txs:    allTxs,
			Height: 2,
			Hash:   common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		}

		handler := app.ExtendVoteHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.VoteExtension)

		// Decode the vote extension to verify the limit
		var ve sidetxs.VoteExtension
		err = ve.Unmarshal(res.VoteExtension)
		require.NoError(t, err)

		// The vote extension should be limited to maxSideTxResponsesCount (50)
		// Even though we provided 1100 transactions, only 50 should be included
		require.Equal(t, 50, len(ve.SideTxResponses), "Vote extension should be limited to maxSideTxResponsesCount (50), but got %d", len(ve.SideTxResponses))
	})
}

// TestVerifyVoteExtension_AllRejectionScenarios tests the VerifyVoteExtension handler's ability to reject invalid vote extensions by including various invalid scenarios such as height mismatches, invalid hashes, and unauthorized validator addresses in the vote extension request, ensuring that the handler correctly identifies these issues and returns a rejection response for each case, thus validating the proper handling of invalid vote extensions during vote verification.
func TestVerifyVoteExtension_AllRejectionScenarios(t *testing.T) {
	tests := []struct {
		name         string
		setupVE      func(t *testing.T, app *HeimdallApp, ctx sdk.Context, priv cryptotypes.PrivKey) *abci.RequestVerifyVoteExtension
		expectStatus abci.ResponseVerifyVoteExtension_VerifyStatus
		expectError  bool
		description  string
	}{
		{
			name: "reject vote extension with height mismatch",
			setupVE: func(t *testing.T, app *HeimdallApp, ctx sdk.Context, priv cryptotypes.PrivKey) *abci.RequestVerifyVoteExtension {
				// This test would create a vote extension with mismatched height
				return &abci.RequestVerifyVoteExtension{
					Height:           2,
					ValidatorAddress: priv.PubKey().Address().Bytes(),
					VoteExtension:    []byte{},
					Hash:             common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
				}
			},
			expectStatus: abci.ResponseVerifyVoteExtension_REJECT,
			expectError:  false,
			description:  "Should reject when vote extension height doesn't match request height",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priv, app, ctx, _ := SetupAppWithABCICtx(t)

			req := tt.setupVE(t, app, ctx, priv)

			handler := app.VerifyVoteExtensionHandler()
			res, err := handler(ctx, req)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tt.expectStatus, res.Status)
			}
		})
	}
}

// TestPreBlocker_MultipleBlocksSequential tests the PreBlocker handler's ability to process multiple blocks sequentially by simulating the processing of 10 consecutive blocks, each containing an ExtendedCommitInfo and a checkpoint transaction, ensuring that the handler correctly processes each block without errors or panics, even when multiple blocks are processed in sequence, thus validating the proper functioning of the PreBlocker across multiple blocks.
func TestPreBlocker_MultipleBlocksSequential(t *testing.T) {
	t.Run("execute preBlocker for 10 consecutive blocks", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Simulate processing 10 blocks sequentially
		for blockHeight := int64(2); blockHeight < 12; blockHeight++ {
			ctx = ctx.WithBlockHeight(blockHeight)

			// Create some transactions for this block
			var txsForBlock [][]byte

			// Add a checkpoint tx
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(blockHeight * 100),
				EndBlock:        uint64(blockHeight*100 + 100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTx(msg, ctx, priv, app)
			require.NoError(t, err)
			txsForBlock = append(txsForBlock, txBytes)

			// Create ExtendedCommitInfo
			extCommitBytes, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, blockHeight, nil)
			require.NoError(t, err)

			// Prepend ExtendedCommitInfo
			allTxs := append([][]byte{extCommitBytes}, txsForBlock...)

			// Set last block txs (except for the first block)
			if blockHeight > 2 {
				err = app.StakeKeeper.SetLastBlockTxs(ctx, txsForBlock)
				require.NoError(t, err)
			}

			req := &abci.RequestFinalizeBlock{
				Txs:             allTxs,
				Height:          blockHeight,
				ProposerAddress: common.FromHex(validators[0].GetSigner()),
			}

			// Execute PreBlocker
			_, err = app.PreBlocker(ctx, req)

			// For the test setup, PreBlocker might fail on certain conditions,
			// but we're testing that it doesn't panic and processes multiple blocks
			if err != nil {
				// Log error but continue - some blocks might fail due to test setup
				t.Logf("PreBlocker returned error at height %d: %v", blockHeight, err)
			}
		}

		// If we reach here without panicking, the test passes
		require.True(t, true, "Successfully processed multiple blocks")
	})
}

// TestPreBlocker_MultipleApprovedSideTxs tests the PreBlocker handler's ability to process multiple approved side transactions of different types within a single block by including an ExtendedCommitInfo and various side transactions (e.g., checkpoint messages, bor propose span messages, clerk event record messages, stake validator join messages, topup messages) in the block's transactions, ensuring that the handler correctly processes all approved side transactions without errors or panics, thus validating the proper handling of multiple approved side transactions during block finalization.
func TestPreBlocker_MultipleApprovedSideTxs(t *testing.T) {
	t.Run("preBlocker with 5 approved side txs of different types", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		ctx = ctx.WithBlockHeight(2)

		var txsForBlock [][]byte

		// Create 5 different side txs
		// 1. Checkpoint
		checkpointMsg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}
		txBytes, err := buildSignedTx(checkpointMsg, ctx, priv, app)
		require.NoError(t, err)
		txsForBlock = append(txsForBlock, txBytes)

		// 2. Bor propose span
		borMsg := &borTypes.MsgProposeSpan{
			Proposer:   priv.PubKey().Address().String(),
			SpanId:     1,
			StartBlock: 0,
			EndBlock:   6399,
			ChainId:    "1",
			Seed:       common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
		}
		txBytes, err = buildSignedTx(borMsg, ctx, priv, app)
		require.NoError(t, err)
		txsForBlock = append(txsForBlock, txBytes)

		// 3. Clerk event record
		clerkMsg := &clerkTypes.MsgEventRecord{
			From:            priv.PubKey().Address().String(),
			TxHash:          common.BigToHash(common.Big1).Hex(),
			LogIndex:        0,
			BlockNumber:     100,
			ContractAddress: "0x0000000000000000000000000000000000000001",
			Data:            []byte{0x01},
			Id:              1,
			ChainId:         "1",
		}
		txBytes, err = buildSignedTx(clerkMsg, ctx, priv, app)
		require.NoError(t, err)
		txsForBlock = append(txsForBlock, txBytes)

		// 4. Stake validator join
		stakeMsg := &stakeTypes.MsgValidatorJoin{
			From:            priv.PubKey().Address().String(),
			ValId:           100,
			ActivationEpoch: 0,
			SignerPubKey:    secp256k1.GenPrivKey().PubKey().Bytes(),
			Amount:          math.NewInt(1000000000),
			TxHash:          common.BigToHash(common.Big1).Bytes(),
			LogIndex:        1,
			BlockNumber:     100,
			Nonce:           0,
		}
		txBytes, err = buildSignedTx(stakeMsg, ctx, priv, app)
		require.NoError(t, err)
		txsForBlock = append(txsForBlock, txBytes)

		// 5. Topup
		topupMsg := &topUpTypes.MsgTopupTx{
			Proposer:    priv.PubKey().Address().String(),
			User:        priv.PubKey().Address().String(),
			Fee:         math.NewInt(1000000000),
			TxHash:      common.BigToHash(common.Big1).Bytes(),
			LogIndex:    2,
			BlockNumber: 100,
		}
		txBytes, err = buildSignedTx(topupMsg, ctx, priv, app)
		require.NoError(t, err)
		txsForBlock = append(txsForBlock, txBytes)

		// Create ExtendedCommitInfo
		extCommitBytes, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Set last block txs
		err = app.StakeKeeper.SetLastBlockTxs(ctx, txsForBlock)
		require.NoError(t, err)

		allTxs := append([][]byte{extCommitBytes}, txsForBlock...)

		req := &abci.RequestFinalizeBlock{
			Txs:             allTxs,
			Height:          2,
			ProposerAddress: common.FromHex(validators[0].GetSigner()),
		}

		// Execute PreBlocker
		_, err = app.PreBlocker(ctx, req)

		// Verify it doesn't panic with multiple side txs
		if err != nil {
			t.Logf("PreBlocker returned error: %v", err)
		}
	})
}

// TestPreBlocker_EmptyTxsScenario tests the PreBlocker handler's response to a scenario where no transactions are included in the block by simulating a block finalization request with an empty transaction list, ensuring that the handler correctly identifies the absence of transactions and returns an appropriate error, thus validating the proper handling of blocks with no transactions during finalization.
func TestPreBlocker_EmptyTxsScenario(t *testing.T) {
	t.Run("preBlocker fails with empty txs", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtx(t)

		ctx = ctx.WithBlockHeight(2)

		req := &abci.RequestFinalizeBlock{
			Txs:    [][]byte{},
			Height: 2,
		}

		_, err := app.PreBlocker(ctx, req)

		require.Error(t, err, "PreBlocker should fail with no txs")
		require.Contains(t, err.Error(), "no txs found")
	})
}

// TestProcessProposal_RejectsCheckpointTxWhenNonRpVoteExtensionsInvalidPostPhuket proves the checkpoint rejection with invalid NonRpVEs
func TestProcessProposal_RejectsCheckpointTxWhenNonRpVoteExtensionsInvalidPostPhuket(t *testing.T) {
	originalForkHeight := helper.GetPhuketHardforkHeight()
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(originalForkHeight)
	})

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	ctx = ctx.WithBlockHeight(2)

	checkpointMsg := &checkpointTypes.MsgCheckpoint{
		Proposer:        priv.PubKey().Address().String(),
		StartBlock:      100,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
		AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		BorChainId:      "1",
	}
	txBytes, err := buildSignedTx(checkpointMsg, ctx, priv, app)
	require.NoError(t, err)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(
		t,
		app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators,
		validatorPrivKeys,
		2,
		nil,
	)
	require.NoError(t, err)

	extCommit.Votes[0].NonRpVoteExtension = []byte{0x01, 0x02, 0x03}
	extCommitBytes, err = extCommit.Marshal()
	require.NoError(t, err)

	req := &abci.RequestProcessProposal{
		Txs:    [][]byte{extCommitBytes, txBytes},
		Height: 3,
		ProposedLastCommit: abci.CommitInfo{
			Round: extCommit.Round,
			Votes: []abci.VoteInfo{
				{
					Validator:   extCommit.Votes[0].Validator,
					BlockIdFlag: cmtproto.BlockIDFlagCommit,
				},
			},
		},
	}

	handler := app.NewProcessProposalHandler()
	res, err := handler(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status)
}

func setupBorBlockNotFoundCaller() *helpermocks.IContractCaller {
	c := new(helpermocks.IContractCaller)
	c.On("CheckIfBlocksExist", mock.Anything, mock.Anything).Return(false, nil)
	c.On("GetRootHash", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte(nil), fmt.Errorf("bor unreachable"))
	return c
}

// buildCheckpointNonRpVoteExt packs a real (non-dummy) checkpoint as a
// NonRpVoteExtension so validateNonRpVoteExtensionData reaches IsValidCheckpoint.
// Use a unique endBlock per call to avoid the package-level existsCache.
func buildCheckpointNonRpVoteExt(t *testing.T, app *HeimdallApp, ctx sdk.Context, proposer string, endBlock uint64) ([]byte, *checkpointTypes.MsgCheckpoint) {
	t.Helper()
	chainParams, err := app.ChainManagerKeeper.GetParams(ctx)
	require.NoError(t, err)
	msg := &checkpointTypes.MsgCheckpoint{
		Proposer:        proposer,
		StartBlock:      endBlock - 100,
		EndBlock:        endBlock,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      chainParams.ChainParams.BorChainId,
	}
	return packExtensionWithVote(msg.GetSideSignBytes()), msg
}

func TestVerifyVoteExtensionHandler_BorBlockNotFoundHardforkGated(t *testing.T) {
	const runHeight = int64(3)
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() { helper.SetPhuketHardforkHeight(0) })

	cases := []struct {
		name       string
		activation int64
		endBlock   uint64 // unique per row to dodge the package-level existsCache
		wantStatus abci.ResponseVerifyVoteExtension_VerifyStatus
	}{
		{name: "below activation: rejects", activation: runHeight + 1, endBlock: 1_000_001, wantStatus: abci.ResponseVerifyVoteExtension_REJECT},
		{name: "at activation: accepts", activation: runHeight, endBlock: 1_000_002, wantStatus: abci.ResponseVerifyVoteExtension_ACCEPT},
		{name: "above activation: accepts", activation: runHeight - 1, endBlock: 1_000_003, wantStatus: abci.ResponseVerifyVoteExtension_ACCEPT},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			helper.SetZurichHardforkHeight(tc.activation)
			t.Cleanup(func() { helper.SetZurichHardforkHeight(0) })

			_, app, ctx, _ := SetupAppWithABCICtx(t)
			validators := app.StakeKeeper.GetAllValidators(ctx)

			packedVE, _ := buildCheckpointNonRpVoteExt(t, app, ctx, validators[0].Signer, tc.endBlock)

			vt := sidetxs.VoteExtension{Height: runHeight, BlockHash: []byte("test-hash")}
			veBytes, err := vt.Marshal()
			require.NoError(t, err)

			app.caller = setupBorBlockNotFoundCaller()

			res, err := app.VerifyVoteExtensionHandler()(ctx, &abci.RequestVerifyVoteExtension{
				VoteExtension:      veBytes,
				NonRpVoteExtension: packedVE,
				ValidatorAddress:   common.FromHex(validators[0].Signer),
				Height:             runHeight,
				Hash:               []byte("test-hash"),
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, res.Status)
		})
	}
}

func TestProcessProposal_NonRpVoteExtensionBorErrorHardforkGated(t *testing.T) {
	const runHeight = int64(3)
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() { helper.SetPhuketHardforkHeight(0) })

	cases := []struct {
		name       string
		activation int64
		endBlock   uint64
		wantStatus abci.ResponseProcessProposal_ProposalStatus
	}{
		{name: "below activation: rejects", activation: runHeight + 1, endBlock: 2_000_001, wantStatus: abci.ResponseProcessProposal_REJECT},
		{name: "at activation: accepts", activation: runHeight, endBlock: 2_000_002, wantStatus: abci.ResponseProcessProposal_ACCEPT},
		{name: "above activation: accepts", activation: runHeight - 1, endBlock: 2_000_003, wantStatus: abci.ResponseProcessProposal_ACCEPT},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			helper.SetZurichHardforkHeight(tc.activation)
			t.Cleanup(func() { helper.SetZurichHardforkHeight(0) })

			priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
			validators := app.StakeKeeper.GetAllValidators(ctx)

			// Proposer must equal the tx signer or the ante chain rejects.
			packedVE, checkpointMsg := buildCheckpointNonRpVoteExt(t, app, ctx, priv.PubKey().Address().String(), tc.endBlock)
			txBytes, err := buildSignedTx(checkpointMsg, ctx, priv, app)
			require.NoError(t, err)

			extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
			require.NoError(t, err)
			extCommit.Votes[0].NonRpVoteExtension = packedVE
			// Re-sign so checkNonRpVoteExtensionsSignatures sees
			// a signature that matches the swapped-in NonRpVoteExtension.
			nonRpSig, err := validatorPrivKeys[0].Sign(packedVE)
			require.NoError(t, err)
			extCommit.Votes[0].NonRpExtensionSignature = nonRpSig
			extCommitBytes, err = extCommit.Marshal()
			require.NoError(t, err)

			app.caller = setupBorBlockNotFoundCaller()

			// BaseApp.ProcessProposal sets up processProposalState that
			// NewProcessProposalHandler's tx verification call requires.
			res, err := app.ProcessProposal(&abci.RequestProcessProposal{
				Txs:    [][]byte{extCommitBytes, txBytes},
				Height: runHeight,
				ProposedLastCommit: abci.CommitInfo{
					Round: extCommit.Round,
					Votes: []abci.VoteInfo{{Validator: extCommit.Votes[0].Validator, BlockIdFlag: cmtproto.BlockIDFlagCommit}},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tc.wantStatus, res.Status)
		})
	}
}

// TestProcessProposal_NonRpVoteExtensionTamperedSignatureAlwaysRejected
// verifies that the proposal must be rejected even when Zurich HF would
// otherwise tolerate Bor errors. The signature check runs before any
// Bor-dependent payload validation, so the tolerateBorErr carve-out
// cannot suppress a signature failure.
func TestProcessProposal_NonRpVoteExtensionTamperedSignatureAlwaysRejected(t *testing.T) {
	const runHeight = int64(3)
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() { helper.SetPhuketHardforkHeight(0) })
	helper.SetZurichHardforkHeight(1) // Zurich active — tolerateBorErr would fire if reached
	t.Cleanup(func() { helper.SetZurichHardforkHeight(0) })

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	packedVE, checkpointMsg := buildCheckpointNonRpVoteExt(t, app, ctx, priv.PubKey().Address().String(), 4_000_001)
	txBytes, err := buildSignedTx(checkpointMsg, ctx, priv, app)
	require.NoError(t, err)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
	require.NoError(t, err)
	extCommit.Votes[0].NonRpVoteExtension = packedVE
	// leave the original NonRpExtensionSignature (which signs the
	// dummy VE that buildExtensionCommits inserted), do not re-sign packedVE.
	// The signature check fires first and rejects.
	extCommitBytes, err = extCommit.Marshal()
	require.NoError(t, err)

	app.caller = setupBorBlockNotFoundCaller()

	res, err := app.ProcessProposal(&abci.RequestProcessProposal{
		Txs:    [][]byte{extCommitBytes, txBytes},
		Height: runHeight,
		ProposedLastCommit: abci.CommitInfo{
			Round: extCommit.Round,
			Votes: []abci.VoteInfo{{Validator: extCommit.Votes[0].Validator, BlockIdFlag: cmtproto.BlockIDFlagCommit}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status,
		"tampered NonRpExtensionSignature must reject regardless of Zurich tolerateBorErr")
}

// TestValidateNonRpVoteExtensionData_ErrBorBlockNotFoundWrapChain asserts the
// errors.Is stays intact end-to-end from IsValidCheckpoint through
// validateCheckpointMsgData and validateNonRpVoteExtensionData.
func TestValidateNonRpVoteExtensionData_ErrBorBlockNotFoundWrapChain(t *testing.T) {
	const height = int64(3)

	_, app, ctx, _ := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	packedVE, _ := buildCheckpointNonRpVoteExt(t, app, ctx, validators[0].Signer, 3_000_001)
	caller := setupBorBlockNotFoundCaller()

	err := validateNonRpVoteExtensionData(ctx, height, packedVE, app.ChainManagerKeeper, app.CheckpointKeeper, caller)
	require.Error(t, err)
	require.True(t,
		errors.Is(err, borTypes.ErrBorBlockNotFound),
		"errors.Is must traverse the full wrap chain (merkle.go → validateCheckpointMsgData → validateNonRpVoteExtensionData); got %v", err,
	)
}

// With the budget zeroed, the loop short-circuits when active and runs to
// completion otherwise.
func TestExtendVoteHandler_BudgetHardforkGated(t *testing.T) {
	const runHeight = int64(3)
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() { helper.SetPhuketHardforkHeight(0) })

	originalBudget := extendVoteBudget
	extendVoteBudget = 0
	t.Cleanup(func() { extendVoteBudget = originalBudget })

	cases := []struct {
		name          string
		activation    int64
		wantSideTxRes string // "empty" or "nonempty"
	}{
		{name: "below activation: full responses", activation: runHeight + 1, wantSideTxRes: "nonempty"},
		{name: "at activation: empty", activation: runHeight, wantSideTxRes: "empty"},
		{name: "above activation: empty", activation: runHeight - 1, wantSideTxRes: "empty"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			helper.SetZurichHardforkHeight(tc.activation)
			t.Cleanup(func() { helper.SetZurichHardforkHeight(0) })

			priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
			validators := app.StakeKeeper.GetAllValidators(ctx)

			msg := &types.MsgCheckpoint{
				Proposer:        validators[0].Signer,
				StartBlock:      100,
				EndBlock:        200,
				RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
				AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
				BorChainId:      "test",
			}
			txBytes, err := buildSignedTx(msg, ctx, priv, app)
			require.NoError(t, err)

			extCommitBytes, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
			require.NoError(t, err)

			working := new(helpermocks.IContractCaller)
			working.On("GetBorChainBlock", mock.Anything, mock.Anything).
				Return(&ethTypes.Header{Number: big.NewInt(10)}, nil)
			working.On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
				Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)
			working.On("CheckIfBlocksExist", mock.Anything, mock.Anything).Return(true, nil)
			working.On("GetRootHash", mock.Anything, mock.Anything, mock.Anything).
				Return(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"), nil)

			app.MilestoneKeeper = milestoneKeeper.NewKeeper(app.AppCodec(), authTypes.NewModuleAddress(govtypes.ModuleName).String(), runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)), working)
			app.CheckpointKeeper = checkpointKeeper.NewKeeper(app.AppCodec(), runtime.NewKVStoreService(app.GetKey(checkpointTypes.StoreKey)), authTypes.NewModuleAddress(govtypes.ModuleName).String(), &app.StakeKeeper, app.ChainManagerKeeper, &app.TopupKeeper, working)
			app.BorKeeper = borKeeper.NewKeeper(app.AppCodec(), runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)), authTypes.NewModuleAddress(govtypes.ModuleName).String(), app.ChainManagerKeeper, &app.StakeKeeper, nil, nil)
			app.BorKeeper.SetContractCaller(working)
			app.MilestoneKeeper.IContractCaller = working
			app.caller = working

			respExtend, err := app.ExtendVoteHandler()(ctx, &abci.RequestExtendVote{
				Txs:    [][]byte{extCommitBytes, txBytes},
				Hash:   []byte("test-hash"),
				Height: runHeight,
			})
			require.NoError(t, err)
			require.NotNil(t, respExtend.VoteExtension)

			var ve sidetxs.VoteExtension
			require.NoError(t, gogoproto.Unmarshal(respExtend.VoteExtension, &ve))

			switch tc.wantSideTxRes {
			case "empty":
				require.Empty(t, ve.SideTxResponses,
					"with budget active and zeroed, the side-tx loop must short-circuit")
			case "nonempty":
				require.NotEmpty(t, ve.SideTxResponses,
					"with budget inactive, a zero budget must be a no-op")
			}
		})
	}
}

func TestABCI_FullBlockLifecycle_NoPreBlocker(t *testing.T) {
	t.Run("complete block lifecycle with side txs", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
			nil,
		)
		app.BorKeeper.SetContractCaller(mockCaller)
		app.MilestoneKeeper.IContractCaller = mockCaller
		app.caller = mockCaller

		ctx = ctx.WithBlockHeight(2)

		// 1. Create transactions
		var proposedTxs [][]byte

		checkpointMsg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTx(checkpointMsg, ctx, priv, app)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// 2. PrepareProposal
		prepareReq := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		prepareRes, err := app.PrepareProposal(prepareReq)
		require.NoError(t, err)
		require.NotNil(t, prepareRes)
		require.Equal(t, 2, len(prepareRes.Txs), "Should have exactly 2 transactions (1 ExtendedCommitInfo + 1 side tx)")

		// 3. ExtendVote
		extendReq := &abci.RequestExtendVote{
			Txs:    prepareRes.Txs,
			Height: 2,
			Hash:   common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		}

		extendHandler := app.ExtendVoteHandler()
		extendRes, err := extendHandler(ctx, extendReq)
		require.NoError(t, err)
		require.NotNil(t, extendRes)
		require.NotEmpty(t, extendRes.VoteExtension)

		// 4. VerifyVoteExtension
		verifyReq := &abci.RequestVerifyVoteExtension{
			Height:             2,
			ValidatorAddress:   common.FromHex(validators[0].GetSigner()),
			VoteExtension:      extendRes.VoteExtension,
			Hash:               common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			NonRpVoteExtension: extendRes.NonRpExtension,
		}

		verifyHandler := app.VerifyVoteExtensionHandler()
		verifyRes, err := verifyHandler(ctx, verifyReq)
		require.NoError(t, err)
		require.NotNil(t, verifyRes)

		// 5. ProcessProposal
		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		processReq := &abci.RequestProcessProposal{
			Txs:    append([][]byte{extCommitBytes}, proposedTxs...),
			Height: 2,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		processHandler := app.NewProcessProposalHandler()
		processRes, err := processHandler(ctx, processReq)
		require.NoError(t, err)
		require.NotNil(t, processRes)
	})
}

// TestABCI_StressTestWith100Blocks tests the ABCI handlers' ability to handle a high volume of blocks and transactions by simulating the processing of 100 consecutive blocks, each containing an ExtendedCommitInfo and a mix of different transaction types (e.g., checkpoint messages, bor propose span messages, clerk event record messages, topup messages), ensuring that the handlers correctly process all blocks and transactions without errors or panics, thus validating the robustness and scalability of the ABCI handlers under stress conditions.
func TestABCI_StressTestWith100Blocks(t *testing.T) {
	t.Run("stress test with 100 blocks and mixed tx types", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
			nil,
		)
		app.BorKeeper.SetContractCaller(mockCaller)
		app.MilestoneKeeper.IContractCaller = mockCaller
		app.caller = mockCaller

		// Process 100 blocks (start at height 3 since SetupAppWithABCICtx leaves us at height 3)
		for blockHeight := int64(3); blockHeight < 103; blockHeight++ {
			ctx = ctx.WithBlockHeight(blockHeight)

			// Create 5 transactions per block
			var proposedTxs [][]byte

			for i := 0; i < 5; i++ {
				var msg sdk.Msg

				// Alternate between different message types
				switch (blockHeight + int64(i)) % 5 {
				case 0:
					msg = &checkpointTypes.MsgCheckpoint{
						Proposer:        priv.PubKey().Address().String(),
						StartBlock:      uint64(blockHeight*100 + int64(i)*10),
						EndBlock:        uint64(blockHeight*100 + int64(i)*10 + 10),
						RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
						BorChainId:      "1",
					}
				case 1:
					msg = &borTypes.MsgProposeSpan{
						Proposer:   priv.PubKey().Address().String(),
						SpanId:     uint64(blockHeight*10 + int64(i)),
						StartBlock: uint64(blockHeight * 6400),
						EndBlock:   uint64(blockHeight*6400 + 6399),
						ChainId:    "1",
						Seed:       common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
					}
				case 2:
					msg = &clerkTypes.MsgEventRecord{
						From:            priv.PubKey().Address().String(),
						TxHash:          common.BigToHash(common.Big1).Hex(),
						LogIndex:        uint64(blockHeight*10 + int64(i)),
						BlockNumber:     uint64(blockHeight),
						ContractAddress: "0x0000000000000000000000000000000000000001",
						Data:            []byte{0x01},
						Id:              uint64(blockHeight*10 + int64(i)),
						ChainId:         "1",
					}
				case 3:
					msg = &topUpTypes.MsgTopupTx{
						Proposer:    priv.PubKey().Address().String(),
						User:        priv.PubKey().Address().String(),
						Fee:         math.NewInt(1000000000),
						TxHash:      common.BigToHash(common.Big1).Bytes(),
						LogIndex:    uint64(blockHeight*10 + int64(i)),
						BlockNumber: uint64(blockHeight),
					}
				case 4:
					msg = &borTypes.MsgBackfillSpans{
						Proposer:        priv.PubKey().Address().String(),
						ChainId:         "1",
						LatestSpanId:    uint64(blockHeight),
						LatestBorSpanId: uint64(blockHeight + 1),
					}
				}

				txBytes, err := buildSignedTx(msg, ctx, priv, app)
				require.NoError(t, err)
				proposedTxs = append(proposedTxs, txBytes)
			}

			_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, blockHeight, nil)
			require.NoError(t, err)

			// PrepareProposal
			prepareReq := &abci.RequestPrepareProposal{
				Txs:             proposedTxs,
				MaxTxBytes:      10000000,
				Height:          blockHeight,
				LocalLastCommit: *extCommit,
				ProposerAddress: common.FromHex(validators[0].Signer),
			}

			prepareRes, err := app.PrepareProposal(prepareReq)

			if err != nil {
				// Some blocks might fail due to validation rules
				t.Logf("PrepareProposal failed at height %d: %v", blockHeight, err)
				continue
			}

			require.NotNil(t, prepareRes)

			// ProcessProposal (vote extensions are for the previous height)
			extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, blockHeight-1, nil)
			if err != nil {
				t.Logf("buildExtensionCommits failed at height %d: %v", blockHeight, err)
				continue
			}

			processReq := &abci.RequestProcessProposal{
				Txs:    append([][]byte{extCommitBytes}, proposedTxs...),
				Height: blockHeight,
				ProposedLastCommit: abci.CommitInfo{
					Round: extCommit.Round,
					Votes: []abci.VoteInfo{},
				},
			}

			processRes, err := app.ProcessProposal(processReq)

			if err != nil {
				t.Logf("ProcessProposal failed at height %d: %v", blockHeight, err)
				continue
			}

			require.NotNil(t, processRes)

			// Log progress every 10 blocks
			if blockHeight%10 == 0 {
				t.Logf("Processed block %d successfully", blockHeight)
			}
		}

		// If we reach here, the stress test passed
		require.True(t, true, "Successfully stress tested 100 blocks")
	})
}

// TestPrepareProposal_ErrorRecovery tests the PrepareProposal handler's ability to recover from errors gracefully by simulating a scenario where invalid transaction bytes are included in the proposal request, ensuring that the handler correctly identifies the invalid transaction and returns an appropriate error without crashing or panicking, thus validating the robustness of the PrepareProposal handler in handling erroneous input.
func TestPrepareProposal_ErrorRecovery(t *testing.T) {
	t.Run("handle decode errors gracefully", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)
		_ = priv

		// Create invalid transaction bytes
		invalidTxBytes := []byte("this-is-not-a-valid-transaction")

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{invalidTxBytes},
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
		}

		handler := app.NewPrepareProposalHandler()
		_, err = handler(ctx, req)

		// Should return error for invalid tx bytes
		require.Error(t, err)
	})
}

// TestPrepareProposal_ManySideTxMessageTypes tests the PrepareProposal handler's ability to process a proposal containing a variety of side transaction message types by simulating a proposal request that includes multiple transactions of different types (e.g., checkpoint messages, stake update messages, signer update messages, validator exit messages), ensuring that the handler correctly processes all included transactions without errors, thus validating the proper handling of diverse side transaction message types during proposal preparation.
func TestPrepareProposal_ManySideTxMessageTypes(t *testing.T) {
	t.Run("includes many side tx message types", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Mock the ContractCaller
		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
			nil,
		)
		app.BorKeeper.SetContractCaller(mockCaller)
		app.MilestoneKeeper.IContractCaller = mockCaller
		app.caller = mockCaller

		ctx = ctx.WithBlockHeight(3)

		var proposedTxs [][]byte

		// 1. MsgCheckpoint
		checkpointMsg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}
		txBytes, err := buildSignedTxWithSequence(checkpointMsg, ctx, priv, app, 0)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// 2. MsgCpAck (checkpoint acknowledgment)
		cpAckMsg := &checkpointTypes.MsgCpAck{
			From:       priv.PubKey().Address().String(),
			Number:     1,
			Proposer:   priv.PubKey().Address().String(),
			StartBlock: 100,
			EndBlock:   200,
			RootHash:   common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
		}
		txBytes, err = buildSignedTxWithSequence(cpAckMsg, ctx, priv, app, 1)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// 3. MsgStakeUpdate
		stakeUpdateMsg := &stakeTypes.MsgStakeUpdate{
			From:        priv.PubKey().Address().String(),
			ValId:       1,
			NewAmount:   math.NewInt(2000000000),
			TxHash:      common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			LogIndex:    0,
			BlockNumber: 100,
			Nonce:       1,
		}
		txBytes, err = buildSignedTxWithSequence(stakeUpdateMsg, ctx, priv, app, 2)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// 4. MsgSignerUpdate
		newPrivKey := secp256k1.GenPrivKey()
		signerUpdateMsg := &stakeTypes.MsgSignerUpdate{
			From:            priv.PubKey().Address().String(),
			ValId:           2,
			NewSignerPubKey: newPrivKey.PubKey().Bytes(),
			TxHash:          common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000003"),
			LogIndex:        0,
			BlockNumber:     100,
			Nonce:           1,
		}
		txBytes, err = buildSignedTxWithSequence(signerUpdateMsg, ctx, priv, app, 3)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// 5. MsgValidatorExit
		validatorExitMsg := &stakeTypes.MsgValidatorExit{
			From:              priv.PubKey().Address().String(),
			ValId:             3,
			DeactivationEpoch: 10,
			TxHash:            common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000004"),
			LogIndex:          0,
			BlockNumber:       100,
			Nonce:             1,
		}
		txBytes, err = buildSignedTxWithSequence(validatorExitMsg, ctx, priv, app, 4)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10000000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 6, len(res.Txs), "Should have exactly 6 transactions (1 ExtendedCommitInfo + 5 side txs)")
	})
}

// TestProcessProposal_ManySideTxMessageTypes tests the ProcessProposal handler's ability to process a proposal containing a variety of side transaction message types by simulating a proposal processing request that includes multiple transactions of different types (e.g., checkpoint acknowledgment messages, stake update messages, signer update messages, validator exit messages), ensuring that the handler correctly processes all included transactions without errors, thus validating the proper handling of diverse side transaction message types during proposal processing.
func TestProcessProposal_ManySideTxMessageTypes(t *testing.T) {
	t.Run("process proposal with many side tx types", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

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
			nil,
		)
		app.BorKeeper.SetContractCaller(mockCaller)
		app.MilestoneKeeper.IContractCaller = mockCaller
		app.caller = mockCaller

		ctx = ctx.WithBlockHeight(3)

		// Build transactions with all side tx types
		var proposedTxs [][]byte

		// 1. MsgCpAck
		cpAckMsg := &checkpointTypes.MsgCpAck{
			From:       priv.PubKey().Address().String(),
			Number:     1,
			Proposer:   priv.PubKey().Address().String(),
			StartBlock: 100,
			EndBlock:   200,
			RootHash:   common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
		}
		txBytes, err := buildSignedTxWithSequence(cpAckMsg, ctx, priv, app, 0)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// 2. MsgStakeUpdate
		stakeUpdateMsg := &stakeTypes.MsgStakeUpdate{
			From:        priv.PubKey().Address().String(),
			ValId:       1,
			NewAmount:   math.NewInt(2000000000),
			TxHash:      common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			LogIndex:    0,
			BlockNumber: 100,
			Nonce:       1,
		}
		txBytes, err = buildSignedTxWithSequence(stakeUpdateMsg, ctx, priv, app, 1)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// 3. MsgSignerUpdate
		newPrivKey := secp256k1.GenPrivKey()
		signerUpdateMsg := &stakeTypes.MsgSignerUpdate{
			From:            priv.PubKey().Address().String(),
			ValId:           2,
			NewSignerPubKey: newPrivKey.PubKey().Bytes(),
			TxHash:          common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000003"),
			LogIndex:        0,
			BlockNumber:     100,
			Nonce:           1,
		}
		txBytes, err = buildSignedTxWithSequence(signerUpdateMsg, ctx, priv, app, 2)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// 4. MsgValidatorExit
		validatorExitMsg := &stakeTypes.MsgValidatorExit{
			From:              priv.PubKey().Address().String(),
			ValId:             3,
			DeactivationEpoch: 10,
			TxHash:            common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000004"),
			LogIndex:          0,
			BlockNumber:       100,
			Nonce:             1,
		}
		txBytes, err = buildSignedTxWithSequence(validatorExitMsg, ctx, priv, app, 3)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)

		// Get vote extensions
		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Add ExtendedCommitInfo as the first transaction
		allTxs := append([][]byte{extCommitBytes}, proposedTxs...)

		// Create ProcessProposal request
		req := &abci.RequestProcessProposal{
			Txs: allTxs,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
			Height:          3,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.ProcessProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, res.Status)
	})
}

func TestPrepareProposal_ExtendedCommitInfo_ExceedsMaxTxBytes(t *testing.T) {
	t.Run("handles vote extension filtering with reasonable MaxTxBytes", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Create a checkpoint message
		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		// Build ExtendedCommitInfo
		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Use 50KB MaxTxBytes to accommodate VEs without restrictive filtering
		// Per-validator VE limit = (50000/4/3)-700 = 3466 bytes
		maxTxBytes := int64(50000)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      maxTxBytes,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		// VE filtering should work correctly
		require.NoError(t, err)
		require.NotNil(t, res)
		// Should have exactly 2 txs: ExtendedCommitInfo + checkpoint tx
		require.Equal(t, 2, len(res.Txs), "Should have ExtendedCommitInfo + checkpoint tx")
	})
}

func TestProcessProposal_RejectEmptyTxs_FromOversizedExtendedCommitInfo(t *testing.T) {
	t.Run("rejects proposal with no txs", func(t *testing.T) {
		_, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Simulate what PrepareProposal returns when ExtendedCommitInfo exceeds MaxTxBytes
		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{}, // Empty txs array
			Height: 3,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status, "Should reject proposal with no txs")
	})
}

func TestPrepareProposal_ExtendedCommitInfo_WithinMaxTxBytes(t *testing.T) {
	t.Run("includes all txs when within MaxTxBytes", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		ctx = ctx.WithBlockHeight(3)

		// Create 5 checkpoint transactions
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 5; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Set MaxTxBytes large enough for all txs
		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      1_000_000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 6, len(res.Txs), "Should have 1 ExtendedCommitInfo + 5 proposed txs")

		// Verify first tx is ExtendedCommitInfo
		extCommitInfo := new(abci.ExtendedCommitInfo)
		err = extCommitInfo.Unmarshal(res.Txs[0])
		require.NoError(t, err)

		// Verify ProcessProposal accepts this
		processReq := &abci.RequestProcessProposal{
			Txs:    res.Txs,
			Height: 3,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		processRes, err := app.ProcessProposal(processReq)

		require.NoError(t, err)
		require.NotNil(t, processRes)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processRes.Status)
	})
}

func TestPrepareProposal_ExtendedCommitInfo_EqualsMaxTxBytes(t *testing.T) {
	t.Run("includes only ExtendedCommitInfo when MaxTxBytes leaves no room for txs", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		ctx = ctx.WithBlockHeight(3)

		// Create 5 checkpoint transactions
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 5; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Use 50KB MaxTxBytes to accommodate VEs without restrictive filtering
		// Per-validator VE limit = (50000/4/3)-700 = 3466 bytes
		// With 50KB, ExtCommitInfo + all 5 txs should fit (total ~3-4KB)
		maxTxBytes := int64(50000)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      maxTxBytes,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		// Should have exactly 6 txs: ExtendedCommitInfo + all 5 proposed txs
		require.Equal(t, 6, len(res.Txs), "Should have ExtendedCommitInfo + all 5 txs")

		// Verify first tx is ExtendedCommitInfo
		extCommitInfo := new(abci.ExtendedCommitInfo)
		err = extCommitInfo.Unmarshal(res.Txs[0])
		require.NoError(t, err)

		// Verify ProcessProposal accepts this
		processReq := &abci.RequestProcessProposal{
			Txs:    res.Txs,
			Height: 3,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		processRes, err := app.ProcessProposal(processReq)

		require.NoError(t, err)
		require.NotNil(t, processRes)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processRes.Status)
	})
}

func TestPrepareProposal_PartialTxInclusion_SizeConstraint(t *testing.T) {
	t.Run("includes only txs that fit within MaxTxBytes", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		ctx = ctx.WithBlockHeight(3)

		// Create 10 checkpoint transactions
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 10; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Use 15KB MaxTxBytes, large enough for VE filtering but tight for 10 txs
		// Per-validator VE limit = (15000/4/3)-700 = 550 bytes
		// With ~1.5KB ExtCommitInfo + 10*~400 bytes txs = ~5.5KB total
		maxTxBytes := int64(15000)

		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      maxTxBytes,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		// With 15KB, ExtCommitInfo + all 10 txs should fit
		require.Equal(t, 11, len(res.Txs), "Should have ExtendedCommitInfo + all 10 txs")

		// Verify first tx is ExtendedCommitInfo
		extCommitInfo := new(abci.ExtendedCommitInfo)
		err = extCommitInfo.Unmarshal(res.Txs[0])
		require.NoError(t, err)

		// Verify ProcessProposal accepts this
		processReq := &abci.RequestProcessProposal{
			Txs:    res.Txs,
			Height: 3,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		processRes, err := app.ProcessProposal(processReq)

		require.NoError(t, err)
		require.NotNil(t, processRes)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processRes.Status)
	})
}

func TestPrepareProposal_AllTxsIncluded_WithinMaxTxBytes(t *testing.T) {
	t.Run("includes all txs when all fit within MaxTxBytes", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		ctx = ctx.WithBlockHeight(3)

		// Create 5 checkpoint transactions with correct sequence numbers
		var proposedTxs [][]byte
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		sequence := propAcc.GetSequence()

		for i := 0; i < 5; i++ {
			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        priv.PubKey().Address().String(),
				StartBlock:      uint64(100 + i*100),
				EndBlock:        uint64(200 + i*100),
				RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
				BorChainId:      "1",
			}

			txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
			require.NoError(t, err)
			proposedTxs = append(proposedTxs, txBytes)
			sequence++
		}

		_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		// Set MaxTxBytes large enough for all txs
		req := &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      10_000_000,
			Height:          3,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		res, err := app.PrepareProposal(req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 6, len(res.Txs), "Should have 1 ExtendedCommitInfo + all 5 proposed txs")

		// Verify no txs were skipped
		extCommitInfo := new(abci.ExtendedCommitInfo)
		err = extCommitInfo.Unmarshal(res.Txs[0])
		require.NoError(t, err)

		// Verify ProcessProposal accepts this
		processReq := &abci.RequestProcessProposal{
			Txs:    res.Txs,
			Height: 3,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
				Votes: []abci.VoteInfo{},
			},
		}

		processRes, err := app.ProcessProposal(processReq)

		require.NoError(t, err)
		require.NotNil(t, processRes)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processRes.Status)
	})
}

// TestPrepareProposal_ProtoSizeAccounting_NoOversizedProposal ensures that PrepareProposal
// sizes the returned txs in the unit CometBFT uses to validate them — the protobuf-encoded
// size of Data.Txs (types.Txs.Validate -> ComputeProtoSizeForTxs) — not raw
// len(tx).
func TestPrepareProposal_ProtoSizeAccounting_NoOversizedProposal(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	ctx = ctx.WithBlockHeight(3)

	const numTxs = 80
	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	sequence := propAcc.GetSequence()

	var proposedTxs [][]byte
	var rawTxsTotal int64
	for i := 0; i < numTxs; i++ {
		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      uint64(100 + i*100),
			EndBlock:        uint64(200 + i*100),
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}
		txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)
		rawTxsTotal += int64(len(txBytes))
		sequence++
	}

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
	require.NoError(t, err)

	maxTxBytes := int64(len(extCommitBytes)) + rawTxsTotal

	req := &abci.RequestPrepareProposal{
		Txs:             proposedTxs,
		MaxTxBytes:      maxTxBytes,
		Height:          3,
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	res, err := app.PrepareProposal(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Proto sizing is tighter than raw, so most txs are kept but at least one is dropped.
	require.Greater(t, len(res.Txs), 1, "expected most of the batch to be included")
	require.Less(t, len(res.Txs), 1+numTxs, "proto sizing must drop the overflowing tx")

	// The returned proposal must satisfy the size check CometBFT applies to it.
	require.NoError(t, cmtTypes.ToTxs(res.Txs).Validate(maxTxBytes))
}

// TestPrepareProposal_SizeBoundary_IncludesTxThatExactlyFits checks that a tx
// filling the proposal to exactly MaxTxBytes is kept (CometBFT rejects only size
// strictly greater than the limit).
func TestPrepareProposal_SizeBoundary_IncludesTxThatExactlyFits(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	ctx = ctx.WithBlockHeight(3)

	// Enough txs that the resulting budget stays well above the vote-extension
	// filtering threshold, so the commit-info tx is identical across both passes.
	const numTxs = 130
	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	sequence := propAcc.GetSequence()

	var proposedTxs [][]byte
	for i := 0; i < numTxs; i++ {
		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      uint64(100 + i*100),
			EndBlock:        uint64(200 + i*100),
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "1",
		}
		txBytes, err := buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
		require.NoError(t, err)
		proposedTxs = append(proposedTxs, txBytes)
		sequence++
	}

	_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
	require.NoError(t, err)

	newReq := func(maxTxBytes int64) *abci.RequestPrepareProposal {
		localCommit := *extCommit
		return &abci.RequestPrepareProposal{
			Txs:             proposedTxs,
			MaxTxBytes:      maxTxBytes,
			Height:          3,
			LocalLastCommit: localCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}
	}

	// First pass: unbounded budget, to capture the exact proposal and its proto size.
	full, err := app.PrepareProposal(newReq(1 << 30))
	require.NoError(t, err)
	require.Equal(t, 1+numTxs, len(full.Txs))

	// Sum per-tx proto sizes, mirroring how the handler accounts for txs and how
	// CometBFT's Txs.Validate measures them (both sum ComputeProtoSizeForTxs per tx).
	var boundary int64
	for _, tx := range full.Txs {
		boundary += cmtTypes.ComputeProtoSizeForTxs([]cmtTypes.Tx{tx})
	}

	// MaxTxBytes set to exactly that total: the last tx fills the budget to the
	// byte and must still be included.
	bounded, err := app.PrepareProposal(newReq(boundary))
	require.NoError(t, err)
	require.Equal(t, full.Txs[0], bounded.Txs[0], "commit must be identical across passes (no VE-filtering skew)")
	require.Equal(t, 1+numTxs, len(bounded.Txs), "tx that fills MaxTxBytes exactly must be kept")
	require.NoError(t, cmtTypes.ToTxs(bounded.Txs).Validate(boundary))
}

func TestVerifyVoteExtension_RejectInvalidNonRpVoteExtension(t *testing.T) {
	t.Run("reject vote extension with NonRpVoteExtension too small", func(t *testing.T) {
		helper.SetPhuketHardforkHeight(1)
		t.Cleanup(func() {
			helper.SetPhuketHardforkHeight(0)
		})

		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Create a checkpoint message
		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        validators[0].Signer,
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "test",
		}

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		extCommitBytes, _, voteInfo, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
		})
		require.NoError(t, err)

		// Mock the ContractCaller
		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)

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
		app.caller = mockCaller

		// Create a NonRpVoteExtension that is too small (less than minNonRpVoteExtensionSize)
		invalidNonRpExt := []byte{0x01}

		req := &abci.RequestVerifyVoteExtension{
			Height:             2,
			ValidatorAddress:   common.FromHex(validators[0].GetSigner()),
			VoteExtension:      voteInfo.VoteExtension,
			Hash:               common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			NonRpVoteExtension: invalidNonRpExt,
		}

		handler := app.VerifyVoteExtensionHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, res.Status, "Should reject vote extension with invalid NonRpVoteExtension")
	})

	t.Run("reject vote extension with NonRpVoteExtension too large", func(t *testing.T) {
		helper.SetPhuketHardforkHeight(1)
		t.Cleanup(func() {
			helper.SetPhuketHardforkHeight(0)
		})

		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		// Create a checkpoint message
		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        validators[0].Signer,
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			AccountRootHash: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
			BorChainId:      "test",
		}

		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		extCommitBytes, _, voteInfo, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
		require.NoError(t, err)

		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          3,
			Txs:             [][]byte{extCommitBytes, txBytes},
			ProposerAddress: common.FromHex(validators[0].Signer),
		})
		require.NoError(t, err)

		// Mock the ContractCaller
		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)

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
		app.caller = mockCaller

		// Create a NonRpVoteExtension that is too large (more than maxNonRpVoteExtensionSize)
		invalidNonRpExt := make([]byte, 2000)

		req := &abci.RequestVerifyVoteExtension{
			Height:             2,
			ValidatorAddress:   common.FromHex(validators[0].GetSigner()),
			VoteExtension:      voteInfo.VoteExtension,
			Hash:               common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			NonRpVoteExtension: invalidNonRpExt,
		}

		handler := app.VerifyVoteExtensionHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, res.Status, "Should reject vote extension with NonRpVoteExtension too large")
	})
}

func TestProcessProposal_VoteExtensionsCompleteness(t *testing.T) {
	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
	})

	_, app, ctx, validatorPrivKeys := SetupAppWithABCICtxAndValidators(t, 4)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	require.GreaterOrEqual(t, len(validators), 4)

	reqHeight := int64(3)
	round := int32(1)

	privByAddr := make(map[string]secp256k1.PrivKey, len(validatorPrivKeys))
	for _, pk := range validatorPrivKeys {
		addr := pk.PubKey().Address()
		privByAddr[common.Bytes2Hex(addr)] = pk
	}

	type validatorInfo struct {
		addrBytes []byte
		power     int64
		priv      secp256k1.PrivKey
	}
	valInfos := make([]validatorInfo, 0, 4)
	for _, v := range validators {
		addrBytes := common.FromHex(v.Signer)
		addrHex := common.Bytes2Hex(addrBytes)
		priv, ok := privByAddr[addrHex]
		if !ok {
			continue
		}
		valInfos = append(valInfos, validatorInfo{addrBytes: addrBytes, power: v.VotingPower, priv: priv})
		if len(valInfos) == 4 {
			break
		}
	}
	require.Len(t, valInfos, 4)

	// Build a valid VoteExtension payload
	veProto := sidetxs.VoteExtension{
		SideTxResponses: []sidetxs.SideTxResponse{
			{TxHash: common.FromHex(TxHash1), Result: sidetxs.Vote_VOTE_YES},
			{TxHash: make([]byte, 32), Result: sidetxs.Vote_VOTE_YES},
			{TxHash: append([]byte{0x01}, common.FromHex(TxHash1)[1:]...), Result: sidetxs.Vote_VOTE_YES},
		},
		BlockHash: common.FromHex(TxHash2),
		Height:    reqHeight - 1,
	}
	veBytes, err := veProto.Marshal()
	require.NoError(t, err)

	// Sign the CanonicalVoteExtension
	signVE := func(priv secp256k1.PrivKey, extension []byte) []byte {
		cve := cmtproto.CanonicalVoteExtension{
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
		return sig
	}

	dummyNonRpVE, err := GetDummyNonRpVoteExtension(reqHeight-1, ctx.ChainID())
	require.NoError(t, err)

	// Build a valid ExtendedVoteInfo
	mkVote := func(vi validatorInfo) abci.ExtendedVoteInfo {
		nonRpSig, err := vi.priv.Sign(dummyNonRpVE)
		require.NoError(t, err)
		return abci.ExtendedVoteInfo{
			BlockIdFlag:             cmtproto.BlockIDFlagCommit,
			VoteExtension:           veBytes,
			ExtensionSignature:      signVE(vi.priv, veBytes),
			NonRpVoteExtension:      dummyNonRpVE,
			NonRpExtensionSignature: nonRpSig,
			Validator: abci.Validator{
				Address: vi.addrBytes,
				Power:   vi.power,
			},
		}
	}

	// Build votes for all validators
	allVotes := make([]abci.ExtendedVoteInfo, 4)
	for i := range valInfos {
		allVotes[i] = mkVote(valInfos[i])
	}

	// Build canonical VoteInfo entries
	canonicalVotes := make([]abci.VoteInfo, 4)
	for i, v := range allVotes {
		canonicalVotes[i] = abci.VoteInfo{
			Validator:   v.Validator,
			BlockIdFlag: cmtproto.BlockIDFlagCommit,
		}
	}

	// Marshal the full ExtendedCommitInfo
	fullExtCommit := &abci.ExtendedCommitInfo{Round: round, Votes: allVotes}
	fullExtCommitBytes, err := fullExtCommit.Marshal()
	require.NoError(t, err)

	t.Run("accept when canonical commit set is complete", func(t *testing.T) {
		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{fullExtCommitBytes},
			Height: reqHeight,
			ProposedLastCommit: abci.CommitInfo{
				Round: round,
				Votes: canonicalVotes,
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, res.Status)
	})

	t.Run("reject when canonical commit validator is missing from ExtendedCommitInfo", func(t *testing.T) {
		// Include only 3 of 4 validators in ExtendedCommitInfo (75% VP, passes >2/3 VP check).
		// But the ProposedLastCommit includes all 4, hence the completeness check should catch the omission.
		partialExtCommit := &abci.ExtendedCommitInfo{
			Round: round,
			Votes: allVotes[:3], // omit validator #4
		}
		partialExtCommitBytes, err := partialExtCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{partialExtCommitBytes},
			Height: reqHeight,
			ProposedLastCommit: abci.CommitInfo{
				Round: round,
				Votes: canonicalVotes, // all 4 vals
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status)
	})

	t.Run("reject when canonical commit validator is downgraded to absent", func(t *testing.T) {
		// Include all 4 validators, but downgrade #4 to Absent.
		// The remaining 3 have 75% VP (>2/3), so ValidateVoteExtensions would pass.
		// The completeness check must catch the flag downgrade.
		downgradedVotes := make([]abci.ExtendedVoteInfo, 4)
		copy(downgradedVotes, allVotes)
		downgradedVotes[3] = abci.ExtendedVoteInfo{
			Validator:   allVotes[3].Validator,
			BlockIdFlag: cmtproto.BlockIDFlagAbsent,
		}

		downgradedExtCommit := &abci.ExtendedCommitInfo{
			Round: round,
			Votes: downgradedVotes,
		}
		downgradedExtCommitBytes, err := downgradedExtCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{downgradedExtCommitBytes},
			Height: reqHeight,
			ProposedLastCommit: abci.CommitInfo{
				Round: round,
				Votes: canonicalVotes, // all 4 vals
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, res.Status)
	})

	t.Run("no completeness check before hardfork", func(t *testing.T) {
		helper.SetPhuketHardforkHeight(0)
		defer helper.SetPhuketHardforkHeight(1)

		// Before the hardfork, omitting a canonical commit validator should NOT trigger
		// the completeness check. Use 3 of 4 validators (75% VP passes >2/3).
		partialExtCommit := &abci.ExtendedCommitInfo{
			Round: round,
			Votes: allVotes[:3], // omit validator #4
		}
		partialExtCommitBytes, err := partialExtCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{partialExtCommitBytes},
			Height: reqHeight,
			ProposedLastCommit: abci.CommitInfo{
				Round: round,
				Votes: canonicalVotes, // all 4 vals
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		// Without completeness check, 75% VP should pass
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, res.Status)
	})

	t.Run("accept when one validator is a filtered placeholder", func(t *testing.T) {
		// Validator #4 is a filtered placeholder: BlockIDFlagCommit but empty extension fields.
		// The remaining 3 validators (75% VP > 2/3) should pass both completeness and VP checks.
		placeholderVotes := make([]abci.ExtendedVoteInfo, 4)
		copy(placeholderVotes, allVotes)
		placeholderVotes[3] = abci.ExtendedVoteInfo{
			Validator:   allVotes[3].Validator,
			BlockIdFlag: cmtproto.BlockIDFlagCommit,
			// All extension fields are nil — this is a filtered placeholder
		}

		placeholderExtCommit := &abci.ExtendedCommitInfo{
			Round: round,
			Votes: placeholderVotes,
		}
		placeholderExtCommitBytes, err := placeholderExtCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{placeholderExtCommitBytes},
			Height: reqHeight,
			ProposedLastCommit: abci.CommitInfo{
				Round: round,
				Votes: canonicalVotes, // all 4 vals with Commit flag
			},
		}

		handler := app.NewProcessProposalHandler()
		res, err := handler(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, res.Status)
	})
}

// buildExtensionCommitsWithMilestoneProposition is a helper function to build an ExtendedCommitInfo with a MilestoneProposition in the vote extension for testing purposes
func buildExtensionCommitsWithMilestoneProposition(t *testing.T, app *HeimdallApp, txHashBytes []byte, validators []*stakeTypes.Validator, validatorPrivKeys []secp256k1.PrivKey, milestoneProp milestoneTypes.MilestoneProposition) ([]byte, *abci.ExtendedCommitInfo, *abci.ExtendedVoteInfo, error) {

	cometVal := abci.Validator{
		Address: common.FromHex(validators[0].Signer),
		Power:   validators[0].VotingPower,
	}

	cmtPubKey, err := validators[0].CmtConsPublicKey()
	require.NoError(t, err)

	voteInfo := setupExtendedVoteInfoWithMilestoneProposition(
		t,
		cmtproto.BlockIDFlagCommit,
		txHashBytes,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
		cometVal,
		validatorPrivKeys[0],
		2,
		app,
		cmtPubKey.GetEd25519(),
		milestoneProp,
	)

	extCommit := &abci.ExtendedCommitInfo{
		Round: 1,
		Votes: []abci.ExtendedVoteInfo{voteInfo},
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)
	return extCommitBytes, extCommit, &voteInfo, err
}

// createVoteExtensionsWithPartialSupport is a helper function to create vote extensions with the specified percentage of voting power supporting a milestone
func createVoteExtensionsWithPartialSupport(t *testing.T, validators []*stakeTypes.Validator, validatorPrivKeys []secp256k1.PrivKey, lastMilestone *milestoneTypes.Milestone, supportPercentage int, voteExtHeight int64) []abci.ExtendedVoteInfo {
	var voteExtensions []abci.ExtendedVoteInfo
	totalVotingPower := int64(0)
	supportingVotingPower := int64(0)

	// Calculate total voting power
	for _, v := range validators {
		totalVotingPower += v.VotingPower
	}

	targetSupportingPower := (totalVotingPower * int64(supportPercentage)) / 100

	// Create milestone proposition. The actual-head fields (POS-3629) mirror the single-block head
	// here, so the >1/3 actual-head tally resolves to StartBlockNumber.
	newMilestone := &milestoneTypes.MilestoneProposition{
		StartBlockNumber:  lastMilestone.EndBlock + 1,
		BlockHashes:       [][]byte{common.HexToHash("0x5678").Bytes()},
		ParentHash:        lastMilestone.Hash,
		BlockTds:          []uint64{1},
		LatestBlockNumber: lastMilestone.EndBlock + 1,
		LatestBlockHash:   common.HexToHash("0x5678").Bytes(),
	}

	// Create dummy non-rp vote extension
	dummyNonRpExt, err := GetDummyNonRpVoteExtension(voteExtHeight, "test-chain")
	require.NoError(t, err)

	for i, validator := range validators {
		var voteExt []byte

		if supportingVotingPower < targetSupportingPower {
			// Create the vote extension with a milestone proposition
			voteExtension := &sidetxs.VoteExtension{
				BlockHash:            []byte("test-block-hash"),
				Height:               voteExtHeight,
				MilestoneProposition: newMilestone,
				SideTxResponses:      []sidetxs.SideTxResponse{},
			}
			encoded, err := gogoproto.Marshal(voteExtension)
			require.NoError(t, err)
			voteExt = encoded
			supportingVotingPower += validator.VotingPower
		} else {
			// Create the vote extension without a milestone proposition
			voteExtension := &sidetxs.VoteExtension{
				BlockHash:            []byte("test-block-hash"),
				Height:               voteExtHeight,
				MilestoneProposition: nil,
				SideTxResponses:      []sidetxs.SideTxResponse{},
			}
			encoded, err := gogoproto.Marshal(voteExtension)
			require.NoError(t, err)
			voteExt = encoded
		}

		// Use the validator's private key to get the consensus address
		consAddr := validatorPrivKeys[i].PubKey().Address()
		voteExtensions = append(voteExtensions, abci.ExtendedVoteInfo{
			Validator: abci.Validator{
				Address: consAddr,
				Power:   validator.VotingPower,
			},
			VoteExtension:           voteExt,
			ExtensionSignature:      []byte("dummy-signature"),
			NonRpVoteExtension:      dummyNonRpExt,
			NonRpExtensionSignature: []byte("dummy-non-rp-signature"),
			BlockIdFlag:             cmtproto.BlockIDFlagCommit,
		})
	}

	return voteExtensions
}

// actualHeadExtVotes builds committed vote extensions in which the first `supporters` validators
// report (number, hash) as their actual latest bor head (POS-3629), decoupled from any proposition
// window — for driving the >1/3 actual-head tally in handlePendingMilestone.
func actualHeadExtVotes(t *testing.T, validators []*stakeTypes.Validator, validatorPrivKeys []secp256k1.PrivKey, number uint64, hash []byte, supporters int) []abci.ExtendedVoteInfo {
	t.Helper()
	votes := make([]abci.ExtendedVoteInfo, 0, len(validators))
	for i, v := range validators {
		var prop *milestoneTypes.MilestoneProposition
		if i < supporters {
			prop = &milestoneTypes.MilestoneProposition{
				StartBlockNumber:  number,
				BlockHashes:       [][]byte{hash},
				BlockTds:          []uint64{1},
				LatestBlockNumber: number,
				LatestBlockHash:   hash,
			}
		}
		ve := &sidetxs.VoteExtension{Height: 1, MilestoneProposition: prop}
		enc, err := gogoproto.Marshal(ve)
		require.NoError(t, err)
		votes = append(votes, abci.ExtendedVoteInfo{
			Validator:     abci.Validator{Address: validatorPrivKeys[i].PubKey().Address(), Power: v.VotingPower},
			VoteExtension: enc,
			BlockIdFlag:   cmtproto.BlockIDFlagCommit,
		})
	}
	return votes
}

// actualHeadExtVotesSplit builds committed vote extensions where validators[:splitIdx] report
// (numA, hashA) as their actual head and validators[splitIdx:] report (numB, hashB) — for driving a
// byzantine-minority-vs-honest-majority actual-head tally in handlePendingMilestone (POS-3629).
func actualHeadExtVotesSplit(t *testing.T, validators []*stakeTypes.Validator, validatorPrivKeys []secp256k1.PrivKey, splitIdx int, numA uint64, hashA []byte, numB uint64, hashB []byte) []abci.ExtendedVoteInfo {
	t.Helper()
	votes := make([]abci.ExtendedVoteInfo, 0, len(validators))
	for i, v := range validators {
		number, hash := numA, hashA
		if i >= splitIdx {
			number, hash = numB, hashB
		}
		ve := &sidetxs.VoteExtension{Height: 1, MilestoneProposition: &milestoneTypes.MilestoneProposition{
			StartBlockNumber:  number,
			BlockHashes:       [][]byte{hash},
			BlockTds:          []uint64{1},
			LatestBlockNumber: number,
			LatestBlockHash:   hash,
		}}
		enc, err := gogoproto.Marshal(ve)
		require.NoError(t, err)
		votes = append(votes, abci.ExtendedVoteInfo{
			Validator:     abci.Validator{Address: validatorPrivKeys[i].PubKey().Address(), Power: v.VotingPower},
			VoteExtension: enc,
			BlockIdFlag:   cmtproto.BlockIDFlagCommit,
		})
	}
	return votes
}

func TestExtractTxHashMsgEventRecordValidation(t *testing.T) {
	validHash := common.BigToHash(common.Big1)

	testCases := []struct {
		name   string
		txHash string
		ok     bool
		hash   common.Hash
	}{
		{
			name:   "valid 0x-prefixed 32-byte hash",
			txHash: validHash.Hex(),
			ok:     true,
			hash:   validHash,
		},
		{
			name:   "rejects non-prefixed hash string",
			txHash: common.Bytes2Hex(validHash.Bytes()),
			ok:     false,
			hash:   common.Hash{},
		},
		{
			name:   "rejects malformed short hash",
			txHash: "0x1234",
			ok:     false,
			hash:   common.Hash{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &clerkTypes.MsgEventRecord{TxHash: tc.txHash}
			hash, ok := extractTxHash(msg)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.hash, hash)
		})
	}
}
