package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	stakingtypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const (
	TxHash = "0x000000000000000000000000000000000000000000000000000000000000dead"
)

func (s *KeeperTestSuite) TestMsgValidatorJoin() {
	ctx, msgServer, keeper, require := s.ctx, s.msgServer, s.stakeKeeper, s.Require()

	pk1 := secp256k1.GenPrivKey().PubKey()
	require.NotNil(pk1)

	pubKey, err := codectypes.NewAnyWithValue(pk1)
	require.NoError(err)

	msgValJoin := stakingtypes.MsgValidatorJoin{
		From:            pk1.Address().String(),
		ValId:           uint64(1),
		ActivationEpoch: uint64(1),
		Amount:          math.NewInt(int64(1000000000000000000)),
		SignerPubKey:    pubKey,
		TxHash:          hmTypes.TxHash{},
		LogIndex:        uint64(1),
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	_, err = msgServer.ValidatorJoin(ctx, &msgValJoin)
	require.NoError(err)

	_, err = keeper.GetValidatorFromValID(ctx, uint64(1))
	require.NotNilf(err, "Should not add validator")

	votingPower, err := helper.GetPowerFromAmount(msgValJoin.Amount.BigInt())
	require.NoError(err)

	newValidator := types.Validator{
		ValId:       msgValJoin.ValId,
		StartEpoch:  msgValJoin.ActivationEpoch,
		EndEpoch:    0,
		Nonce:       msgValJoin.Nonce,
		VotingPower: votingPower.Int64(),
		PubKey:      msgValJoin.SignerPubKey,
		Signer:      msgValJoin.From,
		LastUpdated: "",
	}

	err = keeper.AddValidator(ctx, newValidator)
	require.NoError(err)

	_, err = msgServer.ValidatorJoin(ctx, &msgValJoin)
	require.NotNil(err)
}

func (s *KeeperTestSuite) TestHandleMsgSignerUpdate() {
	ctx, msgServer, keeper, require := s.ctx, s.msgServer, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldSigner := oldValSet.Validators[0]
	newSigner := testutil.GenRandomVals(1, 0, 10, 10, false, 1)
	newSigner[0].ValId = oldSigner.ValId
	newSigner[0].VotingPower = oldSigner.VotingPower

	msgSignerUpdate := stakingtypes.MsgSignerUpdate{
		From:            newSigner[0].Signer,
		ValId:           uint64(1),
		NewSignerPubKey: newSigner[0].GetPubKey(),
		TxHash:          hmTypes.TxHash{},
		LogIndex:        uint64(0),
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	result, err := msgServer.SignerUpdate(ctx, &msgSignerUpdate)

	require.NoErrorf(err, "expected validator update to be ok, got %v", result)

	newValidators := keeper.GetCurrentValidators(ctx)
	require.Equal(len(oldValSet.Validators), len(newValidators), "Number of current validators should be equal")

	setUpdates := types.GetUpdatedValidators(&oldValSet, keeper.GetAllValidators(ctx), 5)

	err = oldValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)

	err = keeper.UpdateValidatorSetInStore(ctx, oldValSet)
	require.NoError(err)

	ValFrmID, err := keeper.GetValidatorFromValID(ctx, oldSigner.ValId)
	require.Nilf(err, "signer should be found, got %v", err)
	require.NotEqual(oldSigner.Signer, newSigner[0].Signer, "Should not update state")
	require.Equalf(ValFrmID.VotingPower, oldSigner.VotingPower, "VotingPower of new signer %v should be equal to old signer %v", ValFrmID.VotingPower, oldSigner.VotingPower)

	removedVal, err := keeper.GetValidatorInfo(ctx, oldSigner.Signer)
	require.Empty(err)
	require.NotEqual(removedVal.VotingPower, int64(0), "should not update state")
}

func (s *KeeperTestSuite) TestHandleMsgValidatorExit() {
	ctx, msgServer, keeper, require := s.ctx, s.msgServer, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	validators := keeper.GetCurrentValidators(ctx)
	msgTxHash := common.Hex2Bytes(TxHash)

	validators[0].EndEpoch = 10
	msgValidatorExit := stakingtypes.MsgValidatorExit{
		From:              validators[0].Signer,
		ValId:             uint64(1),
		DeactivationEpoch: validators[0].EndEpoch,
		TxHash:            hmTypes.TxHash{Hash: msgTxHash},
		LogIndex:          uint64(0),
		BlockNumber:       uint64(0),
		Nonce:             uint64(1),
	}

	_, err := msgServer.ValidatorExit(ctx, &msgValidatorExit)
	require.NoError(err, "expected validator exit to be ok")

	updatedValInfo, err := keeper.GetValidatorInfo(ctx, validators[0].Signer)
	require.NoErrorf(err, "Unable to get validator info from val address, valAddr: %v error: %v ", validators[0].Signer, err)
	require.NotEqual(updatedValInfo.EndEpoch, validators[0].EndEpoch, "should not update deactivation epoch")

	_, err = keeper.GetValidatorFromValID(ctx, validators[0].ValId)
	require.Nilf(err, "Validator should be present even after deactivation")

	_, err = msgServer.ValidatorExit(ctx, &msgValidatorExit)
	require.NoError(err, "should not fail, as state is not updated for validatorExit")
}

func (s *KeeperTestSuite) TestHandleMsgStakeUpdate() {
	ctx, msgServer, keeper, require := s.ctx, s.msgServer, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldVal := oldValSet.Validators[0]

	msgTxHash := common.Hex2Bytes(TxHash)
	newAmount := math.NewInt(2000000000000000000)

	msgStakeUpdate := stakingtypes.MsgStakeUpdate{
		From:        oldVal.Signer,
		ValId:       oldVal.ValId,
		NewAmount:   newAmount,
		TxHash:      hmTypes.TxHash{Hash: msgTxHash},
		LogIndex:    uint64(0),
		BlockNumber: uint64(0),
		Nonce:       uint64(1),
	}

	_, err = msgServer.StakeUpdate(ctx, &msgStakeUpdate)
	require.NoError(err, "expected validator stake update to be ok")

	updatedVal, err := keeper.GetValidatorInfo(ctx, oldVal.Signer)
	require.NoErrorf(err, "unable to fetch validator info %v", err)
	require.NotEqualf(newAmount.Int64(), updatedVal.VotingPower, "Validator VotingPower should not be updated to %v", newAmount.Int64())
}

func (s *KeeperTestSuite) TestExitedValidatorJoiningAgain() {
	ctx, msgServer, keeper, require := s.ctx, s.msgServer, s.stakeKeeper, s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	pk1 := secp256k1.GenPrivKey().PubKey()
	require.NotNil(pk1)

	pubKey, err := codectypes.NewAnyWithValue(pk1)
	require.NoError(err)

	addr := pk1.Address().String()

	index := simulation.RandIntBetween(r1, 0, 100)
	logIndex := uint64(index)

	validatorId := uint64(1)
	validator, err := types.NewValidator(
		validatorId,
		10,
		15,
		1,
		int64(0), // power
		pk1,
		addr,
	)

	require.NoError(err)

	err = keeper.AddValidator(ctx, *validator)

	require.NoError(err)

	isCurrentValidator := validator.IsCurrentValidator(14)
	require.False(isCurrentValidator)

	totalValidators := keeper.GetAllValidators(ctx)
	require.Equal(totalValidators[0].Signer, addr)
	msgValJoin := stakingtypes.MsgValidatorJoin{
		From:            addr,
		ValId:           validatorId,
		ActivationEpoch: uint64(1),
		Amount:          math.NewInt(int64(100000)),
		SignerPubKey:    pubKey,
		TxHash:          hmTypes.TxHash{},
		LogIndex:        logIndex,
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	_, err = msgServer.ValidatorJoin(ctx, &msgValJoin)
	require.NotNil(err)
}

func (s *KeeperTestSuite) TestTopupSuccessBeforeValidatorJoin() {
	/* TODO HV2: @Vaibhav topup is available, hence this test has to be fixed and enabled. With real data (e.g. not "123" as a valid hash)
	ctx, msgServer, keeper,require := s.ctx, s.msgServer, s.stakeKeeper,s.Require()

	pubKey := hmTypes.NewPubKey([]byte{123})
	signerAddress := hmTypes.HexToHeimdallAddress(pubKey.Address().Hex())

	txHash := hmTypes.HexToHeimdallHash("123")
	logIndex := uint64(2)
	amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)

	validatorId := hmTypes.NewValidatorID(uint64(1))

	chainParams := app.ChainKeeper.GetParams(ctx)

	msgTopup := topupTypes.NewMsgTopup(signerAddress, signerAddress, sdk.NewInt(2000000000000000000), txHash, logIndex, uint64(2))

	stakingInfoTopUpFee := &stakingInfo.stakingInfoTopUpFee{
		User: signerAddress.EthAddress(),
		Fee:  big.NewInt(100000000000000000),
	}

	txReceipt := &ethTypes.Receipt{
		BlockNumber: big.NewInt(10),
	}

	stakingInfoStaked := &stakingInfo.StakingInfoStaked{
		Signer:          signerAddress.EthAddress(),
		ValidatorId:     new(big.Int).SetUint64(validatorId.Uint64()),
		ActivationEpoch: big.NewInt(1),
		Amount:          amount,
		Total:           big.NewInt(10),
		SignerPubkey:    pubKey.Bytes()[1:],
	}

	msgValJoin := types.NewMsgValidatorJoin(
		signerAddress,
		validatorId.Uint64(),
		uint64(1),
		sdk.NewInt(2000000000000000000),
		pubKey,
		txHash,
		logIndex,
		0,
		1,
	)

	suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

	suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

	suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), mock.Anything, msgTopup.LogIndex).Return(stakinginfoTopUpFee, nil)

	topupResult := suite.topupHandler(ctx, msgTopup)
	require.True(t, topupResult.IsOK(), "expected topup to be done, got %v", topupResult)

	result := suite.handler(ctx, msgValJoin)
	require.True(t, result.IsOK(), "expected validator stake update to be ok, got %v", result)
	*/
}
