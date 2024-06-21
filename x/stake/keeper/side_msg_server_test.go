package keeper_test

import (
	"encoding/json"
	"errors"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var addressCodec = address.HexCodec{}

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) hmModule.Vote {
	cfg := s.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote hmModule.Vote) {
	cfg := s.sideMsgCfg

	cfg.GetPostHandler(msg)(ctx, msg, vote)
}

// TODO HV2: Maybe add testcases based on type of keys and ensure that only eth keys pass? For example:
//  1. Compressed & Uncompressed (i.e Eth) format. The former should fail.
//  2. Different key implementations of cryptotypes.PubKey (e.g. ed25519).
//     (because of this type assertion: pubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey))
func (s *KeeperTestSuite) TestSideHandleMsgValidatorJoin() {
	ctx, require := s.ctx, s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	txHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}
	index := simulation.RandIntBetween(r1, 0, 100)
	logIndex := uint64(index)
	validatorId := uint64(1)
	amount, _ := big.NewInt(0).SetString("1000000000000000000", 10)

	pubKey := secp256k1.GenPrivKey().PubKey()
	require.NotNil(pubKey)

	addr := pubKey.Address()

	chainParams, err := s.cmKeeper.GetParams(ctx)
	require.NoError(err)

	blockNumber := big.NewInt(10)
	nonce := big.NewInt(3)

	s.Run("Success", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)
		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_YES)
	})

	s.Run("No receipt", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(nil, nil)
		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")

	})

	s.Run("No EventLog", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)
		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(nil, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Signer pubKey", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			secp256k1.GenPrivKey().PubKey(),
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)
		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Signer addr", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address{},
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)
		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Validator Id", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			uint64(10),
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     big.NewInt(1),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Activation Epoch", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(10),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Amount", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(100000000000000000),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Block Number", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			uint64(20),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid nonce", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			uint64(9),
		)

		require.NoError(err)

		stakingInfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(addr.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubKey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, msgValJoin.LogIndex).Return(stakingInfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})
}

func (s *KeeperTestSuite) TestSideHandleMsgSignerUpdate() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	stakeSim.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldSigner := oldValSet.Validators[0]
	newSigner := stakeSim.GenRandomVals(2, 0, 10, 10, false, 1)
	newSigner[0].ValId = oldSigner.ValId
	newSigner[0].VotingPower = oldSigner.VotingPower
	chainParams, err := s.cmKeeper.GetParams(ctx)
	require.NoError(err)

	blockNumber := big.NewInt(10)
	nonce := big.NewInt(5)

	oldSignerBytes, err := addressCodec.StringToBytes(oldSigner.Signer)
	require.NoError(err)

	oldSignerAddress := common.BytesToAddress(oldSignerBytes)

	newSigner0Bytes, err := addressCodec.StringToBytes(newSigner[0].Signer)
	require.NoError(err)

	newSigner0Address := common.BytesToAddress(newSigner0Bytes)

	newSigner0Pk, ok := newSigner[0].PubKey.GetCachedValue().(cryptotypes.PubKey)
	require.True(ok)

	newSigner1Pk, ok := newSigner[1].PubKey.GetCachedValue().(cryptotypes.PubKey)
	require.True(ok)

	// gen msg
	msgTxHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}

	s.Run("Success", func() {
		msg, err := types.NewMsgSignerUpdate(newSigner[0].Signer, oldSigner.ValId, newSigner0Pk, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ValId),
			Nonce:        nonce,
			OldSigner:    oldSignerAddress,
			NewSigner:    newSigner0Address,
			SignerPubkey: newSigner0Pk.Bytes()[1:],
		}

		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_YES, "Side tx handler should Pass")
	})

	s.Run("No event log", func() {
		msg, err := types.NewMsgSignerUpdate(newSigner[0].Signer, oldSigner.ValId, newSigner0Pk, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: blockNumber}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(nil, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")

	})

	s.Run("Invalid BlockNumber", func() {
		msg, err := types.NewMsgSignerUpdate(
			newSigner[0].Signer,
			oldSigner.ValId,
			newSigner0Pk,
			msgTxHash,
			0,
			uint64(9),
			nonce.Uint64(),
		)

		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ValId),
			Nonce:        nonce,
			OldSigner:    oldSignerAddress,
			NewSigner:    newSigner0Address,
			SignerPubkey: newSigner0Pk.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid validator", func() {
		msg, err := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(6), newSigner0Pk, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ValId),
			Nonce:        nonce,
			OldSigner:    oldSignerAddress,
			NewSigner:    newSigner0Address,
			SignerPubkey: newSigner0Pk.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid signer pubKey", func() {
		msg, err := types.NewMsgSignerUpdate(newSigner[0].Signer, oldSigner.ValId, newSigner1Pk, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ValId),
			Nonce:        nonce,
			OldSigner:    oldSignerAddress,
			NewSigner:    newSigner0Address,
			SignerPubkey: newSigner0Pk.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid new signer address", func() {
		msg, err := types.NewMsgSignerUpdate(common.Address{}.Hex(), oldSigner.ValId, newSigner0Pk, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ValId),
			Nonce:        nonce,
			OldSigner:    oldSignerAddress,
			NewSigner:    common.Address{},
			SignerPubkey: newSigner0Pk.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid nonce", func() {
		msg, err := types.NewMsgSignerUpdate(newSigner[0].Signer, oldSigner.ValId, newSigner0Pk, msgTxHash, 0, blockNumber.Uint64(), uint64(12))
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ValId),
			Nonce:        nonce,
			OldSigner:    oldSignerAddress,
			NewSigner:    newSigner0Address,
			SignerPubkey: newSigner0Pk.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})
}

