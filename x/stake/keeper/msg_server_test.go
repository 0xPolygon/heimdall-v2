package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	stakingtypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestMsgValidatorJoin() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()

	pk1 := secp256k1.GenPrivKey().PubKey()
	require.NotNil(pk1)

	pubkey, err := codectypes.NewAnyWithValue(pk1)
	require.NoError(err)

	msgValJoin := stakingtypes.MsgValidatorJoin{
		From:            pk1.Address().String(),
		ValId:           uint64(1),
		ActivationEpoch: uint64(1),
		Amount:          math.NewInt(int64(1000000000000000000)),
		SignerPubKey:    pubkey,
		TxHash:          hmTypes.TxHash{},
		LogIndex:        uint64(1),
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	_, err = msgServer.ValidatorJoin(ctx, &msgValJoin)
	require.NoError(err)

	_, ok := keeper.GetValidatorFromValID(ctx, uint64(1))
	require.False(false, ok, "Should not add validator")
}

func (s *KeeperTestSuite) TestHandleMsgSignerUpdate() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()
	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet := keeper.GetValidatorSet(ctx)

	oldSigner := oldValSet.Validators[0]
	newSigner := testutil.GenRandomVal(1, 0, 10, 10, false, 1)
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

	require.NoError(err, "expected validator update to be ok, got %v", result)

	newValidators := keeper.GetCurrentValidators(ctx)
	require.Equal(len(oldValSet.Validators), len(newValidators), "Number of current validators should be equal")

	setUpdates := types.GetUpdatedValidators(&oldValSet, keeper.GetAllValidators(ctx), 5)

	err = oldValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)

	_ = keeper.UpdateValidatorSetInStore(ctx, oldValSet)

	ValFrmID, ok := keeper.GetValidatorFromValID(ctx, oldSigner.ValId)
	require.True(ok, "signer should be found, got %v", ok)
	require.NotEqual(oldSigner.Signer, newSigner[0].Signer, "Should not update state")
	require.Equal(ValFrmID.VotingPower, oldSigner.VotingPower, "VotingPower of new signer %v should be equal to old signer %v", ValFrmID.VotingPower, oldSigner.VotingPower)

	removedVal, err := keeper.GetValidatorInfo(ctx, oldSigner.Signer)
	require.Empty(err)
	require.NotEqual(removedVal.VotingPower, int64(0), "should not update state")
}

func (s *KeeperTestSuite) TestHandleMsgValidatorExit() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()
	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 0)
	validators := keeper.GetCurrentValidators(ctx)
	msgTxHash := hmTypes.HexToHeimdallHash("123")

	validators[0].EndEpoch = 10
	msgValidatorExit := stakingtypes.MsgValidatorExit{
		From:              validators[0].Signer,
		ValId:             uint64(1),
		DeactivationEpoch: validators[0].EndEpoch,
		TxHash:            hmTypes.TxHash(msgTxHash),
		LogIndex:          uint64(0),
		BlockNumber:       uint64(0),
		Nonce:             uint64(1),
	}

	_, err := msgServer.ValidatorExit(ctx, &msgValidatorExit)

	require.NoError(err, "expected validator exit to be ok")

	updatedValInfo, err := keeper.GetValidatorInfo(ctx, validators[0].Signer)

	require.NoError(err, "Unable to get validator info from val address,ValAddr:%v Error:%v ", validators[0].Signer, err)
	require.NotEqual(updatedValInfo.EndEpoch, validators[0].EndEpoch, "should not update deactivation epoch")

	_, found := keeper.GetValidatorFromValID(ctx, validators[0].ValId)
	require.True(found, "Validator should be present even after deactivation")

	_, err = msgServer.ValidatorExit(ctx, &msgValidatorExit)
	require.NoError(err, "should not fail, as state is not updated for validatorExit")
}

func (s *KeeperTestSuite) TestHandleMsgStakeUpdate() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()

	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet := keeper.GetValidatorSet(ctx)
	oldVal := oldValSet.Validators[0]

	msgTxHash := hmTypes.HexToHeimdallHash("123")
	newAmount := math.NewInt(2000000000000000000)

	msgStakeUpdate := stakingtypes.MsgStakeUpdate{
		From:        oldVal.Signer,
		ValId:       oldVal.ValId,
		NewAmount:   newAmount,
		TxHash:      hmTypes.TxHash(msgTxHash),
		LogIndex:    uint64(0),
		BlockNumber: uint64(0),
		Nonce:       uint64(1),
	}

	_, err := msgServer.StakeUpdate(ctx, &msgStakeUpdate)
	require.NoError(err, "expected validator stake update to be ok")

	updatedVal, err := keeper.GetValidatorInfo(ctx, oldVal.Signer)
	require.NoError(err, "unable to fetch validator info %v-", err)
	require.NotEqual(newAmount.Int64(), updatedVal.VotingPower, "Validator VotingPower should not be updated to %v", newAmount.Int64())
}

func (s *KeeperTestSuite) TestExitedValidatorJoiningAgain() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	pk1 := secp256k1.GenPrivKey().PubKey()
	require.NotNil(pk1)

	pubkey, err := codectypes.NewAnyWithValue(pk1)
	require.NoError(err)

	addr := pk1.Address().String()

	index := simulation.RandIntBetween(r1, 0, 100)
	logIndex := uint64(index)

	validatorId := uint64(1)
	validator := types.NewValidator(
		validatorId,
		10,
		15,
		1,
		int64(0), // power
		pk1,
		addr,
	)

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
		SignerPubKey:    pubkey,
		TxHash:          hmTypes.TxHash{},
		LogIndex:        logIndex,
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	_, err = msgServer.ValidatorJoin(ctx, &msgValJoin)
	require.NotNil(err)
}

// TODO HV2 Please implement the following test after writing topUp module
/*
func (s *KeeperTestSuite) TestTopupSuccessBeforeValidatorJoin() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()

	pubKey := hmTypes.NewPubKey([]byte{123})
	signerAddress := hmTypes.HexToHeimdallAddress(pubKey.Address().Hex())

	txHash := hmTypes.HexToHeimdallHash("123")
	logIndex := uint64(2)
	amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)

	validatorId := hmTypes.NewValidatorID(uint64(1))

	chainParams := app.ChainKeeper.GetParams(ctx)

	msgTopup := topupTypes.NewMsgTopup(signerAddress, signerAddress, sdk.NewInt(2000000000000000000), txHash, logIndex, uint64(2))

	stakinginfoTopUpFee := &stakinginfo.StakinginfoTopUpFee{
		User: signerAddress.EthAddress(),
		Fee:  big.NewInt(100000000000000000),
	}

	txreceipt := &ethTypes.Receipt{
		BlockNumber: big.NewInt(10),
	}

	stakinginfoStaked := &stakinginfo.StakinginfoStaked{
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
}
*/
