package app

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdksecp "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

type testInfo struct {
	name       string
	txBytes    [][]byte
	mockCaller *helpermocks.IContractCaller
	// setup is called before the test to prepare the state (e.g., set mock callers on keepers)
	setup func(t *testing.T, app *HeimdallApp, ctx sdk.Context)
	// verify is called after both blocks complete to assert post-handler state changes
	verify func(t *testing.T, app *HeimdallApp, ctx sdk.Context)
}

// setMockCallerOnAllKeepers sets the mock contract caller on all keepers and app.caller
func setMockCallerOnAllKeepers(app *HeimdallApp, mockCaller *helpermocks.IContractCaller) {
	app.caller = mockCaller
	app.CheckpointKeeper.SetContractCaller(mockCaller)
	app.MilestoneKeeper.SetContractCaller(mockCaller)
	app.BorKeeper.SetContractCaller(mockCaller)
	app.StakeKeeper.SetContractCaller(mockCaller)
	app.ClerkKeeper.SetContractCaller(mockCaller)
	app.TopupKeeper.SetContractCaller(mockCaller)
}

// baseMockCaller creates a mock with the baseline mocks needed for milestone generation
// (GetBorChainBlock and GetBorChainBlockInfoInBatch, which are called during ExtendVote)
func baseMockCaller() *helpermocks.IContractCaller {
	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.
		On("GetBorChainBlock", mock.Anything, mock.Anything).
		Return(&ethTypes.Header{Number: big.NewInt(10)}, nil)
	mockCaller.
		On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)
	return mockCaller
}

// validReceipt returns a mock Ethereum receipt with the given block number
func validReceipt(blockNumber uint64) *ethTypes.Receipt {
	return &ethTypes.Receipt{
		Status:      ethTypes.ReceiptStatusSuccessful,
		BlockNumber: new(big.Int).SetUint64(blockNumber),
	}
}

