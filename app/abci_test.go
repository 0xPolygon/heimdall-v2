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
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
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

func TestPrepareProposalHandler(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
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
		&app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators,
		validatorPrivKeys,
		app.LastBlockHeight(),
	)
	require.NoError(t, err)

	// Prepare/Process proposal
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
		BorChainId:      "1",
	}

	txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2)

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

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2)

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

	extCommitBytes, extCommit, voteInfo, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2)

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

	// padding
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

	extCommitBytes, _, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)

	app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes})

	finalizeReq := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytes, txBytes},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
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

	stateSyncEvent := &statesender.StatesenderStateSynced{
		Id:              new(big.Int).SetUint64(1),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Data:            []byte("test-data"),
	}

	event := &stakinginfo.StakinginfoTopUpFee{
		User: common.Address(sdk.AccAddress(addr2.String())),
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
	mockBorKeeper := borKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		authTypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ChainManagerKeeper,
		&app.StakeKeeper,
		nil,
		nil,
	)

	mockClerkKeeper := clerkKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		app.ChainManagerKeeper,
		mockCaller,
	)
	app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[borTypes.ModuleName] = bor.NewAppModule(mockBorKeeper, mockCaller)
	app.BorKeeper.SetContractCaller(mockCaller)

	app.ModuleManager.Modules[clerkTypes.ModuleName] = clerk.NewAppModule(mockClerkKeeper)
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

	// coins, _ := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})

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
				rec := clerkTypes.NewMsgEventRecord(
					validators[0].Signer,
					TxHash1,
					1,
					50,
					1,
					propAddr,
					make([]byte, 0),
					"0",
				)
				return &rec
			}(),
		},
		// {
		// 	name: "topup [MsgProposeSpan]] happy path",
		// 	msg: func() *topUpTypes.MsgTopupTx {
		// 		rec := topUpTypes.NewMsgTopupTx(
		// 			validators[0].Signer,
		// 			validators[0].Signer,
		// 			coins.AmountOf(authTypes.FeeToken),
		// 			[]byte(TxHash1),
		// 			1,
		// 			1,
		// 		)
		// 		return rec
		// 	}(),
		// },
	}

	mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)

	mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(stateSyncEvent, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			txBytes, err := buildSignedTx(tc.msg, validators[0].Signer, ctx, priv, app)
			var txBytesCmt cmtTypes.Tx = txBytes

			extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

			app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes})

			extCommitBytes2, _, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)

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
	helper.SetTestInitialHeight(3)
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
	// app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[borTypes.ModuleName] = bor.NewAppModule(mockBorKeeper, mockCaller)
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

	mockCaller.On("GetBorChainBlockAuthor", mock.Anything).Return(&val1Addr, nil)

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

		txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		txBytes, err := buildSignedTx(msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

} // completed

func TestAllUnhappyPathClerkSideTxs(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
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

	mockClerkKeeper := clerkKeeper.NewKeeper(
		app.AppCodec(),
		runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)),
		mockChainKeeper,
		mockCaller,
	)

	app.BorKeeper.SetContractCaller(mockCaller)
	// app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[clerkTypes.ModuleName] = clerk.NewAppModule(mockClerkKeeper)
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

	t.Run("no reciept", func(t *testing.T) {

		logIndex := uint64(200)
		blockNumber := uint64(51)

		ac := address.NewHexCodec()
		Address2 := "0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff"

		addrBz2, err := ac.StringToBytes(Address2)
		msg := clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			logIndex,
			blockNumber,
			10,
			addrBz2,
			make([]byte, 0),
			"101",
		)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(1),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
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

		msg := clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			logIndex,
			blockNumber,
			10,
			addrBz2,
			make([]byte, 0),
			"0",
		)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()

		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
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

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		event := &statesender.StatesenderStateSynced{
			Id:              new(big.Int).SetUint64(msg.Id),
			ContractAddress: common.BytesToAddress([]byte(msg.ContractAddress)),
			Data:            b,
		}
		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil).Once()

		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
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

		msg := clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			make([]byte, 0),
			"0",
		)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Once()

		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		//clerkKeeper.Keeper.ChainKeeper.(*clerktestutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

		finalizeReq := abci.RequestFinalizeBlock{
			Txs:             [][]byte{extCommitBytes, txBytes},
			Height:          3,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}
		_, err = app.PreBlocker(ctx, &finalizeReq)
		require.NoError(t, err)

	})

} // Completed

