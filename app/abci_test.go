package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

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
	_, err = app.Commit()
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

	fmt.Println("finally!")

	// Test FinalizeBlock handler
	// finalizeReq := abci.RequestFinalizeBlock{
	// 	Txs:    [][]byte{extCommitBytes, txBytes},
	// 	Height: 3,
	// }
	// _, err = app.PreBlocker(ctx, &finalizeReq)
	// require.NoError(t, err)

	// // Commit the block
	// _, err = app.Commit()
	// require.NoError(t, err)
}

var defaultFeeAmount = big.NewInt(10).Exp(big.NewInt(10), big.NewInt(15), nil).Int64()
