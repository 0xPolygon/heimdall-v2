package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	stakingtypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const (
	TxHash1 = "0x000000000000000000000000000000000000000000000000000000000000dead"
)

func (s *KeeperTestSuite) TestMsgValidatorJoin() {
	ctx, msgServer, keeper, require := s.ctx, s.msgServer, s.stakeKeeper, s.Require()

	pk1 := ed25519.GenPrivKey().PubKey()
	require.NotNil(pk1)

	// Msg with wrong pub key
	msgValJoin := stakingtypes.MsgValidatorJoin{
		From:            pk1.Address().String(),
		ValId:           uint64(1),
		ActivationEpoch: uint64(1),
		Amount:          math.NewInt(int64(1000000000000000000)),
		SignerPubKey:    pk1.Bytes(),
		TxHash:          []byte{},
		LogIndex:        uint64(1),
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	require.Panics(func() {
		_, _ = msgServer.ValidatorJoin(ctx, &msgValJoin)
	})

	pk1 = secp256k1.GenPrivKey().PubKey()
	require.NotNil(pk1)

	msgValJoin = stakingtypes.MsgValidatorJoin{
		From:            pk1.Address().String(),
		ValId:           uint64(1),
		ActivationEpoch: uint64(1),
		Amount:          math.NewInt(int64(1000000000000000000)),
		SignerPubKey:    pk1.Bytes(),
		TxHash:          []byte{},
		LogIndex:        uint64(1),
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	_, err := msgServer.ValidatorJoin(ctx, &msgValJoin)
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
	ctx, msgServer, keeper, require, checkpointKeeper := s.ctx, s.msgServer, s.stakeKeeper, s.Require(), s.checkpointKeeper

	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0, 0)
	checkpointKeeper.EXPECT().GetAckCount(ctx).AnyTimes().Return(uint64(1), nil)

	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldSigner := oldValSet.Validators[0]
	newSigner := testutil.GenRandomVals(1, 0, 10, 10, false, 1, 0)
	newSigner[0].ValId = oldSigner.ValId
	newSigner[0].VotingPower = oldSigner.VotingPower

	msgSignerUpdateSameKey := stakingtypes.MsgSignerUpdate{
		From:            oldSigner.Signer,
		ValId:           oldSigner.ValId,
		NewSignerPubKey: oldSigner.GetPubKey(),
		TxHash:          []byte{},
		LogIndex:        uint64(0),
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	result, err := msgServer.SignerUpdate(ctx, &msgSignerUpdateSameKey)
	require.Error(err)
	require.Contains(err.Error(), "newSigner same as oldSigner")

	msgSignerUpdate := stakingtypes.MsgSignerUpdate{
		From:            newSigner[0].Signer,
		ValId:           uint64(1),
		NewSignerPubKey: newSigner[0].GetPubKey(),
		TxHash:          []byte{},
		LogIndex:        uint64(0),
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	result, err = msgServer.SignerUpdate(ctx, &msgSignerUpdate)

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
	ctx, msgServer, keeper, require, checkpointKeeper := s.ctx, s.msgServer, s.stakeKeeper, s.Require(), s.checkpointKeeper

	// pass 0 as time alive to generate non de-activated validators
	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0, 0)
	checkpointKeeper.EXPECT().GetAckCount(ctx).AnyTimes().Return(uint64(1), nil)

	validators := keeper.GetCurrentValidators(ctx)
	msgTxHash := common.Hex2Bytes(TxHash1)

	validators[0].EndEpoch = 10
	msgValidatorExit := stakingtypes.MsgValidatorExit{
		From:              validators[0].Signer,
		ValId:             uint64(1),
		DeactivationEpoch: validators[0].EndEpoch,
		TxHash:            msgTxHash,
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
	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0, 0)
	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldVal := oldValSet.Validators[0]

	msgTxHash := common.Hex2Bytes(TxHash1)
	newAmount := math.NewInt(2000000000000000000)

	msgStakeUpdate := stakingtypes.MsgStakeUpdate{
		From:        oldVal.Signer,
		ValId:       oldVal.ValId,
		NewAmount:   newAmount,
		TxHash:      msgTxHash,
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

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	pk1 := secp256k1.GenPrivKey().PubKey()
	require.NotNil(pk1)

	addr := pk1.Address().String()

	index := simulation.RandIntBetween(r, 0, 100)
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
		SignerPubKey:    pk1.Bytes(),
		TxHash:          []byte{},
		LogIndex:        logIndex,
		BlockNumber:     uint64(0),
		Nonce:           uint64(1),
	}

	_, err = msgServer.ValidatorJoin(ctx, &msgValJoin)
	require.NotNil(err)
}

func (s *KeeperTestSuite) TestValidatorPubKey() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()
	testutil.LoadRandomValidatorSet(require, 1, keeper, ctx, false, 0, 0)
	valPubKey := keeper.GetAllValidators(ctx)[0].GetPubKey()
	valAddr := keeper.GetAllValidators(ctx)[0].Signer
	modPubKey := secp256k1.PubKey{Key: valPubKey}
	require.Equal(valPubKey, modPubKey.Bytes())
	genAddress := util.FormatAddress(modPubKey.Address().String())
	require.Equal(valAddr, genAddress)
}