func TestAllUnhappyPathTopupSideTxs(t *testing.T) {

	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
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
		mockCaller,
	)
	mockTopupKeeper := app.TopupKeeper
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
	// app.BorKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.MilestoneKeeper.IContractCaller = mockCaller
	app.caller = mockCaller

	app.ModuleManager.Modules[topUpTypes.ModuleName] = topup.NewAppModule(mockTopupKeeper, mockCaller)
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

	t.Run("no reciept", func(t *testing.T) {

		logIndex := uint64(10)
		blockNumber := uint64(599)
		hash := []byte(TxHash1)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})

		msg := *topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockCaller.
			On("GetBorChainBlock", mock.Anything, mock.Anything).
			Return(&ethTypes.Header{
				Number: big.NewInt(10),
			}, nil)
		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
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

		msg := *topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()

		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
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

		msg := *topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)
		event := &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr1.String())),
			Fee:  coins.AmountOf(authTypes.FeeToken).BigInt(),
		}

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil).Once()

		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
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

		msg := *topUpTypes.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)
		event := &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr2.String())),
			Fee:  coins.AmountOf(authTypes.FeeToken).BigInt(),
		}
		fmt.Println("habaka", txReceipt)

		mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
		mockCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)

		mockCaller.
			On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
			Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		// mockChainKeeper.EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).AnyTimes()

		txBytes, err := buildSignedTx(&msg, validators[0].Signer, ctx, priv, app)
		var txBytesCmt cmtTypes.Tx = txBytes

		extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, txBytesCmt.Hash(), validators, validatorPrivKeys, 2)
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

		ve.Unmarshal(respExtend.VoteExtension)
		require.Equal(t, ve.SideTxResponses[0].Result, sidetxs.Vote_VOTE_NO, "expected at least one vote == VOTE_NO in the results")

	})

} // completed

func TestMilestoneHappyPath(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{
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

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2)

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

	app.MilestoneKeeper.AddMilestone(ctx, testMilestone1)

	reqExtend := abci.RequestExtendVote{
		Txs:    respPrep.Txs,
		Hash:   []byte("test-hash"),
		Height: 3,
	}
	respExtend, err := app.ExtendVoteHandler()(ctx, &reqExtend)
	require.NoError(t, err)
	require.NotNil(t, respExtend.VoteExtension)

	var ve sidetxs.VoteExtension
	ve.Unmarshal(respExtend.VoteExtension)

	extCommitBytesWithMilestone, _, _, err := buildExtensionCommitsWithMilestoneProposition(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, *ve.MilestoneProposition)

	finalizeReq := abci.RequestFinalizeBlock{
		Txs:             [][]byte{extCommitBytesWithMilestone, txBytes},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	_, err = app.PreBlocker(ctx, &finalizeReq)

}

func TestMilestoneUnhappyPaths(t *testing.T) {
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

	extCommitBytes, extCommit, _, err := buildExtensionCommits(t, &app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2)

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

		app.MilestoneKeeper.AddMilestone(ctx, testMilestone1)

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
	mockCaller.AssertCalled(t, "GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64"))

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

	// VerifyVoteExtension — **here's the fix: pass the consensus address** 🎉
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
		Txs:             [][]byte{extCommitBytes, txBytes},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
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
		Txs:             [][]byte{extCommitBytes2, txBytesBor},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
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
		Txs:             [][]byte{extCommitBytes3, txBytesClerk},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
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
		Txs:             [][]byte{extCommitBytes4, txBytesTopUp},
		Height:          3,
		ProposerAddress: common.FromHex(validators[0].Signer),
	}
	_, err = app.PreBlocker(ctx, &finalizeReqTopUpSidetx)
	require.NoError(t, err)

	//---------------------------------------------------------------------------
	//--------------------happy path for

}

var defaultFeeAmount = big.NewInt(10).Exp(big.NewInt(10), big.NewInt(15), nil).Int64()

func TestUpdateBlockProducerStatus(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCIctx(t)

	// Setup initial state for latest active and failed producers
	initialActiveProducers := map[uint64]struct{}{1: {}, 2: {}}
	err := app.BorKeeper.UpdateLatestActiveProducer(ctx, initialActiveProducers)
	require.NoError(t, err)

	err = app.BorKeeper.AddLatestFailedProducer(ctx, 3)
	require.NoError(t, err)
	err = app.BorKeeper.AddLatestFailedProducer(ctx, 4)
	require.NoError(t, err)

	// The supporting producers for the new block
	supportingProducerIDs := map[uint64]struct{}{5: {}, 6: {}}

	// Call the function
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

// TestUpdateBlockProducerStatus_ErrorCases tests error scenarios for updateBlockProducerStatus
func TestUpdateBlockProducerStatus_ErrorCases(t *testing.T) {
	t.Run("empty_supporting_producers", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// Empty supporting producers map
		supportingProducerIDs := map[uint64]struct{}{}

		err := app.updateBlockProducerStatus(ctx, supportingProducerIDs)
		require.NoError(t, err)

		// Verify the state is updated correctly
		latestActive, err := app.BorKeeper.GetLatestActiveProducer(ctx)
		require.NoError(t, err)
		require.Empty(t, latestActive)

		latestFailed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
		require.NoError(t, err)
		require.Empty(t, latestFailed)
	})

	t.Run("large_producer_set", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// Large set of supporting producers
		supportingProducerIDs := make(map[uint64]struct{})
		for i := uint64(1); i <= 100; i++ {
			supportingProducerIDs[i] = struct{}{}
		}

		err := app.updateBlockProducerStatus(ctx, supportingProducerIDs)
		require.NoError(t, err)

		latestActive, err := app.BorKeeper.GetLatestActiveProducer(ctx)
		require.NoError(t, err)
		require.Equal(t, len(supportingProducerIDs), len(latestActive))
	})
}