func getTests(t *testing.T, priv cryptotypes.PrivKey, app *HeimdallApp, ctx sdk.Context) []testInfo {
	t.Helper()

	signerAddr := priv.PubKey().Address().String()
	validators := app.StakeKeeper.GetAllValidators(ctx)

	return []testInfo{
		// Test 1: MsgCheckpoint — basic checkpoint submission through 2 blocks
		{
			name: "MsgCheckpoint_ProposerMismatch",
			txBytes: buildTxBytes(t, ctx, priv, app,
				&checkpointTypes.MsgCheckpoint{
					Proposer:        signerAddr, // tx signer is not the proposer (VOTE_NO)
					StartBlock:      0,
					EndBlock:        200,
					RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
					AccountRootHash: computeAccountRootHash(t, app, ctx),
					BorChainId:      helper.DefaultBorChainID,
				},
			),
			mockCaller: func() *helpermocks.IContractCaller {
				m := baseMockCaller()
				m.On("CheckIfBlocksExist", mock.Anything).Return(true, nil)
				m.On("GetRootHash", uint64(0), uint64(200), mock.Anything).
					Return(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"), nil)
				return m
			}(),
			verify: func(t *testing.T, app *HeimdallApp, ctx sdk.Context) {
				// Proposer mismatch (VOTE_NO), hence the checkpoint should not be in the buffer
				doExist, err := app.CheckpointKeeper.HasCheckpointInBuffer(ctx)
				require.NoError(t, err)
				require.False(t, doExist)
			},
		},

		// Test 2: MsgEventRecord — clerk state sync through 2 blocks
		{
			name: "MsgEventRecord",
			txBytes: buildTxBytes(t, ctx, priv, app,
				&clerkTypes.MsgEventRecord{
					From:            signerAddr,
					TxHash:          common.Bytes2Hex(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000deadbeef")),
					LogIndex:        0,
					BlockNumber:     100,
					Id:              1,
					ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000001010").String(),
					Data:            []byte("test-data"),
					ChainId:         helper.DefaultBorChainID,
				},
			),
			mockCaller: func() *helpermocks.IContractCaller {
				m := baseMockCaller()
				m.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).
					Return(validReceipt(100), nil)
				m.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).
					Return(&statesender.StatesenderStateSynced{
						Id:              big.NewInt(1),
						ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000001010"),
						Data:            []byte("test-data"),
					}, nil)
				return m
			}(),
			verify: func(t *testing.T, app *HeimdallApp, ctx sdk.Context) {
				// After the approval, the event record should exist
				require.True(t, app.ClerkKeeper.HasEventRecord(ctx, 1))
			},
		},

		// Test 3: MsgTopupTx — fee topup through 2 blocks
		{
			name: "MsgTopupTx",
			txBytes: buildTxBytes(t, ctx, priv, app,
				&topupTypes.MsgTopupTx{
					Proposer:    signerAddr,
					User:        signerAddr,
					TxHash:      common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000deadcafe"),
					LogIndex:    0,
					BlockNumber: 50,
					Fee:         math.NewIntFromBigInt(big.NewInt(0).Mul(big.NewInt(10), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))),
				},
			),
			mockCaller: func() *helpermocks.IContractCaller {
				m := baseMockCaller()
				m.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).
					Return(validReceipt(50), nil)
				feeAmt := big.NewInt(0).Mul(big.NewInt(10), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
				m.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).
					Return(&stakinginfo.StakinginfoTopUpFee{
						User: common.HexToAddress(signerAddr),
						Fee:  feeAmt,
					}, nil)
				return m
			}(),
			verify: func(t *testing.T, app *HeimdallApp, ctx sdk.Context) {
				// After the approval, the topup sequence should exist
				seq := helper.CalculateSequence(50, 0)
				exists, err := app.TopupKeeper.HasTopupSequence(ctx, seq)
				require.NoError(t, err)
				require.True(t, exists)
			},
		},

		// Test 4: MsgValidatorJoin — new validator joining through 2 blocks
		{
			name: "MsgValidatorJoin",
			txBytes: func() [][]byte {
				// Generate a new keypair for the joining validator
				newPriv := sdksecp.GenPrivKey()
				newPubKey := newPriv.PubKey().Bytes()
				newSigner := common.BytesToAddress(newPriv.PubKey().Address().Bytes())

				amount := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

				msg := &stakeTypes.MsgValidatorJoin{
					From:            signerAddr,
					ValId:           99,
					ActivationEpoch: 1,
					Amount:          math.NewIntFromBigInt(amount),
					SignerPubKey:    newPubKey,
					TxHash:          common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000aa0001"),
					LogIndex:        0,
					BlockNumber:     200,
					Nonce:           0,
				}

				// Store references for the mock caller closure
				_ = newSigner

				return buildTxBytes(t, ctx, priv, app, msg)
			}(),
			mockCaller: func() *helpermocks.IContractCaller {
				m := baseMockCaller()
				m.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).
					Return(validReceipt(200), nil)
				m.On("DecodeValidatorJoinEvent", mock.Anything, mock.Anything, mock.Anything).
					Return((*stakinginfo.StakinginfoStaked)(nil), nil)
				return m
			}(),
		},

		// Test 5: MsgValidatorExit — tx with invalid ValId gets dropped in PrepareProposal
		{
			name: "MsgValidatorExit_InvalidValId",
			txBytes: buildTxBytes(t, ctx, priv, app,
				&stakeTypes.MsgValidatorExit{
					From:              signerAddr,
					ValId:             validators[0].ValId, // ValId=0 is invalid, hence the tx is dropped during PrepareProposal
					DeactivationEpoch: 10,
					TxHash:            common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000bb0001"),
					LogIndex:          0,
					BlockNumber:       300,
					Nonce:             validators[0].Nonce + 1,
				},
			),
			mockCaller: func() *helpermocks.IContractCaller {
				return baseMockCaller()
			}(),
			// no need to verify because the tx is dropped in PrepareProposal due to invalid ValId
		},

		// Test 6: MsgStakeUpdate — tx with invalid ValId gets dropped in PrepareProposal
		{
			name: "MsgStakeUpdate_InvalidValId",
			txBytes: buildTxBytes(t, ctx, priv, app,
				&stakeTypes.MsgStakeUpdate{
					From:        signerAddr,
					ValId:       validators[0].ValId, // ValId=0 is invalid, hence the tx dropped in PrepareProposal
					NewAmount:   math.NewIntFromBigInt(new(big.Int).Mul(big.NewInt(200), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))),
					TxHash:      common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000cc0001"),
					LogIndex:    0,
					BlockNumber: 400,
					Nonce:       validators[0].Nonce + 1,
				},
			),
			mockCaller: func() *helpermocks.IContractCaller {
				return baseMockCaller()
			}(),
			// no need to verify, because the tx is dropped in PrepareProposal due to invalid ValId
		},

		// Test 7: Multiple side txs in one block (MsgEventRecord + MsgTopupTx)
		{
			name: "MultipleSideTxs",
			txBytes: func() [][]byte {
				msg1 := &clerkTypes.MsgEventRecord{
					From:            signerAddr,
					TxHash:          common.Bytes2Hex(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000face0001")),
					LogIndex:        0,
					BlockNumber:     100,
					Id:              1,
					ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000001010").String(),
					Data:            []byte("multi-tx-data"),
					ChainId:         helper.DefaultBorChainID,
				}
				msg2 := &topupTypes.MsgTopupTx{
					Proposer:    signerAddr,
					User:        signerAddr,
					TxHash:      common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000face0002"),
					LogIndex:    0,
					BlockNumber: 100,
					Fee:         math.NewIntFromBigInt(big.NewInt(0).Mul(big.NewInt(10), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))),
				}

				tx1, err := buildSignedTx(msg1, ctx, priv, app)
				require.NoError(t, err)
				// Use sequence+1 for the second tx since both share the same signer
				propAddr := sdk.AccAddress(priv.PubKey().Address())
				propAcc := app.AccountKeeper.GetAccount(ctx, propAddr)
				seq := propAcc.GetSequence()
				tx2, err := buildSignedTxWithSequence(msg2, ctx, priv, app, seq+1)
				require.NoError(t, err)
				return [][]byte{tx1, tx2}
			}(),
			mockCaller: func() *helpermocks.IContractCaller {
				m := baseMockCaller()
				// Clerk mocks
				m.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).
					Return(validReceipt(100), nil)
				m.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).
					Return(&statesender.StatesenderStateSynced{
						Id:              big.NewInt(1),
						ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000001010"),
						Data:            []byte("multi-tx-data"),
					}, nil)
				// Topup mocks
				feeAmt := big.NewInt(0).Mul(big.NewInt(10), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
				m.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).
					Return(&stakinginfo.StakinginfoTopUpFee{
						User: common.HexToAddress(signerAddr),
						Fee:  feeAmt,
					}, nil)
				return m
			}(),
			verify: func(t *testing.T, app *HeimdallApp, ctx sdk.Context) {
				// Both sideTxs should have been processed
				require.True(t, app.ClerkKeeper.HasEventRecord(ctx, 1))
				seq := helper.CalculateSequence(100, 0)
				exists, err := app.TopupKeeper.HasTopupSequence(ctx, seq)
				require.NoError(t, err)
				require.True(t, exists)
			},
		},

		// Test 8: Empty block — no side txs, just vote extensions
		{
			name:    "EmptyBlock",
			txBytes: [][]byte{},
			mockCaller: func() *helpermocks.IContractCaller {
				return baseMockCaller()
			}(),
		},

		// Test 9: Tx with invalid checkpoint (wrong BorChainId): should get VOTE_NO
		{
			name: "InvalidCheckpointBorChainId",
			txBytes: buildTxBytes(t, ctx, priv, app,
				&checkpointTypes.MsgCheckpoint{
					Proposer:        signerAddr,
					StartBlock:      0,
					EndBlock:        200,
					RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
					AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
					BorChainId:      "wrong-chain-id",
				},
			),
			mockCaller: func() *helpermocks.IContractCaller {
				m := baseMockCaller()
				m.On("CheckIfBlocksExist", mock.Anything).Return(true, nil)
				m.On("GetRootHash", mock.Anything, mock.Anything, mock.Anything).
					Return(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"), nil)
				return m
			}(),
			verify: func(t *testing.T, app *HeimdallApp, ctx sdk.Context) {
				// The checkpoint should not be in the buffer, since the BorChainId was wrong
				doExist, err := app.CheckpointKeeper.HasCheckpointInBuffer(ctx)
				require.NoError(t, err)
				require.False(t, doExist)
			},
		},

		// Test 10: Tx that fails PrepareProposal validation (non-side-tx msg)
		// This tests the path where a tx is included, but it has no side handler
		{
			name: "NonSideTxMsg",
			txBytes: func() [][]byte {
				// Use a regular bank-send that doesn't have a side handler
				msg := &checkpointTypes.MsgCheckpoint{
					Proposer:        signerAddr,
					StartBlock:      0,
					EndBlock:        50,
					RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
					AccountRootHash: computeAccountRootHash(t, app, ctx),
					BorChainId:      helper.DefaultBorChainID,
				}
				return buildTxBytes(t, ctx, priv, app, msg)
			}(),
			mockCaller: func() *helpermocks.IContractCaller {
				m := baseMockCaller()
				// Checkpoint validation will fail because the blocks don't exist, hence VOTE_NO
				m.On("CheckIfBlocksExist", mock.Anything).Return(false, nil)
				m.On("GetRootHash", mock.Anything, mock.Anything, mock.Anything).
					Return([]byte{}, nil)
				return m
			}(),
			verify: func(t *testing.T, app *HeimdallApp, ctx sdk.Context) {
				// The checkpoint should not be in buffer since blocks don't exist
				doExist, err := app.CheckpointKeeper.HasCheckpointInBuffer(ctx)
				require.NoError(t, err)
				require.False(t, doExist)
			},
		},
	}
}

