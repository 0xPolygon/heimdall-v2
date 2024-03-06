package keeper_test

import (
	"math/big"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	voteTypes "github.com/0xPolygon/heimdall-v2/x/types"
)

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) voteTypes.Vote {
	cfg := s.sideMsgCfg
	return cfg.SideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote voteTypes.Vote) {
	cfg := s.sideMsgCfg

	cfg.PostHandler(msg)(ctx, msg, vote)
}

func (s *KeeperTestSuite) TestSideHandleMsgValidatorJoin() {
	ctx, _, _ := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	txHash := hmTypes.TxHash{[]byte("123")}
	index := simulation.RandIntBetween(r1, 0, 100)
	logIndex := uint64(index)
	validatorId := uint64(1)
	amount, _ := big.NewInt(0).SetString("1000000000000000000", 10)

	pubkey := secp256k1.GenPrivKey().PubKey()
	require.NotNil(pubkey)

	address := pubkey.Address()

	chainParams := s.cmKeeper.GetParams(ctx)
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(3)

	s.Run("Success", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_YES)
	})

	s.Run("No receipt", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")

	})

	s.Run("No EventLog", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(nil, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Signer pubkey", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
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

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Signer address", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          hmTypes.ZeroPubKey.Address(),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Validator Id", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			uint64(10),
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     big.NewInt(1),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Activation Epoch", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(10),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Amount", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(1),
			math.NewInt(100000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid Block Number", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			uint64(20),
			nonce.Uint64(),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})

	s.Run("Invalid nonce", func() {
		s.contractCaller.Mock = mock.Mock{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin, err := types.NewMsgValidatorJoin(
			address.String(),
			validatorId,
			uint64(1),
			math.NewInt(int64(1000000000000000000)),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			uint64(9),
		)

		require.NoError(err)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          common.Address(address.Bytes()),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		s.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress, txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := s.sideHandler(ctx, msgValJoin)
		require.Equal(result, voteTypes.Vote_VOTE_NO, "Side tx handler should Fail")
	})
}

func (s *KeeperTestSuite) TestSideHandleMsgSignerUpdate() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
	require := s.Require()

	keeper := suite.app.StakingKeeper
	// pass 0 as time alive to generate non de-activated validators
	stakeSim.LoadValidatorSet(require, 4, keeper, ctx, false, 0)
	oldValSet := keeper.GetValidatorSet(ctx)

	oldSigner := oldValSet.Validators[0]
	newSigner := stakingSim.GenRandomVal(1, 0, 10, 10, false, 1)
	newSigner[0].ID = oldSigner.ID
	newSigner[0].VotingPower = oldSigner.VotingPower
	chainParams := app.ChainKeeper.GetParams(ctx)
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(5)

	// gen msg
	msgTxHash := hmTypes.HexToHeimdallHash("123")

	s.Run("Success", func() {
		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}

		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.Equal(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should be success")
		require.Equal(t, abci.SideTxResultType_Yes, result.Result, "Result should be `yes`")
	})

	s.Run("No Eventlog", func() {
		s.contractCaller.Mock = mock.Mock{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}

		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(nil, nil)

		result := s.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeErrDecodeEvent), result.Code)
	})

	s.Run("Invalid BlockNumber", func() {
		s.contractCaller.Mock = mock.Mock{}

		msg := types.NewMsgSignerUpdate(
			newSigner[0].Signer, uint64(oldSigner.ID),
			newSigner[0].PubKey,
			msgTxHash,
			0,
			uint64(9),
			nonce.Uint64(),
		)

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	s.Run("Invalid validator", func() {
		s.contractCaller.Mock = mock.Mock{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(6), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	s.Run("Invalid signer pubkey", func() {
		s.contractCaller.Mock = mock.Mock{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), hmTypes.NewPubKey([]byte{123}), msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	s.Run("Invalid new signer address", func() {
		s.contractCaller.Mock = mock.Mock{}

		msg := types.NewMsgSignerUpdate(hmTypes.ZeroHeimdallAddress, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    hmTypes.ZeroHeimdallAddress.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	s.Run("Invalid nonce", func() {
		s.contractCaller.Mock = mock.Mock{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), uint64(12))

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		s.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := s.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})
}

// func (s *KeeperTestSuite) TestSideHandleMsgValidatorExit() {
// 	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
// 	keeper := app.StakingKeeper
// 	// pass 0 as time alive to generate non de-activated validators
// 	chSim.LoadValidatorSet(t, 4, keeper, ctx, false, 0)
// 	validators := keeper.GetCurrentValidators(ctx)
// 	msgTxHash := hmTypes.HexToHeimdallHash("123")
// 	chainParams := app.ChainKeeper.GetParams(ctx)
// 	logIndex := uint64(0)
// 	blockNumber := big.NewInt(10)
// 	nonce := big.NewInt(9)

// 	s.Run("Success", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		txreceipt := &ethTypes.Receipt{
// 			BlockNumber: blockNumber,
// 		}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
// 		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
// 			User:              validators[0].Signer.EthAddress(),
// 			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
// 			Nonce:             nonce,
// 			DeactivationEpoch: big.NewInt(10),
// 			Amount:            amount,
// 		}
// 		validators[0].EndEpoch = 10

// 		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := s.sideHandler(ctx, msg)
// 		require.Equal(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should be success")
// 		require.Equal(t, abci.SideTxResultType_Yes, result.Result, "Result should be `yes`")
// 	})

// 	s.Run("No Receipt", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		txreceipt := &ethTypes.Receipt{
// 			BlockNumber: blockNumber,
// 		}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)

// 		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
// 		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
// 			User:              validators[0].Signer.EthAddress(),
// 			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
// 			Nonce:             nonce,
// 			DeactivationEpoch: big.NewInt(10),
// 			Amount:            amount,
// 		}
// 		validators[0].EndEpoch = 10

// 		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("No Eventlog", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		txreceipt := &ethTypes.Receipt{
// 			BlockNumber: blockNumber,
// 		}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		validators[0].EndEpoch = 10

// 		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(nil, nil)

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid BlockNumber", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)

// 		txreceipt := &ethTypes.Receipt{
// 			BlockNumber: blockNumber,
// 		}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
// 			User:              validators[0].Signer.EthAddress(),
// 			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
// 			Nonce:             nonce,
// 			DeactivationEpoch: big.NewInt(10),
// 			Amount:            amount,
// 		}
// 		validators[0].EndEpoch = 10

// 		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			uint64(5),
// 			nonce.Uint64(),
// 		)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid validatorId", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		txreceipt := &ethTypes.Receipt{
// 			BlockNumber: blockNumber,
// 		}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
// 		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
// 			User:              validators[0].Signer.EthAddress(),
// 			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
// 			Nonce:             nonce,
// 			DeactivationEpoch: big.NewInt(10),
// 			Amount:            amount,
// 		}
// 		validators[0].EndEpoch = 10

// 		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(66),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid DeactivationEpoch", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		txreceipt := &ethTypes.Receipt{
// 			BlockNumber: blockNumber,
// 		}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
// 		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
// 			User:              validators[0].Signer.EthAddress(),
// 			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
// 			Nonce:             nonce,
// 			DeactivationEpoch: big.NewInt(10),
// 			Amount:            amount,
// 		}

// 		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			uint64(1000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid Nonce", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		txreceipt := &ethTypes.Receipt{
// 			BlockNumber: blockNumber,
// 		}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
// 		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
// 			User:              validators[0].Signer.EthAddress(),
// 			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
// 			Nonce:             nonce,
// 			DeactivationEpoch: big.NewInt(10),
// 			Amount:            amount,
// 		}
// 		validators[0].EndEpoch = 10

// 		s.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			uint64(6),
// 		)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})
// }

// func (s *KeeperTestSuite) TestSideHandleMsgStakeUpdate() {
// 	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
// 	keeper := app.StakingKeeper

// 	// pass 0 as time alive to generate non de-activated validators
// 	chSim.LoadValidatorSet(t, 4, keeper, ctx, false, 0)
// 	oldValSet := keeper.GetValidatorSet(ctx)
// 	oldVal := oldValSet.Validators[0]

// 	chainParams := app.ChainKeeper.GetParams(ctx)

// 	msgTxHash := hmTypes.HexToHeimdallHash("123")
// 	blockNumber := big.NewInt(10)
// 	nonce := big.NewInt(1)

// 	s.Run("Success", func() {
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64())

// 		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
// 			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
// 			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
// 			Nonce:       nonce,
// 		}
// 		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

// 		result := s.sideHandler(ctx, msg)
// 		require.Equal(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should be success")
// 		require.Equal(t, abci.SideTxResultType_Yes, result.Result, "Result should be `yes`")
// 	})

// 	s.Run("No Receipt", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64())

// 		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)

// 		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
// 			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
// 			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
// 			Nonce:       nonce,
// 		}
// 		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("No Eventlog", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64())

// 		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)
// 		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(nil, nil)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid BlockNumber", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			uint64(15),
// 			nonce.Uint64())

// 		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
// 			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
// 			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
// 			Nonce:       nonce,
// 		}
// 		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid ValidatorID", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			uint64(13),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64())

// 		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
// 			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
// 			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
// 			Nonce:       nonce,
// 		}
// 		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid Amount", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(200000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64())

// 		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
// 			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
// 			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
// 			Nonce:       nonce,
// 		}
// 		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})

// 	s.Run("Invalid Nonce", func() {
// 		s.contractCaller.Mock = mock.Mock{}
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			uint64(9))

// 		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
// 		s.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

// 		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
// 			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
// 			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
// 			Nonce:       nonce,
// 		}
// 		s.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

// 		result := s.sideHandler(ctx, msg)
// 		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
// 		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
// 	})
// }

// func (s *KeeperTestSuite) TestPostHandler() {
// 	t, ctx := suite.T(), suite.ctx

// 	// post tx handler
// 	result := suite.postHandler(ctx, nil, abci.SideTxResultType_Yes)
// 	require.False(t, result.IsOK(), "Post handler should fail")
// 	require.Equal(t, sdk.CodeUnknownRequest, result.Code)
// }

// func (s *KeeperTestSuite) TestPostHandleMsgValidatorJoin() {
// 	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
// 	s1 := rand.NewSource(time.Now().UnixNano())
// 	r1 := rand.New(s1)
// 	txHash := hmTypes.HexToHeimdallHash("123")
// 	index := simulation.RandIntBetween(r1, 0, 100)
// 	logIndex := uint64(index)
// 	validatorId := uint64(1)

// 	privKey1 := secp256k1.GenPrivKey()
// 	pubkey := hmTypes.NewPubKey(privKey1.PubKey().Bytes())
// 	address := pubkey.Address()

// 	blockNumber := big.NewInt(10)
// 	nonce := big.NewInt(3)

// 	s.Run("No Result", func() {

// 		msgValJoin := types.NewMsgValidatorJoin(
// 			address.String(),
// 			validatorId,
// 			uint64(1),
// 			math.NewInt(int64(1000000000000000000)),
// 			pubkey,
// 			txHash,
// 			logIndex,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := suite.postHandler(ctx, msgValJoin, abci.SideTxResultType_No)
// 		require.False(t, result.IsOK(), "Post handler should fail")
// 		require.Equal(t, common.CodeSideTxValidationFailed, result.Code)
// 		require.Equal(t, 0, len(result.Events), "No error should be emitted for failed post-tx")

// 		_, ok := app.StakingKeeper.GetValidatorFromValID(ctx, hmTypes.ValidatorID(validatorId))
// 		require.False(t, ok, "Should not add validator")
// 	})

// 	s.Run("Success", func() {
// 		msgValJoin := types.NewMsgValidatorJoin(
// 			address.String(),
// 			validatorId,
// 			uint64(1),
// 			math.NewInt(int64(1000000000000000000)),
// 			pubkey,
// 			txHash,
// 			logIndex,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := suite.postHandler(ctx, msgValJoin, abci.SideTxResultType_Yes)
// 		require.True(t, result.IsOK(), "expected validator join to be ok, got %v", result)

// 		actualResult, ok := app.StakingKeeper.GetValidatorFromValID(ctx, hmTypes.ValidatorID(validatorId))
// 		require.True(t, ok, "Should add validator")
// 		require.NotNil(t, actualResult, "got %v", actualResult)
// 	})

// 	s.Run("Replay", func() {
// 		blockNumber := big.NewInt(11)

// 		msgValJoin := types.NewMsgValidatorJoin(
// 			address.String(),
// 			validatorId,
// 			uint64(1),
// 			math.NewInt(int64(1000000000000000000)),
// 			pubkey,
// 			txHash,
// 			logIndex,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := suite.postHandler(ctx, msgValJoin, abci.SideTxResultType_Yes)
// 		require.True(t, result.IsOK(), "expected validator join to be ok, got %v", result)

// 		actualResult, ok := app.StakingKeeper.GetValidatorFromValID(ctx, hmTypes.ValidatorID(validatorId))
// 		require.True(t, ok, "Should add validator")
// 		require.NotNil(t, actualResult, "got %v", actualResult)

// 		result = suite.postHandler(ctx, msgValJoin, abci.SideTxResultType_Yes)
// 		require.False(t, result.IsOK(), "expected validator join to be ok, got %v", result)
// 	})

// 	s.Run("Invalid Power", func() {
// 		msgValJoin := types.NewMsgValidatorJoin(
// 			address.String(),
// 			validatorId,
// 			uint64(1),
// 			sdk.NewInt(1),
// 			pubkey,
// 			txHash,
// 			logIndex,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := suite.postHandler(ctx, msgValJoin, abci.SideTxResultType_Yes)
// 		require.True(t, !result.IsOK(), errs.CodeToDefaultMsg(result.Code))
// 	})
// }

// func (s *KeeperTestSuite) TestPostHandleMsgSignerUpdate() {
// 	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
// 	keeper := app.StakingKeeper
// 	// pass 0 as time alive to generate non de-activated validators
// 	chSim.LoadValidatorSet(t, 4, keeper, ctx, false, 0)
// 	oldValSet := keeper.GetValidatorSet(ctx)

// 	oldSigner := oldValSet.Validators[0]
// 	newSigner := stakingSim.GenRandomVal(1, 0, 10, 10, false, 1)
// 	newSigner[0].ID = oldSigner.ID
// 	newSigner[0].VotingPower = oldSigner.VotingPower
// 	blockNumber := big.NewInt(10)
// 	nonce := big.NewInt(5)

// 	// gen msg
// 	msgTxHash := hmTypes.HexToHeimdallHash("123")

// 	s.Run("No Success", func() {
// 		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())
// 		result := suite.postHandler(ctx, msg, abci.SideTxResultType_No)
// 		require.True(t, !result.IsOK(), errs.CodeToDefaultMsg(result.Code))
// 	})

// 	s.Run("Success", func() {
// 		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

// 		result := suite.postHandler(ctx, msg, abci.SideTxResultType_Yes)
// 		require.True(t, result.IsOK(), "Post handler should succeed")

// 		newValidators := keeper.GetCurrentValidators(ctx)
// 		require.Equal(t, len(oldValSet.Validators), len(newValidators), "Number of current validators should be equal")

// 		setUpdates := helper.GetUpdatedValidators(&oldValSet, keeper.GetAllValidators(ctx), 5)
// 		err := oldValSet.UpdateWithChangeSet(setUpdates)
// 		require.NoError(t, err)
// 		err = keeper.UpdateValidatorSetInStore(ctx, oldValSet)
// 		require.NoError(t, err)

// 		ValFrmID, ok := keeper.GetValidatorFromValID(ctx, oldSigner.ID)
// 		require.True(t, ok, "new signer should be found, got %v", ok)
// 		require.Equal(t, ValFrmID.Signer.Bytes(), newSigner[0].Signer.Bytes(), "New Signer should be mapped to old validator ID")
// 		require.Equal(t, ValFrmID.VotingPower, oldSigner.VotingPower, "VotingPower of new signer %v should be equal to old signer %v", ValFrmID.VotingPower, oldSigner.VotingPower)

// 		removedVal, err := keeper.GetValidatorInfo(ctx, oldSigner.Signer.Bytes())
// 		require.Empty(t, err, "deleted validator should be found, got %v", err)
// 		require.Equal(t, removedVal.VotingPower, int64(0), "removed validator VotingPower should be zero")
// 	})
// }

// func (s *KeeperTestSuite) TestPostHandleMsgValidatorExit() {
// 	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
// 	keeper := app.StakingKeeper
// 	// pass 0 as time alive to generate non de-activated validators
// 	chSim.LoadValidatorSet(t, 4, keeper, ctx, false, 0)
// 	validators := keeper.GetCurrentValidators(ctx)
// 	msgTxHash := hmTypes.HexToHeimdallHash("123")
// 	blockNumber := big.NewInt(10)
// 	nonce := big.NewInt(9)

// 	s.Run("No Success", func() {
// 		validators[0].EndEpoch = 10

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := suite.postHandler(ctx, msg, abci.SideTxResultType_No)
// 		require.True(t, !result.IsOK(), errs.CodeToDefaultMsg(result.Code))
// 	})

// 	s.Run("Success", func() {
// 		validators[0].EndEpoch = 10

// 		msg := types.NewMsgValidatorExit(
// 			validators[0].Signer,
// 			uint64(validators[0].ID),
// 			validators[0].EndEpoch,
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64(),
// 		)

// 		result := suite.postHandler(ctx, msg, abci.SideTxResultType_Yes)
// 		require.True(t, result.IsOK(), "Post handler should succeed")

// 		currentVals := keeper.GetCurrentValidators(ctx)
// 		require.Equal(t, 4, len(currentVals), "No of current validators should exist before epoch passes")

// 		app.CheckpointKeeper.UpdateACKCountWithValue(ctx, 20)
// 		currentVals = keeper.GetCurrentValidators(ctx)
// 		require.Equal(t, 3, len(currentVals), "No of current validators should reduce after epoch passes")
// 	})
// }

// func (s *KeeperTestSuite) TestPostHandleMsgStakeUpdate() {
// 	ctx, msgServer, keeper := s.ctx, s.msgServer, s.stakeKeeper
// 	keeper := app.StakingKeeper

// 	// pass 0 as time alive to generate non de-activated validators
// 	chSim.LoadValidatorSet(t, 4, keeper, ctx, false, 0)
// 	oldValSet := keeper.GetValidatorSet(ctx)
// 	oldVal := oldValSet.Validators[0]

// 	msgTxHash := hmTypes.HexToHeimdallHash("123")
// 	blockNumber := big.NewInt(10)
// 	nonce := big.NewInt(1)
// 	newAmount := new(big.Int).SetInt64(2000000000000000000)

// 	s.Run("No result", func() {
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64())

// 		result := suite.postHandler(ctx, msg, abci.SideTxResultType_No)
// 		require.True(t, !result.IsOK(), errs.CodeToDefaultMsg(result.Code))

// 		updatedVal, err := keeper.GetValidatorInfo(ctx, oldVal.Signer.Bytes())
// 		require.Empty(t, err, "unable to fetch validator info %v-", err)

// 		acctualPower, err := helper.GetPowerFromAmount(newAmount)
// 		require.NoError(t, err)
// 		require.NotEqual(t, acctualPower.Int64(), updatedVal.VotingPower, "Validator VotingPower should be updated to %v", newAmount.Uint64())
// 	})

// 	s.Run("Success", func() {
// 		msg := types.NewMsgStakeUpdate(
// 			oldVal.Signer,
// 			oldVal.ID.Uint64(),
// 			sdk.NewInt(2000000000000000000),
// 			msgTxHash,
// 			0,
// 			blockNumber.Uint64(),
// 			nonce.Uint64())

// 		result := suite.postHandler(ctx, msg, abci.SideTxResultType_Yes)
// 		require.True(t, result.IsOK(), "expected validator stake update to be ok, got %v", result)

// 		updatedVal, err := keeper.GetValidatorInfo(ctx, oldVal.Signer.Bytes())
// 		require.Empty(t, err, "unable to fetch validator info %v-", err)

// 		acctualPower, err := helper.GetPowerFromAmount(new(big.Int).SetInt64(2000000000000000000))
// 		require.NoError(t, err)
// 		require.Equal(t, acctualPower.Int64(), updatedVal.VotingPower, "Validator VotingPower should be updated to %v", newAmount.Uint64())
// 	})
// }

// func TestEventCheck(t *testing.T) {
// 	t.Parallel()

// 	eventLogs := []string{
// 		`{
// 		"type": "0x2",
// 		"root": "0x",
// 		"status": "0x1",
// 		"cumulativeGasUsed": "0x155957",
// 		"logsBloom": "0x20000000000000000000000000800008000000000000000000001000000000000200000040000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000200001000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000010000000000000000000000000000000200000000000000004000000000",
// 		"logs": [
// 		  {
// 				"address": "0xa59c847bd5ac0172ff4fe912c5d29e5a71a7512b",
// 				"topics": [
// 				  "0x086044c0612a8c965d4cccd907f0d588e40ad68438bd4c1274cac60f4c3a9d1f",
// 				  "0x0000000000000000000000000000000000000000000000000000000000000013",
// 				  "0x00000000000000000000000072f93a2740e00112d5f2cef404c0aa16fae21fa4",
// 				  "0x0000000000000000000000003a5f70ac0551d5fae2b2379c6e558f6b7efa6a0d"
// 				],
// 				"data": "0x000000000000000000000000000000000000000000000000000000000000039400000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000040c8df79b1015f5bc235c9242bee499f2a5e25ad2c33b70561c2ba67118a00a1fe024dccfb65960ab91dfcd30b521fa1aa692b21ca251ec8063b2fc0a343a1b0bc",
// 				"blockNumber": "0x116fbce",
// 				"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
// 				"transactionIndex": "0x10",
// 				"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
// 				"logIndex": "0x14",
// 				"removed": false
// 		  }
// 		],
// 		"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
// 		"contractAddress": "0x0000000000000000000000000000000000000000",
// 		"gasUsed": "0x75a65",
// 		"effectiveGasPrice": "0x20fb31dfe",
// 		"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
// 		"blockNumber": "0x116fbce",
// 		"transactionIndex": "0x10"
// 	  }`,
// 		`{
// 		"type": "0x2",
// 		"root": "0x",
// 		"status": "0x1",
// 		"cumulativeGasUsed": "0x155957",
// 		"logsBloom": "0x20000000000000000000000000800008000000000000000000001000000000000200000040000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000200001000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000010000000000000000000000000000000200000000000000004000000000",
// 		"logs": [
// 			{
// 				"address": "0xa59c847bd5ac0172ff4fe912c5d29e5a71a7512b",
// 				"topics": [
// 					"0x35af9eea1f0e7b300b0a14fae90139a072470e44daa3f14b5069bebbc1265bda",
// 					"0x0000000000000000000000000000000000000000000000000000000000000013",
// 					"0x00000000000000000000000072f93a2740e00112d5f2cef404c0aa16fae21fa4",
// 					"0x0000000000000000000000003a5f70ac0551d5fae2b2379c6e558f6b7efa6a0d"
// 				],
// 				"data": "0x000000000000000000000000000000000000000000000000000000000000039400000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000040c8df79b1015f5bc235c9242bee499f2a5e25ad2c33b70561c2ba67118a00a1fe024dccfb65960ab91dfcd30b521fa1aa692b21ca251ec8063b2fc0a343a1b0bc",
// 				"blockNumber": "0x116fbce",
// 				"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
// 				"transactionIndex": "0x10",
// 				"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
// 				"logIndex": "0x14",
// 				"removed": false
// 			}
// 			],
// 			"transactionHash": "0x496c8a0b022c3aa582f275f1f84424c679155b120fc09e7ec8051334e653ef64",
// 			"contractAddress": "0x0000000000000000000000000000000000000000",
// 			"gasUsed": "0x75a65",
// 			"effectiveGasPrice": "0x20fb31dfe",
// 			"blockHash": "0x1377ba799fa1f4d3924297f951e3a8095735334d1ff8099aae9467947164213f",
// 			"blockNumber": "0x116fbce",
// 			"transactionIndex": "0x10"
// 		}`,
// 	}

// 	testCases := []struct {
// 		actualEventLog    string
// 		disguisedEventLog string
// 		decodeEventName   string
// 		expectedErr       error
// 	}{
// 		{
// 			actualEventLog:    eventLogs[1],
// 			disguisedEventLog: eventLogs[0],
// 			decodeEventName:   "DecodeValidatorStakeUpdateEvent",
// 			expectedErr:       errors.New("topic event mismatch"),
// 		},
// 		{
// 			actualEventLog:    eventLogs[0],
// 			disguisedEventLog: eventLogs[1],
// 			decodeEventName:   "DecodeSignerUpdateEvent",
// 			expectedErr:       errors.New("topic event mismatch"),
// 		},
// 	}

// 	receipt := ethTypes.Receipt{}

// 	for _, tc := range testCases {
// 		err := json.Unmarshal([]byte(tc.disguisedEventLog), &receipt)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}

// 		err = decodeEvent(t, tc.decodeEventName, receipt)
// 		if err == nil {
// 			t.Error(err)
// 			return
// 		}

// 		assert.EqualError(t, err, tc.expectedErr.Error())

// 		err = json.Unmarshal([]byte(tc.actualEventLog), &receipt)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}

// 		err = decodeEvent(t, tc.decodeEventName, receipt)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}

// 		assert.NoError(t, err)
// 	}
// }

// func decodeEvent(t *testing.T, eventName string, receipt ethTypes.Receipt) error {
// 	t.Helper()

// 	var err error
// 	contractCaller, err := helper.NewContractCaller()

// 	if err != nil {
// 		t.Error("Error creating contract caller")
// 	}

// 	switch eventName {
// 	case "DecodeNewHeaderBlockEvent":
// 		_, err = contractCaller.DecodeNewHeaderBlockEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeValidatorStakeUpdateEvent":
// 		_, err = contractCaller.DecodeValidatorStakeUpdateEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeSignerUpdateEvent":
// 		_, err = contractCaller.DecodeSignerUpdateEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeValidatorTopupFeesEvent":
// 		_, err = contractCaller.DecodeValidatorTopupFeesEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeValidatorJoinEvent":
// 		_, err = contractCaller.DecodeValidatorJoinEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeValidatorExitEvent":
// 		_, err = contractCaller.DecodeValidatorExitEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeStateSyncedEvent":
// 		_, err = contractCaller.DecodeStateSyncedEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeSlashedEvent":
// 		_, err = contractCaller.DecodeSlashedEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	case "DecodeUnJailedEvent":
// 		_, err = contractCaller.DecodeUnJailedEvent(receipt.Logs[0].Address, &receipt, uint64(receipt.Logs[0].Index))

// 	default:
// 		return errors.New("Unrecognized event")
// 	}

// 	return err
// }