func (s *KeeperTestSuite) TestSideHandleMsgValidatorExit() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	stakeSim.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	validators := keeper.GetCurrentValidators(ctx)
	msgTxHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}
	chainParams, err := s.cmKeeper.GetParams(ctx)
	require.NoError(err)
	logIndex := uint64(0)
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(9)

	validator0Bytes, err := addressCodec.StringToBytes(validators[0].Signer)
	require.NoError(err)

	validator0Address := common.BytesToAddress(validator0Bytes)

	s.Run("Success", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakingInfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validator0Address,
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ValId),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, logIndex).Return(stakingInfoUnstakeInit, nil)

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_YES, "Side tx handler should Fail")
	})

	s.Run("No Receipt", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(nil, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakingInfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validator0Address,
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ValId),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, logIndex).Return(stakingInfoUnstakeInit, nil)

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		require.NoError(err)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("No event log", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		validators[0].EndEpoch = 10

		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, logIndex).Return(nil, nil)

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid BlockNumber", func() {
		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)

		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		stakingInfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validator0Address,
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ValId),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, logIndex).Return(stakingInfoUnstakeInit, nil)

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			validators[0].EndEpoch,
			msgTxHash,
			0,
			uint64(5),
			nonce.Uint64(),
		)
		require.NoError(err)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid validatorId", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakingInfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validator0Address,
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ValId),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, logIndex).Return(stakingInfoUnstakeInit, nil)

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(66),
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid DeactivationEpoch", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakingInfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validator0Address,
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ValId),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}

		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, logIndex).Return(stakingInfoUnstakeInit, nil)

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			uint64(1000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Nonce", func() {
		txReceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakingInfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validator0Address,
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ValId),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, logIndex).Return(stakingInfoUnstakeInit, nil)

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			uint64(6),
		)
		require.NoError(err)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})
}