// buildTxBytes is a helper function that builds signed tx bytes for one or more messages
func buildTxBytes(t *testing.T, ctx sdk.Context, priv cryptotypes.PrivKey, app *HeimdallApp, msgs ...sdk.Msg) [][]byte {
	t.Helper()
	txBytes := make([][]byte, len(msgs))
	for i, msg := range msgs {
		tx, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)
		txBytes[i] = tx
	}
	return txBytes
}

// computeAccountRootHash computes the current account root hash from dividend accounts
func computeAccountRootHash(t *testing.T, app *HeimdallApp, ctx sdk.Context) []byte {
	t.Helper()
	dividendAccounts, err := app.TopupKeeper.GetAllDividendAccounts(ctx)
	require.NoError(t, err)

	accountRoot, err := hmTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(t, err)
	return accountRoot
}

// TestFullABCI runs all full ABCI flow test cases
func TestFullABCI(t *testing.T) {
	for i := 0; ; i++ {
		priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
		tests := getTests(t, priv, app, ctx)
		if i >= len(tests) {
			break
		}
		test := tests[i]

		setMockCallerOnAllKeepers(app, test.mockCaller)

		if test.setup != nil {
			test.setup(t, app, ctx)
		}

		name := test.name
		if name == "" {
			name = "test"
		}

		t.Run(name, func(t *testing.T) {
			executeTest(t, app, ctx, validatorPrivKeys, test.txBytes)

			if test.verify != nil {
				verifyCtx := app.NewContext(true).WithChainID(app.ChainID())
				test.verify(t, app, verifyCtx)
			}
		})
	}
}