// TestGetValidatorSetForHeight_EdgeCases tests edge cases for getValidatorSetForHeight
func TestGetValidatorSetForHeight_EdgeCases(t *testing.T) {
	t.Run("height_at_initial", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// Test at initial height
		initialHeight := helper.GetInitialHeight()
		valSet, err := app.getValidatorSetForHeight(ctx, initialHeight)
		require.NoError(t, err)
		require.NotNil(t, valSet)
	})

	t.Run("height_before_tally_fix", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// Test before tally fix height
		height := helper.GetTallyFixHeight() - 1
		valSet, err := app.getValidatorSetForHeight(ctx, height)
		require.NoError(t, err)
		require.NotNil(t, valSet)
	})

	t.Run("height_after_tally_fix", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctxAndValidators(t, 3)

		// Add some validator history for testing
		// Note: This might need actual validator setup depending on the keeper implementation
		height := helper.GetTallyFixHeight() + 10
		valSet, err := app.getValidatorSetForHeight(ctx, height)
		require.NoError(t, err)
		require.NotNil(t, valSet)
	})
}

// TestRejectUnknownVoteExtFields_Comprehensive tests rejectUnknownVoteExtFields thoroughly
func TestRejectUnknownVoteExtFields_Comprehensive(t *testing.T) {
	t.Run("valid_vote_extension", func(t *testing.T) {
		voteExt := sidetxs.VoteExtension{
			Height:               100,
			BlockHash:            []byte("test_hash"),
			SideTxResponses:      []sidetxs.SideTxResponse{},
			MilestoneProposition: nil,
		}
		bz, err := voteExt.Marshal()
		require.NoError(t, err)

		err = rejectUnknownVoteExtFields(bz)
		require.NoError(t, err)
	})

	t.Run("vote_extension_with_milestone", func(t *testing.T) {
		voteExt := sidetxs.VoteExtension{
			Height:    100,
			BlockHash: []byte("test_hash"),
			MilestoneProposition: &milestoneTypes.MilestoneProposition{
				StartBlockNumber: 1000,
				BlockHashes:      [][]byte{[]byte("hash1"), []byte("hash2")},
				BlockTds:         []uint64{100, 200},
			},
		}
		bz, err := voteExt.Marshal()
		require.NoError(t, err)

		err = rejectUnknownVoteExtFields(bz)
		require.NoError(t, err)
	})

	t.Run("empty_bytes", func(t *testing.T) {
		err := rejectUnknownVoteExtFields([]byte{})
		// Empty bytes might be valid as they represent an empty proto message
		require.NoError(t, err)
	})

	t.Run("invalid_bytes", func(t *testing.T) {
		invalidBytes := []byte{0xff, 0xff, 0xff, 0xff}
		err := rejectUnknownVoteExtFields(invalidBytes)
		require.Error(t, err)
	})
}

// TestCheckAndAddFutureSpan_EdgeCases tests edge cases for checkAndAddFutureSpan
func TestCheckAndAddFutureSpan_EdgeCases(t *testing.T) {
	t.Run("milestone_below_span_start", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// Setup last span
		lastSpan := borTypes.Span{
			Id:         1,
			StartBlock: 1000,
			EndBlock:   2000,
			BorChainId: "1",
		}

		// Milestone ends before span start - no new span should be created
		majorityMilestone := &milestoneTypes.MilestoneProposition{
			StartBlockNumber: 500,
			BlockHashes:      [][]byte{[]byte("hash1"), []byte("hash2")},
			BlockTds:         []uint64{100, 200},
		}

		supportingValidatorIDs := map[uint64]struct{}{1: {}, 2: {}}

		err := app.checkAndAddFutureSpan(ctx, majorityMilestone, lastSpan, supportingValidatorIDs)
		require.NoError(t, err)
	})

	t.Run("empty_supporting_validators", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		lastSpan := borTypes.Span{
			Id:         1,
			StartBlock: 256,
			EndBlock:   1024,
			BorChainId: "1",
		}

		majorityMilestone := &milestoneTypes.MilestoneProposition{
			StartBlockNumber: 1000,
			BlockHashes:      make([][]byte, 100),
			BlockTds:         make([]uint64, 100),
		}

		supportingValidatorIDs := map[uint64]struct{}{}

		err := app.checkAndAddFutureSpan(ctx, majorityMilestone, lastSpan, supportingValidatorIDs)
		// Should handle empty validator set gracefully
		require.Error(t, err) // Will likely error due to missing params or setup
	})
}