func (s *KeeperTestSuite) TestSideHandleMsgStakeUpdate() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	stakeSim.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldVal := oldValSet.Validators[0]

	chainParams, err := s.cmKeeper.GetParams(ctx)
	require.NoError(err)

	msgTxHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(1)

	s.Run("Success", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		stakingInfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ValId),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}
		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(stakingInfoStakeUpdate, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_YES, "Side tx handler should succeed")
	})

	s.Run("No Receipt", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(nil, nil)

		stakingInfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ValId),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}
		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(stakingInfoStakeUpdate, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("No event log", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)
		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(nil, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid BlockNumber", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			uint64(15),
			nonce.Uint64())
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		stakingInfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ValId),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}
		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(stakingInfoStakeUpdate, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid ValidatorID", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			uint64(13),
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())

		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		stakingInfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ValId),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}
		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(stakingInfoStakeUpdate, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Amount", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(200000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())

		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		stakingInfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ValId),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}
		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(stakingInfoStakeUpdate, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Nonce", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			uint64(9))
		require.NoError(err)

		txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		s.contractCaller.On("GetConfirmedTxReceipt", common.Hash(msgTxHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

		stakingInfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ValId),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}
		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress, txReceipt, uint64(0)).Return(stakingInfoStakeUpdate, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgValidatorJoin() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	txHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}
	index := simulation.RandIntBetween(r1, 0, 100)
	logIndex := uint64(index)
	validatorId := uint64(1)

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := pubKey.Address()

	blockNumber := big.NewInt(10)
	nonce := big.NewInt(3)

	s.Run("No Result", func() {

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		s.postHandler(ctx, msgValJoin, hmModule.Vote_VOTE_NO)

		_, err = keeper.GetValidatorFromValID(ctx, validatorId)
		require.Errorf(err, "Should not add validator")
	})

	s.Run("Success", func() {
		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		s.postHandler(ctx, msgValJoin, hmModule.Vote_VOTE_YES)

		actualResult, err := keeper.GetValidatorFromValID(ctx, validatorId)
		require.Nil(err, "Should add validator")
		require.NotNil(actualResult, "got %v", actualResult)
	})

	s.Run("Replay", func() {
		blockNumber := big.NewInt(11)

		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		s.postHandler(ctx, msgValJoin, hmModule.Vote_VOTE_YES)

		actualResult, err := keeper.GetValidatorFromValID(ctx, validatorId)
		require.Nil(err, "Should add validator")
		require.NotNil(actualResult, "got %v", actualResult)

		s.postHandler(ctx, msgValJoin, hmModule.Vote_VOTE_YES)
	})

	s.Run("Invalid Power", func() {
		msgValJoin, err := types.NewMsgValidatorJoin(
			addr.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1)),
			pubKey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		s.postHandler(ctx, msgValJoin, hmModule.Vote_VOTE_YES)
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgSignerUpdate() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	stakeSim.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldSigner := oldValSet.Validators[0]
	newSigner := stakeSim.GenRandomVals(1, 0, 10, 10, false, 1)
	newSigner[0].ValId = oldSigner.ValId
	newSigner[0].VotingPower = oldSigner.VotingPower
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(5)

	// gen msg
	msgTxHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}

	newSigner0Pk, ok := newSigner[0].PubKey.GetCachedValue().(cryptotypes.PubKey)
	require.True(ok)

	s.Run("No Success", func() {
		msg, err := types.NewMsgSignerUpdate(newSigner[0].Signer, oldSigner.ValId, newSigner0Pk, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())
		require.NoError(err)

		s.postHandler(ctx, msg, hmModule.Vote_VOTE_NO)
	})

	s.Run("Success", func() {
		msg, err := types.NewMsgSignerUpdate(newSigner[0].Signer, oldSigner.ValId, newSigner0Pk, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		s.postHandler(ctx, msg, hmModule.Vote_VOTE_YES)

		newValidators := keeper.GetCurrentValidators(ctx)
		require.Equal(len(oldValSet.Validators), len(newValidators), "Number of current validators should be equal")

		setUpdates := types.GetUpdatedValidators(&oldValSet, keeper.GetAllValidators(ctx), 5)
		err = oldValSet.UpdateWithChangeSet(setUpdates)
		require.NoError(err)
		err = keeper.UpdateValidatorSetInStore(ctx, oldValSet)
		require.NoError(err)

		ValFrmID, err := keeper.GetValidatorFromValID(ctx, oldSigner.ValId)
		require.NoErrorf(err, "new signer should be found, got %v", ok)
		require.Equal(ValFrmID.Signer, newSigner[0].Signer, "New Signer should be mapped to old validator ID")
		require.Equal(ValFrmID.VotingPower, oldSigner.VotingPower, "VotingPower of new signer %v should be equal to old signer %v", ValFrmID.VotingPower, oldSigner.VotingPower)

		removedVal, err := keeper.GetValidatorInfo(ctx, oldSigner.Signer)
		require.Empty(err, "deleted validator should be found, got %v", err)
		require.Equal(removedVal.VotingPower, int64(0), "removed validator VotingPower should be zero")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgValidatorExit() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	stakeSim.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	validators := keeper.GetCurrentValidators(ctx)
	msgTxHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(9)

	s.Run("No Success", func() {
		validators[0].EndEpoch = 10

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		s.postHandler(ctx, msg, hmModule.Vote_VOTE_NO)
	})

	s.Run("Success", func() {
		validators[0].EndEpoch = 10

		msg, err := types.NewMsgValidatorExit(
			validators[0].Signer,
			validators[0].ValId,
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)
		require.NoError(err)

		s.postHandler(ctx, msg, hmModule.Vote_VOTE_YES)

		currentVals := keeper.GetCurrentValidators(ctx)
		require.Equal(4, len(currentVals), "No of current validators should exist before epoch passes")

		s.checkpointKeeper.AckCount = 20
		currentVals = keeper.GetCurrentValidators(ctx)
		require.Equal(3, len(currentVals), "No of current validators should reduce after epoch passes")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgStakeUpdate() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// pass 0 as time alive to generate non de-activated validators
	stakeSim.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	oldVal := oldValSet.Validators[0]

	msgTxHash := hmTypes.TxHash{Hash: common.Hash{}.Bytes()}
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(1)
	newAmount := new(big.Int).SetInt64(2000000000000000000)

	s.Run("No result", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())
		require.NoError(err)

		s.postHandler(ctx, msg, hmModule.Vote_VOTE_NO)

		updatedVal, err := keeper.GetValidatorInfo(ctx, oldVal.Signer)
		require.Empty(err, "unable to fetch validator info %v-", err)

		actualPower, err := helper.GetPowerFromAmount(newAmount)
		require.NoError(err)
		require.NotEqual(actualPower.Int64(), updatedVal.VotingPower, "Validator VotingPower should be updated to %v", newAmount.Uint64())
	})

	s.Run("Success", func() {
		msg, err := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ValId,
			math.NewInt(int64(2000000000000000000)),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())
		require.NoError(err)

		s.postHandler(ctx, msg, hmModule.Vote_VOTE_YES)

		updatedVal, err := keeper.GetValidatorInfo(ctx, oldVal.Signer)
		require.Empty(err, "unable to fetch validator info %v-", err)

		actualPower, err := helper.GetPowerFromAmount(new(big.Int).SetInt64(2000000000000000000))
		require.NoError(err)
		require.Equal(actualPower.Int64(), updatedVal.VotingPower, "Validator VotingPower should be updated to %v", newAmount.Uint64())
	})
}

func TestEventCheck(t *testing.T) {
	t.Parallel()

	eventLogs := []string{
		`{
		"type": "0x2",
		"root": "0x",
		"status": "0x1",
		"cumulativeGasUsed": "0x155957",
		"logsBloom": "0x20000000000000000000000000800008000000000000000000001000000000000200000040000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000200001000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000010000000000000000000000000000000200000000000000004000000000",
		"logs": [
		  {
				"address": "0xa59c847bd5ac0172ff4fe912c5d29e5a71a7512b",
				"topics": [
				  "0x086044c0612a8c965d4cccd907f0d588e40ad68438bd4c1274cac60f4c3a9d1f",
				  "0x0000000000000000000000000000000000000000000000000000000000000013",
				  "0x00000000000000000000000072f93a2740e00112d5f2cef404c0aa16fae21fa4",
				  "0x0000000000000000000000003a5f70ac0551d5fae2b2379c6e558f6b7efa6a0d"
				],
				"data": "0x000000000000000000000000000000000000000000000000000000000000039400000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000040c8df79b1015f5bc235c9242bee499f2a5e25ad2c33b70561c2ba67118a00a1fe024dccfb65960ab91dfcd30b521fa1aa692b21ca251ec8063b2fc0a343a1b0bc",
				"blockNumber": "0x116fbce",
				"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
				"transactionIndex": "0x10",
				"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
				"logIndex": "0x14",
				"removed": false
		  }
		],
		"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
		"contractAddress": "0x0000000000000000000000000000000000000000",
		"gasUsed": "0x75a65",
		"effectiveGasPrice": "0x20fb31dfe",
		"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
		"blockNumber": "0x116fbce",
		"transactionIndex": "0x10"
	  }`,
		`{
		"type": "0x2",
		"root": "0x",
		"status": "0x1",
		"cumulativeGasUsed": "0x155957",
		"logsBloom": "0x20000000000000000000000000800008000000000000000000001000000000000200000040000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000200001000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000010000000000000000000000000000000200000000000000004000000000",
		"logs": [
			{
				"address": "0xa59c847bd5ac0172ff4fe912c5d29e5a71a7512b",
				"topics": [
					"0x35af9eea1f0e7b300b0a14fae90139a072470e44daa3f14b5069bebbc1265bda",
					"0x0000000000000000000000000000000000000000000000000000000000000013",
					"0x00000000000000000000000072f93a2740e00112d5f2cef404c0aa16fae21fa4",
					"0x0000000000000000000000003a5f70ac0551d5fae2b2379c6e558f6b7efa6a0d"
				],
				"data": "0x000000000000000000000000000000000000000000000000000000000000039400000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000040c8df79b1015f5bc235c9242bee499f2a5e25ad2c33b70561c2ba67118a00a1fe024dccfb65960ab91dfcd30b521fa1aa692b21ca251ec8063b2fc0a343a1b0bc",
				"blockNumber": "0x116fbce",
				"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
				"transactionIndex": "0x10",
				"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
				"logIndex": "0x14",
				"removed": false
			}
			],
			"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
			"contractAddress": "0x0000000000000000000000000000000000000000",
			"gasUsed": "0x75a65",
			"effectiveGasPrice": "0x20fb31dfe",
			"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
			"blockNumber": "0x116fbce",
			"transactionIndex": "0x10"
		}`,
	}

	testCases := []struct {
		actualEventLog    string
		disguisedEventLog string
		decodeEventName   string
		expectedErr       error
	}{
		{
			actualEventLog:    eventLogs[1],
			disguisedEventLog: eventLogs[0],
			decodeEventName:   "DecodeValidatorStakeUpdateEvent",
			expectedErr:       errors.New("topic event mismatch"),
		},
		{
			actualEventLog:    eventLogs[0],
			disguisedEventLog: eventLogs[1],
			decodeEventName:   "DecodeSignerUpdateEvent",
			expectedErr:       errors.New("topic event mismatch"),
		},
	}

	receipt := ethTypes.Receipt{}

	for _, tc := range testCases {
		err := json.Unmarshal([]byte(tc.disguisedEventLog), &receipt)
		if err != nil {
			t.Error(err)
			return
		}

		err = decodeEvent(t, tc.decodeEventName, receipt)
		if err == nil {
			t.Error(err)
			return
		}

		assert.EqualError(t, err, tc.expectedErr.Error())

		err = json.Unmarshal([]byte(tc.actualEventLog), &receipt)
		if err != nil {
			t.Error(err)
			return
		}

		err = decodeEvent(t, tc.decodeEventName, receipt)
		if err != nil {
			t.Error(err)
			return
		}

		assert.NoError(t, err)
	}
}

func decodeEvent(t *testing.T, eventName string, receipt ethTypes.Receipt) error {
	t.Helper()

	var err error
	contractCaller, err := helper.NewContractCaller()

	if err != nil {
		t.Error("Error creating contract caller")
	}

	switch eventName {
	case "DecodeNewHeaderBlockEvent":
		_, err = contractCaller.DecodeNewHeaderBlockEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeValidatorStakeUpdateEvent":
		_, err = contractCaller.DecodeValidatorStakeUpdateEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeSignerUpdateEvent":
		_, err = contractCaller.DecodeSignerUpdateEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeValidatorTopupFeesEvent":
		_, err = contractCaller.DecodeValidatorTopupFeesEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeValidatorJoinEvent":
		_, err = contractCaller.DecodeValidatorJoinEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeValidatorExitEvent":
		_, err = contractCaller.DecodeValidatorExitEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeStateSyncedEvent":
		_, err = contractCaller.DecodeStateSyncedEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeSlashedEvent":
		_, err = contractCaller.DecodeSlashedEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	case "DecodeUnJailedEvent":
		_, err = contractCaller.DecodeUnJailedEvent(receipt.Logs[0].Address.Hex(), &receipt, uint64(receipt.Logs[0].Index))

	default:
		return errors.New("unrecognized event")
	}

	return err
}