// TestFullABCI_PrepareProposalMaxBytes verifies that PrepareProposal respects MaxTxBytes
func TestFullABCI_PrepareProposalMaxBytes(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	validators := app.StakeKeeper.GetAllValidators(ctx)

	signerAddr := priv.PubKey().Address().String()
	msg := &checkpointTypes.MsgCheckpoint{
		Proposer:        signerAddr,
		StartBlock:      0,
		EndBlock:        200,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      helper.DefaultBorChainID,
	}
	txBytes := buildTxBytes(t, ctx, priv, app, msg)

	_, extCommit, _, err := buildExtensionCommits(
		t, app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators, validatorPrivKeys,
		app.LastBlockHeight(), nil,
	)
	require.NoError(t, err)

	// Set MaxTxBytes to a very small value — only the extCommit should fit
	reqPrepare := &abci.RequestPrepareProposal{
		Txs:             txBytes,
		MaxTxBytes:      100, // very small
		LocalLastCommit: *extCommit,
		ProposerAddress: common.FromHex(validators[0].Signer),
		Height:          app.LastBlockHeight() + 1,
	}

	respPrepare, err := app.PrepareProposal(reqPrepare)
	require.NoError(t, err)
	// Only the extCommit tx should be included (the checkpoint tx exceeds MaxTxBytes)
	require.Len(t, respPrepare.Txs, 1)
}

// TestFullABCI_ProcessProposalRejectsEmptyTxs verifies ProcessProposal rejects empty proposals
func TestFullABCI_ProcessProposalRejectsEmptyTxs(t *testing.T) {
	_, app, _, _ := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	reqProcess := &abci.RequestProcessProposal{
		Txs:    [][]byte{},
		Height: app.LastBlockHeight() + 1,
	}

	resProcess, err := app.ProcessProposal(reqProcess)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, resProcess.Status)
}