// TestCheckAndRotateCurrentSpan_EdgeCases tests edge cases for checkAndRotateCurrentSpan
func TestCheckAndRotateCurrentSpan_EdgeCases(t *testing.T) {
	t.Run("no_milestone_exists", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// No milestone in the system
		err := app.checkAndRotateCurrentSpan(ctx)
		// Should handle gracefully when no milestone exists - the function checks if milestone exists
		// and returns early without error if none exists
		require.NoError(t, err)
	})

	t.Run("recent_milestone", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// Add a recent milestone
		milestone := milestoneTypes.Milestone{
			Proposer:   common.HexToAddress("0x1").Hex(),
			StartBlock: 100,
			EndBlock:   200,
			Hash:       []byte("hash"),
			BorChainId: "1",
			Timestamp:  uint64(ctx.BlockTime().Unix()),
		}
		err := app.MilestoneKeeper.AddMilestone(ctx, milestone)
		require.NoError(t, err)

		// Set last milestone block to current block (recent)
		err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, uint64(ctx.BlockHeight()))
		require.NoError(t, err)

		// Should not rotate since milestone is recent
		err = app.checkAndRotateCurrentSpan(ctx)
		require.NoError(t, err)
	})
}

// buildExtensionCommitsWithMultipleValidators creates extension commits with all validators
func buildExtensionCommitsWithMultipleValidators(
	t *testing.T,
	app *HeimdallApp,
	txHashBytes []byte,
	validators []*stakeTypes.Validator,
	validatorPrivKeys []secp256k1.PrivKey,
	height int64,
) ([]byte, *abci.ExtendedCommitInfo, error) {
	t.Helper()

	extCommit := &abci.ExtendedCommitInfo{
		Round: 1,
		Votes: []abci.ExtendedVoteInfo{},
	}

	// Add votes from all validators
	for i, val := range validators {
		cometVal := abci.Validator{
			Address: common.FromHex(val.Signer),
			Power:   val.VotingPower,
		}

		cmtPubKey, err := val.CmtConsPublicKey()
		require.NoError(t, err)

		voteInfo := setupExtendedVoteInfoWithNonRp(
			t,
			cmtproto.BlockIDFlagCommit,
			txHashBytes,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000002dead"),
			cometVal,
			validatorPrivKeys[i],
			height,
			app,
			cmtPubKey.GetEd25519(),
		)

		extCommit.Votes = append(extCommit.Votes, voteInfo)
	}

	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)
	return extCommitBytes, extCommit, err
}

// TestPrepareProposalHandler_AdditionalEdgeCases tests additional edge cases for PrepareProposalHandler
func TestPrepareProposalHandler_AdditionalEdgeCases(t *testing.T) {
	t.Run("max_tx_bytes_exceeded_first_tx", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		handler := app.NewPrepareProposalHandler()

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Create a very large transaction
		msg := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        make([]byte, 10000), // Large payload
			AccountRootHash: make([]byte, 10000),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTx(msg, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		// Set MaxTxBytes very low
		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      100, // Very low limit
			Height:          3,   // VoteExtBlockHeight + 1
			LocalLastCommit: *extCommit,
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		// Should only include the vote extension, not the large tx
		require.Equal(t, 1, len(resp.Txs))
	})

	t.Run("tx_decode_error", func(t *testing.T) {
		_, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		handler := app.NewPrepareProposalHandler()

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Invalid transaction bytes
		invalidTxBytes := []byte{0xff, 0xff, 0xff}

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{invalidTxBytes},
			MaxTxBytes:      1000000,
			Height:          3, // VoteExtBlockHeight + 1
			LocalLastCommit: *extCommit,
		}

		_, err = handler(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error occurred while decoding tx")
	})
}

// TestProcessProposalHandler_AdditionalEdgeCases tests additional edge cases for ProcessProposalHandler
func TestProcessProposalHandler_AdditionalEdgeCases(t *testing.T) {
	t.Run("empty_txs", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.NewProcessProposalHandler()

		req := &abci.RequestProcessProposal{
			Txs:                [][]byte{},
			Height:             2,
			ProposedLastCommit: abci.CommitInfo{},
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
	})

	t.Run("invalid_extended_commit_info", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.NewProcessProposalHandler()

		// Invalid marshaled data
		invalidBytes := []byte{0xff, 0xff, 0xff}

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{invalidBytes},
			Height: 2,
			ProposedLastCommit: abci.CommitInfo{
				Round: 0,
			},
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
	})

	t.Run("round_mismatch", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		handler := app.NewProcessProposalHandler()

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			app.LastBlockHeight(),
		)
		require.NoError(t, err)

		// Change the round
		extCommit.Round = 5

		bz, err := extCommit.Marshal()
		require.NoError(t, err)

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

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{bz, txBytes},
			Height: 2,
			ProposedLastCommit: abci.CommitInfo{
				Round: 0, // Different from extCommit.Round
			},
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
	})

	t.Run("tx_decode_error_in_process", func(t *testing.T) {
		_, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		handler := app.NewProcessProposalHandler()

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			app.LastBlockHeight(),
		)
		require.NoError(t, err)

		bz, err := extCommit.Marshal()
		require.NoError(t, err)

		// Add invalid tx bytes
		invalidTxBytes := []byte{0xff, 0xff, 0xff}

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{bz, invalidTxBytes},
			Height: 2,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
			},
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
	})
}

// TestExtendVoteHandler_AdditionalEdgeCases tests additional edge cases for ExtendVoteHandler
func TestExtendVoteHandler_AdditionalEdgeCases(t *testing.T) {
	t.Run("vote_extensions_disabled", func(t *testing.T) {
		priv, app, ctx, _ := SetupAppWithABCIctx(t)

		// Set consensus params to disable vote extensions
		params := cmtproto.ConsensusParams{
			Abci: &cmtproto.ABCIParams{
				VoteExtensionsEnableHeight: 1000, // Far in the future
			},
		}
		ctx = ctx.WithConsensusParams(params)

		handler := app.ExtendVoteHandler()

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

		extCommit := abci.ExtendedCommitInfo{
			Round: 0,
			Votes: []abci.ExtendedVoteInfo{},
		}
		extCommitBz, err := extCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestExtendVote{
			Height: 2,
			Txs:    [][]byte{extCommitBz, txBytes},
			Hash:   common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		}

		_, err = handler(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vote extensions are disabled")
	})

	t.Run("no_txs", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.ExtendVoteHandler()

		req := &abci.RequestExtendVote{
			Height: 2,
			Txs:    [][]byte{},
			Hash:   common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		}

		// This will panic or error due to accessing Txs[0]
		require.Panics(t, func() {
			_, _ = handler(ctx, req)
		})
	})

	t.Run("invalid_extended_commit_unmarshal", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.ExtendVoteHandler()

		// Invalid bytes that can't be unmarshaled
		invalidBytes := []byte{0xff, 0xff, 0xff}

		req := &abci.RequestExtendVote{
			Height: 2,
			Txs:    [][]byte{invalidBytes},
			Hash:   common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		}

		_, err := handler(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error occurred while decoding ExtendedCommitInfo")
	})
}

// TestVerifyVoteExtensionHandler_AdditionalEdgeCases tests additional edge cases for VerifyVoteExtensionHandler
func TestVerifyVoteExtensionHandler_AdditionalEdgeCases(t *testing.T) {
	t.Run("height_mismatch", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.VerifyVoteExtensionHandler()

		voteExt := sidetxs.VoteExtension{
			Height:          999, // Wrong height
			BlockHash:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			SideTxResponses: []sidetxs.SideTxResponse{},
		}

		bz, err := voteExt.Marshal()
		require.NoError(t, err)

		valKey := secp256k1.GenPrivKey()

		req := &abci.RequestVerifyVoteExtension{
			Height:           2, // Different from voteExt.Height
			VoteExtension:    bz,
			Hash:             common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			ValidatorAddress: valKey.PubKey().Address(),
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, resp.Status)
	})

	t.Run("block_hash_mismatch", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.VerifyVoteExtensionHandler()

		voteExt := sidetxs.VoteExtension{
			Height:          2,
			BlockHash:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000ffff"), // Wrong hash
			SideTxResponses: []sidetxs.SideTxResponse{},
		}

		bz, err := voteExt.Marshal()
		require.NoError(t, err)

		valKey := secp256k1.GenPrivKey()

		req := &abci.RequestVerifyVoteExtension{
			Height:           2,
			VoteExtension:    bz,
			Hash:             common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), // Different hash
			ValidatorAddress: valKey.PubKey().Address(),
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, resp.Status)
	})

	t.Run("duplicate_votes_in_side_tx_responses", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.VerifyVoteExtensionHandler()

		// Create duplicate tx hashes
		txHash := []byte("duplicate_hash")
		voteExt := sidetxs.VoteExtension{
			Height:    2,
			BlockHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			SideTxResponses: []sidetxs.SideTxResponse{
				{
					TxHash: txHash,
					Result: sidetxs.Vote_VOTE_YES,
				},
				{
					TxHash: txHash, // Duplicate
					Result: sidetxs.Vote_VOTE_NO,
				},
			},
		}

		bz, err := voteExt.Marshal()
		require.NoError(t, err)

		valKey := secp256k1.GenPrivKey()

		req := &abci.RequestVerifyVoteExtension{
			Height:           2,
			VoteExtension:    bz,
			Hash:             common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			ValidatorAddress: valKey.PubKey().Address(),
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, resp.Status)
	})

	t.Run("unmarshal_error", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		handler := app.VerifyVoteExtensionHandler()

		valKey := secp256k1.GenPrivKey()

		req := &abci.RequestVerifyVoteExtension{
			Height:           2,
			VoteExtension:    []byte{0xff, 0xff, 0xff}, // Invalid bytes
			Hash:             common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			ValidatorAddress: valKey.PubKey().Address(),
		}

		resp, err := handler(ctx, req)
		require.NoError(t, err)
		require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, resp.Status)
	})
}