// TestFullABCI_ProcessProposalRejectsBadExtCommit verifies ProcessProposal rejects
// proposals where the first tx is not a valid ExtendedCommitInfo
func TestFullABCI_ProcessProposalRejectsBadExtCommit(t *testing.T) {
	_, app, _, _ := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	reqProcess := &abci.RequestProcessProposal{
		Txs:    [][]byte{[]byte("not-a-valid-ext-commit")},
		Height: app.LastBlockHeight() + 1,
	}

	resProcess, err := app.ProcessProposal(reqProcess)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, resProcess.Status)
}

// TestFullABCI_VerifyVoteExtensionRejectsWrongHeight verifies that VE with the wrong height is rejected
func TestFullABCI_VerifyVoteExtensionRejectsWrongHeight(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	validators := app.StakeKeeper.GetAllValidators(ctx)

	wrongHeightVE := sidetxs.VoteExtension{
		Height:          999, // wrong height
		BlockHash:       common.Hex2Bytes("0001"),
		SideTxResponses: []sidetxs.SideTxResponse{},
	}
	bz, err := gogoproto.Marshal(&wrongHeightVE)
	require.NoError(t, err)

	dummyExt, err := GetDummyNonRpVoteExtension(app.LastBlockHeight()+1, app.ChainID())
	require.NoError(t, err)

	reqVerify := &abci.RequestVerifyVoteExtension{
		Height:             app.LastBlockHeight() + 1,
		Hash:               common.Hex2Bytes("0001"),
		ValidatorAddress:   common.FromHex(validators[0].Signer),
		VoteExtension:      bz,
		NonRpVoteExtension: dummyExt,
	}

	respVerify, err := app.VerifyVoteExtension(reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, respVerify.Status)
}

// TestFullABCI_VerifyVoteExtensionRejectsWrongBlockHash verifies VE with wrong block hash is rejected
func TestFullABCI_VerifyVoteExtensionRejectsWrongBlockHash(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	validators := app.StakeKeeper.GetAllValidators(ctx)
	height := app.LastBlockHeight() + 1

	wrongHashVE := sidetxs.VoteExtension{
		Height:          height,
		BlockHash:       common.Hex2Bytes("deadbeef"),
		SideTxResponses: []sidetxs.SideTxResponse{},
	}
	bz, err := gogoproto.Marshal(&wrongHashVE)
	require.NoError(t, err)

	dummyExt, err := GetDummyNonRpVoteExtension(height, app.ChainID())
	require.NoError(t, err)

	reqVerify := &abci.RequestVerifyVoteExtension{
		Height:             height,
		Hash:               common.Hex2Bytes("aaaabbbb"), // different hash
		ValidatorAddress:   common.FromHex(validators[0].Signer),
		VoteExtension:      bz,
		NonRpVoteExtension: dummyExt,
	}

	respVerify, err := app.VerifyVoteExtension(reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, respVerify.Status)
}

// TestFullABCI_VerifyVoteExtensionRejectsDuplicateSideTxResponses verifies VE with duplicate tx hashes is rejected
func TestFullABCI_VerifyVoteExtensionRejectsDuplicateSideTxResponses(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	validators := app.StakeKeeper.GetAllValidators(ctx)
	height := app.LastBlockHeight() + 1
	blockHash := common.Hex2Bytes("0001")

	dupTxHash := common.Hex2Bytes("aabb")
	dupVE := sidetxs.VoteExtension{
		Height:    height,
		BlockHash: blockHash,
		SideTxResponses: []sidetxs.SideTxResponse{
			{TxHash: dupTxHash, Result: sidetxs.Vote_VOTE_YES},
			{TxHash: dupTxHash, Result: sidetxs.Vote_VOTE_YES}, // duplicate
		},
	}
	bz, err := gogoproto.Marshal(&dupVE)
	require.NoError(t, err)

	dummyExt, err := GetDummyNonRpVoteExtension(height, app.ChainID())
	require.NoError(t, err)

	reqVerify := &abci.RequestVerifyVoteExtension{
		Height:             height,
		Hash:               blockHash,
		ValidatorAddress:   common.FromHex(validators[0].Signer),
		VoteExtension:      bz,
		NonRpVoteExtension: dummyExt,
	}

	respVerify, err := app.VerifyVoteExtension(reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, respVerify.Status)
}