// TestPreBlocker_AdditionalEdgeCases tests additional edge cases for PreBlocker
func TestPreBlocker_AdditionalEdgeCases(t *testing.T) {
	t.Run("empty_txs", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		req := &abci.RequestFinalizeBlock{
			Height: 2,
			Txs:    [][]byte{},
		}

		_, err := app.PreBlocker(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no txs found")
	})

	t.Run("invalid_extended_commit_unmarshal", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		req := &abci.RequestFinalizeBlock{
			Height: 2,
			Txs:    [][]byte{[]byte{0xff, 0xff, 0xff}},
		}

		_, err := app.PreBlocker(ctx, req)
		require.Error(t, err)
	})

	t.Run("vote_extensions_disabled_at_next_height", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctx(t)

		// Set vote extensions to be disabled at next height
		params := cmtproto.ConsensusParams{
			Abci: &cmtproto.ABCIParams{
				VoteExtensionsEnableHeight: 1000,
			},
		}
		ctx = ctx.WithConsensusParams(params)

		extCommit := abci.ExtendedCommitInfo{
			Round: 0,
			Votes: []abci.ExtendedVoteInfo{},
		}
		bz, err := extCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestFinalizeBlock{
			Height: 2,
			Txs:    [][]byte{bz},
		}

		_, err = app.PreBlocker(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vote extensions are disabled")
	})

	t.Run("non_empty_ves_at_initial_height", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctxAndValidators(t, 2)

		// Set height to enable height (initial)
		params := cmtproto.ConsensusParams{
			Abci: &cmtproto.ABCIParams{
				VoteExtensionsEnableHeight: 1,
			},
		}
		ctx = ctx.WithConsensusParams(params).WithBlockHeight(1)

		// Create extended commit with votes (should be empty at initial height)
		extCommit := abci.ExtendedCommitInfo{
			Round: 0,
			Votes: []abci.ExtendedVoteInfo{
				{
					Validator: abci.Validator{
						Address: []byte("validator1"),
						Power:   100,
					},
					VoteExtension: []byte("some_extension"),
				},
			},
		}
		bz, err := extCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestFinalizeBlock{
			Height: 1,
			Txs:    [][]byte{bz},
		}

		_, err = app.PreBlocker(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "non-empty VEs found in the initial height")
	})
}