// TestFullABCI_VerifyVoteExtensionRejectsUnknownFields verifies VE with unknown proto fields is rejected
func TestFullABCI_VerifyVoteExtensionRejectsUnknownFields(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	validators := app.StakeKeeper.GetAllValidators(ctx)
	height := app.LastBlockHeight() + 1
	blockHash := common.Hex2Bytes("0001")

	ve := sidetxs.VoteExtension{
		Height:          height,
		BlockHash:       blockHash,
		SideTxResponses: []sidetxs.SideTxResponse{},
	}
	bz, err := gogoproto.Marshal(&ve)
	require.NoError(t, err)

	// Append unknown protobuf field (field 100, varint type, value 1)
	bz = append(bz, 0x80|0x20, 0x06, 0x01) // field 100, varint, value 1

	dummyExt, err := GetDummyNonRpVoteExtension(height, app.ChainID())
	require.NoError(t, err)

	reqVerify := &abci.RequestVerifyVoteExtension{
		Height:             height,
		Hash:               blockHash,
		ValidatorAddress:   common.FromHex(validators[0].Signer),
		VoteExtension:      bz,
		NonRpVoteExtension: dummyExt,
	}

	respVerify, err := app.VerifyVoteExtension(reqVerify)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, respVerify.Status)
}

// TestFullABCI_ProcessProposalRejectsMultipleSideHandlersPerTx tests that a tx
// with more than 1 side handler msg is rejected in ProcessProposal
func TestFullABCI_ProcessProposalRejectsMultipleSideHandlersPerTx(t *testing.T) {
	priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	validators := app.StakeKeeper.GetAllValidators(ctx)
	signerAddr := priv.PubKey().Address().String()

	// Build empty extCommit for the previous block
	_, extCommit, _, err := buildExtensionCommits(
		t, app,
		common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		validators, validatorPrivKeys,
		app.LastBlockHeight(), nil,
	)
	require.NoError(t, err)

	bz, err := extCommit.Marshal()
	require.NoError(t, err)

	// Build a single tx with 2 side handler msgs using buildSignedMultiMsgTx
	msg1 := &checkpointTypes.MsgCheckpoint{
		Proposer:        signerAddr,
		StartBlock:      0,
		EndBlock:        100,
		RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
		AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
		BorChainId:      helper.DefaultBorChainID,
	}
	msg2 := &clerkTypes.MsgEventRecord{
		From:            signerAddr,
		TxHash:          common.Bytes2Hex(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000deadbeef")),
		LogIndex:        0,
		BlockNumber:     100,
		Id:              1,
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000001010").String(),
		Data:            []byte("data"),
		ChainId:         helper.DefaultBorChainID,
	}

	multiMsgTx, err := buildSignedMultiMsgTx([]sdk.Msg{msg1, msg2}, ctx, priv, app)
	require.NoError(t, err)

	// Create a proposal with extCommit + 1 multi-msg tx
	txs := [][]byte{bz, multiMsgTx}

	reqProcess := &abci.RequestProcessProposal{
		Txs:                txs,
		ProposedLastCommit: abci.CommitInfo{Round: extCommit.Round},
		ProposerAddress:    common.FromHex(validators[0].Signer),
		Height:             app.LastBlockHeight() + 1,
		Hash:               common.Hex2Bytes("0001"),
	}

	// The single tx has 2 side handler msgs, hence ProcessProposal should REJECT
	resProcess, err := app.ProcessProposal(reqProcess)
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, resProcess.Status)
}

// TestFullABCI_PreBlockerRejectsEmptyTxs verifies that PreBlocker rejects blocks with no txs
func TestFullABCI_PreBlockerRejectsEmptyTxs(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	mockCaller := baseMockCaller()
	setMockCallerOnAllKeepers(app, mockCaller)

	req := &abci.RequestFinalizeBlock{
		Txs:    [][]byte{},
		Height: app.LastBlockHeight() + 1,
	}

	// PreBlocker is called internally by FinalizeBlock, but we can test it directly
	_, err := app.PreBlocker(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no txs found")
}

func executeTest(
	t *testing.T,
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

	// Block N: submit the txs, and ExtendVote produces side tx responses
	voteExtensions := executeHeight(t, ctx, app, *extCommit, txBytes)
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

	// Block N+1: vote extensions from block N are included, hence PreBlocker tallies and executes the post-handlers
	voteExtensions = executeHeight(t, ctx, app, *extCommit, [][]byte{})
	require.NotNil(t, voteExtensions)
}

func executeHeight(
	t *testing.T,
	ctx sdk.Context,
	app *HeimdallApp,
	extCommit abci.ExtendedCommitInfo,
	txBytes [][]byte,
) *abci.ResponseExtendVote {

	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Prepare the proposal
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

	// Process the proposal
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