// TestMultipleSideTxHandlers comprehensively tests PrepareProposal and ProcessProposal
// behavior when transactions contain multiple side transaction handlers.
func TestMultipleSideTxHandlers(t *testing.T) {
	t.Run("single_checkpoint_sidetx", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Single checkpoint message
		msg := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{msg}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1000000,
			Height:          CurrentHeight,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		// Call PrepareProposal which invokes our custom NewPrepareProposalHandler
		// (registered via app.SetPrepareProposal in NewHeimdallApp)
		resp, err := app.PrepareProposal(req)
		require.NoError(t, err)
		// Should include vote extension + the valid single side tx (2 txs total)
		require.Equal(t, 2, len(resp.Txs), "Expected vote extension + 1 valid checkpoint tx")
	})

	t.Run("two_checkpoint_sidetxs_same_type_filtered", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Two checkpoint messages (same type)
		msg1 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
			BorChainId:      "1",
		}
		msg2 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      201,
			EndBlock:        300,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000beef"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003beef"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{msg1, msg2}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1000000,
			Height:          CurrentHeight,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		// Call PrepareProposal which invokes our custom NewPrepareProposalHandler
		// (registered via app.SetPrepareProposal in NewHeimdallApp)
		resp, err := app.PrepareProposal(req)
		require.NoError(t, err)
		// Should only include vote extension (1 tx), filtering out the tx with multiple side handlers
		require.Equal(t, 1, len(resp.Txs), "Expected only vote extension, tx with 2 checkpoints should be filtered")
	})

	t.Run("checkpoint_and_eventrecord_different_types_filtered", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Checkpoint message
		msg1 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
			BorChainId:      "1",
		}

		// Event record message (different type)
		contractAddr, _ := sdk.AccAddressFromHex("0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff")
		msg2 := clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			200,
			51,
			10,
			contractAddr,
			make([]byte, 0),
			"101",
		)

		txBytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{msg1, &msg2}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1000000,
			Height:          CurrentHeight,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		// Call PrepareProposal which invokes our custom NewPrepareProposalHandler
		// (registered via app.SetPrepareProposal in NewHeimdallApp)
		resp, err := app.PrepareProposal(req)
		require.NoError(t, err)
		// Should only include vote extension (1 tx), filtering out the tx with multiple side handlers
		require.Equal(t, 1, len(resp.Txs), "Expected only vote extension, tx with 2 checkpoints should be filtered")
	})

	t.Run("checkpoint_and_topup_different_types_filtered", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Checkpoint message
		msg1 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
			BorChainId:      "1",
		}

		// Topup message (different type)
		msg2 := &topUpTypes.MsgTopupTx{
			Proposer:    priv.PubKey().Address().String(),
			User:        "user123",
			Fee:         math.NewInt(1000000),
			TxHash:      common.Hex2Bytes(TxHash1),
			LogIndex:    100,
			BlockNumber: 50,
		}

		txBytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{msg1, msg2}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1000000,
			Height:          CurrentHeight,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		// Call PrepareProposal which invokes our custom NewPrepareProposalHandler
		// (registered via app.SetPrepareProposal in NewHeimdallApp)
		resp, err := app.PrepareProposal(req)
		require.NoError(t, err)
		// Should only include vote extension (1 tx), filtering out the tx with multiple side handlers
		require.Equal(t, 1, len(resp.Txs), "Expected only vote extension, tx with 2 checkpoints should be filtered")
	})

	t.Run("two_eventrecord_sidetxs_same_type_filtered", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Two event record messages (same type)
		contractAddr, _ := sdk.AccAddressFromHex("0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff")
		msg1 := clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash1,
			200,
			51,
			10,
			contractAddr,
			make([]byte, 0),
			"101",
		)
		msg2 := clerkTypes.NewMsgEventRecord(
			addressUtils.FormatAddress("0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"),
			TxHash2,
			201,
			52,
			11,
			contractAddr,
			make([]byte, 0),
			"101",
		)

		txBytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{&msg1, &msg2}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1000000,
			Height:          CurrentHeight,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		// Call PrepareProposal which invokes our custom NewPrepareProposalHandler
		// (registered via app.SetPrepareProposal in NewHeimdallApp)
		resp, err := app.PrepareProposal(req)
		require.NoError(t, err)
		// Should only include vote extension (1 tx), filtering out the tx with multiple side handlers
		require.Equal(t, 1, len(resp.Txs), "Expected only vote extension, tx with 2 checkpoints should be filtered")
	})

	t.Run("multiple_separate_txs_each_with_one_sidetx", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Create two separate transactions, each with a single checkpoint
		msg1 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
			BorChainId:      "1",
		}
		msg2 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      201,
			EndBlock:        300,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000beef"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003beef"),
			BorChainId:      "1",
		}

		// Build two separate transactions
		tx1Bytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{msg1}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		// Increment the account sequence for the second tx
		propAddr := sdk.AccAddress(priv.PubKey().Address())
		propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
		require.NoError(t, propAcc.SetSequence(propAcc.GetSequence()+1))
		app.AccountKeeper.SetAccount(ctx, propAcc)

		tx2Bytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{msg2}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		req := &abci.RequestPrepareProposal{
			Txs:             [][]byte{tx1Bytes, tx2Bytes},
			MaxTxBytes:      1000000,
			Height:          CurrentHeight,
			LocalLastCommit: *extCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
		}

		// Call PrepareProposal which invokes our custom NewPrepareProposalHandler
		// (registered via app.SetPrepareProposal in NewHeimdallApp)
		resp, err := app.PrepareProposal(req)
		require.NoError(t, err)
		// Should include vote extension + both valid checkpoint txs (3 txs total)
		// This proves blocks CAN contain multiple txs with different side handlers
		require.Equal(t, 3, len(resp.Txs), "Expected vote extension + 2 separate checkpoint txs")
	})

	t.Run("process_proposal_rejects_multiple_sidetxs", func(t *testing.T) {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCIctx(t)
		validators := app.StakeKeeper.GetAllValidators(ctx)

		_, extCommit, _, err := buildExtensionCommits(
			t,
			&app,
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
			validators,
			validatorPrivKeys,
			VoteExtBlockHeight,
		)
		require.NoError(t, err)

		// Two checkpoint messages
		msg1 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      100,
			EndBlock:        200,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
			BorChainId:      "1",
		}
		msg2 := &types.MsgCheckpoint{
			Proposer:        priv.PubKey().Address().String(),
			StartBlock:      201,
			EndBlock:        300,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000beef"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003beef"),
			BorChainId:      "1",
		}

		txBytes, err := buildSignedTxWithMultipleMsgs(t, []sdk.Msg{msg1, msg2}, priv.PubKey().Address().String(), ctx, priv, app)
		require.NoError(t, err)

		extCommitBz, err := extCommit.Marshal()
		require.NoError(t, err)

		req := &abci.RequestProcessProposal{
			Txs:    [][]byte{extCommitBz, txBytes},
			Height: CurrentHeight,
			ProposedLastCommit: abci.CommitInfo{
				Round: extCommit.Round,
			},
		}

		// Call ProcessProposal which invokes our custom NewProcessProposalHandler
		// (registered via app.SetProcessProposal in NewHeimdallApp)
		resp, err := app.ProcessProposal(req)
		require.NoError(t, err)
		// Should reject proposal with multiple side handlers in one tx
		require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status, "ProcessProposal should reject tx with multiple side handlers")
	})
}

func SetupAppWithABCIctx(t *testing.T) (cryptotypes.PrivKey, HeimdallApp, sdk.Context, []secp256k1.PrivKey) {
	return SetupAppWithABCIctxAndValidators(t, 1)
}

func SetupAppWithABCIctxAndValidators(t *testing.T, numValidators int) (cryptotypes.PrivKey, HeimdallApp, sdk.Context, []secp256k1.PrivKey) {
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
	return priv, *app, ctx, validatorPrivKeys
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

func buildSignedTxWithSequence(msg sdk.Msg, ctx sdk.Context, priv cryptotypes.PrivKey, app HeimdallApp, sequence uint64) ([]byte, error) {
	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	if propAcc == nil {
		propAcc = authTypes.NewBaseAccount(propAddr, priv.PubKey(), 1, 0)
		app.AccountKeeper.SetAccount(ctx, propAcc)
	} else if propAcc.GetPubKey() == nil {
		// Some genesis accounts (e.g. created from raw addresses) may not have a pubkey yet.
		propAcc.SetPubKey(priv.PubKey())
		app.AccountKeeper.SetAccount(ctx, propAcc)
	}

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

func buildSignedTx(msg sdk.Msg, signer string, ctx sdk.Context, priv cryptotypes.PrivKey, app HeimdallApp) ([]byte, error) {
	_ = signer // signer is kept for backwards compatibility; the tx signer is derived from priv.
	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	var sequence uint64
	if propAcc != nil {
		sequence = propAcc.GetSequence()
	}
	return buildSignedTxWithSequence(msg, ctx, priv, app, sequence)
}

func buildExtensionCommits(
	t *testing.T,
	app *HeimdallApp,
	txHashBytes []byte,
	validators []*stakeTypes.Validator,
	validatorPrivKeys []secp256k1.PrivKey,
	height int64,
) ([]byte, *abci.ExtendedCommitInfo, *abci.ExtendedVoteInfo, error) {

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
		height,
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

func buildExtensionCommitsWithMilestoneProposition(t *testing.T, app *HeimdallApp, txHashBytes []byte, validators []*stakeTypes.Validator, validatorPrivKeys []secp256k1.PrivKey, milestoneProp milestoneTypes.MilestoneProposition) ([]byte, *abci.ExtendedCommitInfo, *abci.ExtendedVoteInfo, error) {

	cometVal := abci.Validator{
		Address: common.FromHex(validators[0].Signer),
		Power:   validators[0].VotingPower,
	}

	cmtPubKey, err := validators[0].CmtConsPublicKey()

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

// buildSignedTxWithMultipleMsgs builds a transaction with multiple messages
func buildSignedTxWithMultipleMsgs(
	t *testing.T,
	msgs []sdk.Msg,
	signer string,
	ctx sdk.Context,
	priv cryptotypes.PrivKey,
	app HeimdallApp,
) ([]byte, error) {
	t.Helper()

	propAddr := sdk.AccAddress(priv.PubKey().Address())
	propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
	var sequence uint64
	if propAcc == nil {
		propAcc = authTypes.NewBaseAccount(propAddr, priv.PubKey(), 1, 0)
		app.AccountKeeper.SetAccount(ctx, propAcc)
	} else if propAcc.GetPubKey() == nil {
		propAcc.SetPubKey(priv.PubKey())
		app.AccountKeeper.SetAccount(ctx, propAcc)
	}
	sequence = propAcc.GetSequence()

	// Fund the account
	testutil.FundAccount(
		ctx,
		app.BankKeeper,
		propAddr,
		sdk.NewCoins(sdk.NewInt64Coin("pol", 43*defaultFeeAmount)),
	)

	// Build the tx with multiple messages
	txConfig := authtx.NewTxConfig(app.AppCodec(), authtx.DefaultSignModes)
	defaultSignMode, err := authsigning.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())
	require.NoError(t, err)
	app.SetTxDecoder(txConfig.TxDecoder())

	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	err = txBuilder.SetMsgs(msgs...)
	require.NoError(t, err)

	sigV2 := signing.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  defaultSignMode,
			Signature: nil,
		},
		Sequence: sequence,
	}
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
	sigV2, err = tx.SignWithPrivKey(
		context.TODO(),
		defaultSignMode,
		signerData,
		txBuilder,
		priv,
		txConfig,
		sequence,
	)
	if err != nil {
		return nil, err
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	return txBytes, err
}
