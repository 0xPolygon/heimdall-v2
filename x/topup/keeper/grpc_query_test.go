package keeper_test

import (
	"bytes"
	"math/big"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	hTypes "github.com/0xPolygon/heimdall-v2/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/testutil"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (s *KeeperTestSuite) TestGRPCGetTopupTxSequence_Success() {
	ctx, tk, queryClient, require, contractCaller := s.ctx, s.keeper, s.queryClient, s.Require(), &s.contractCaller

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	logIndex := uint64(simulation.RandIntBetween(r, 0, 100))
	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
	sequence := new(big.Int).Mul(txReceipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	err := tk.SetTopupSequence(ctx, sequence.String())
	require.NoError(err)
	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   TxHash,
		LogIndex: logIndex,
	}

	res, err := queryClient.GetTopupTxSequence(ctx, req)
	require.NoError(err)
	require.NotNil(res.Sequence)
	require.Equal(sequence.String(), res.Sequence)
}

func (s *KeeperTestSuite) TestGRPCGetTopupTxSequence_InvalidTxHash() {
	ctx, tk, queryClient, require, contractCaller := s.ctx, s.keeper, s.queryClient, s.Require(), &s.contractCaller

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	logIndex := uint64(simulation.RandIntBetween(r, 0, 100))
	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
	sequence := new(big.Int).Mul(txReceipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	err := tk.SetTopupSequence(ctx, sequence.String())
	require.NoError(err)
	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   "",
		LogIndex: logIndex,
	}

	res, err := queryClient.GetTopupTxSequence(ctx, req)
	require.Error(err)
	require.Nil(res)
	require.Contains(err.Error(), "invalid tx hash")
}

func (s *KeeperTestSuite) TestGRPCGetTopupTxSequence_NotFound() {
	ctx, tk, queryClient, require, contractCaller := s.ctx, s.keeper, s.queryClient, s.Require(), &s.contractCaller

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	logIndex := r.Uint64()
	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   TxHash,
		LogIndex: logIndex,
	}

	res, err := queryClient.GetTopupTxSequence(ctx, req)
	require.Error(err)
	require.Nil(res)
}

func (s *KeeperTestSuite) TestGRPCIsTopupTxOld_IsOld() {
	ctx, tk, queryClient, require, contractCaller := s.ctx, s.keeper, s.queryClient, s.Require(), &s.contractCaller
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	logIndex := r.Uint64()
	blockNumber := r.Uint64()
	blockN := new(big.Int).SetUint64(blockNumber)
	sequence := new(big.Int).Mul(blockN, big.NewInt(types.DefaultLogIndexUnit))
	txReceipt := &ethTypes.Receipt{BlockNumber: blockN}
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
	err := tk.SetTopupSequence(ctx, sequence.String())
	require.NoError(err)
	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   TxHash,
		LogIndex: logIndex,
	}

	res, err := queryClient.IsTopupTxOld(ctx, req)
	require.NoError(err)
	require.True(res.IsOld)
}

func (s *KeeperTestSuite) TestGRPCIsTopupTxOld_InvalidTxHash() {
	ctx, tk, queryClient, require, contractCaller := s.ctx, s.keeper, s.queryClient, s.Require(), &s.contractCaller
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	logIndex := r.Uint64()
	blockNumber := r.Uint64()
	blockN := new(big.Int).SetUint64(blockNumber)
	sequence := new(big.Int).Mul(blockN, big.NewInt(types.DefaultLogIndexUnit))
	txReceipt := &ethTypes.Receipt{BlockNumber: blockN}
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
	err := tk.SetTopupSequence(ctx, sequence.String())
	require.NoError(err)
	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   "",
		LogIndex: logIndex,
	}

	res, err := queryClient.IsTopupTxOld(ctx, req)
	require.Error(err)
	require.Nil(res)
	require.Contains(err.Error(), "invalid tx hash")
}

func (s *KeeperTestSuite) TestGRPCIsTopupTxOld_IsNotOld() {
	ctx, tk, queryClient, require, contractCaller := s.ctx, s.keeper, s.queryClient, s.Require(), &s.contractCaller
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	logIndex := r.Uint64()
	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   TxHash,
		LogIndex: logIndex,
	}

	res, err := queryClient.IsTopupTxOld(ctx, req)
	require.NoError(err)
	require.False(res.IsOld)
}

func (s *KeeperTestSuite) TestGRPCGetDividendAccountByAddress_Success() {
	ctx, tk, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      util.FormatAddress(AccountHash),
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)
	ok, err := tk.HasDividendAccount(ctx, dividendAccount.User)
	require.NoError(err)
	require.Equal(ok, true)

	req := &types.QueryDividendAccountRequest{
		Address: AccountHash,
	}

	res, err := queryClient.GetDividendAccountByAddress(ctx, req)
	require.NoError(err)
	require.Equal(res.DividendAccount, dividendAccount)
}

func (s *KeeperTestSuite) TestGRPCGetDividendAccountByAddress_NotFound() {
	ctx, tk, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	ok, err := tk.HasDividendAccount(ctx, dividendAccount.User)
	require.NoError(err)
	require.Equal(ok, false)

	req := &types.QueryDividendAccountRequest{
		Address: AccountHash,
	}

	res, err := queryClient.GetDividendAccountByAddress(ctx, req)
	require.Error(err)
	require.Contains(err.Error(), "not found")
	require.Empty(res)
}

func (s *KeeperTestSuite) TestGRPCGetDividendAccountRootHash_Success() {
	ctx, tk, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)

	req := &types.QueryDividendAccountRootHashRequest{}

	res, err := queryClient.GetDividendAccountRootHash(ctx, req)
	require.NoError(err)
	require.NotNil(res.AccountRootHash)
	require.NotEmpty(res.AccountRootHash)
}

func (s *KeeperTestSuite) TestGRPCGetDividendAccountRootHash_NotFound() {
	ctx, queryClient, require := s.ctx, s.queryClient, s.Require()

	req := &types.QueryDividendAccountRootHashRequest{}

	res, err := queryClient.GetDividendAccountRootHash(ctx, req)
	require.Error(err)
	require.Contains(err.Error(), "cannot construct tree with no content")
	require.Nil(res)
}

func (s *KeeperTestSuite) TestGRPCVerifyAccountProofByAddress_Success() {
	ctx, tk, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)

	// Retrieve all dividend accounts to create the proof
	dividendAccounts, err := tk.GetAllDividendAccounts(ctx)
	require.NoError(err)

	// Dynamically generate the proof for the given account
	proofBytes, _, err := hTypes.GetAccountProof(dividendAccounts, AccountHash)
	require.NoError(err)
	AccountHashProof := common.Bytes2Hex(proofBytes)

	req := &types.QueryVerifyAccountProofRequest{
		Address: AccountHash,
		Proof:   AccountHashProof,
	}
	res, err := queryClient.VerifyAccountProofByAddress(ctx, req)
	require.NoError(err)
	require.True(res.IsVerified)
}

func (s *KeeperTestSuite) TestGRPCVerifyAccountProofByAddress_InvalidAddress() {
	ctx, tk, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)

	// Retrieve all dividend accounts to create the proof
	dividendAccounts, err := tk.GetAllDividendAccounts(ctx)
	require.NoError(err)

	// Dynamically generate the proof for the given account
	proofBytes, _, err := hTypes.GetAccountProof(dividendAccounts, AccountHash)
	require.NoError(err)
	AccountHashProof := common.Bytes2Hex(proofBytes)

	req := &types.QueryVerifyAccountProofRequest{
		Address: AccountHash + "invalid",
		Proof:   AccountHashProof,
	}
	res, err := queryClient.VerifyAccountProofByAddress(ctx, req)
	require.Error(err)
	require.Nil(res)
	require.Contains(err.Error(), "invalid address")
}

func (s *KeeperTestSuite) TestGRPCVerifyAccountProofByAddress_InvalidProof_Empty() {
	ctx, tk, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)

	req := &types.QueryVerifyAccountProofRequest{
		Address: AccountHash,
		Proof:   "",
	}
	res, err := queryClient.VerifyAccountProofByAddress(ctx, req)
	require.Error(err)
	require.Nil(res)
	require.Contains(err.Error(), "proof is empty")
}

func (s *KeeperTestSuite) TestGRPCVerifyAccountProofByAddress_InvalidProof_NotMultipleOf32() {
	ctx, _, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	badProof := common.Bytes2Hex(bytes.Repeat([]byte{0x01}, 33)) // 33 bytes

	req := &types.QueryVerifyAccountProofRequest{
		Address: AccountHash,
		Proof:   "0x" + badProof,
	}
	res, err := queryClient.VerifyAccountProofByAddress(ctx, req)
	require.Error(err)
	require.Nil(res)
	require.Contains(err.Error(), "proof length must be a multiple of 32 bytes")
}

func (s *KeeperTestSuite) TestGRPCVerifyAccountProofByAddress_InvalidProof_TooLong() {
	ctx, _, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	// Generate a proof longer than MaxProofLength and aligned to 32 bytes
	tooLongSize := ((util.MaxProofLength / 32) + 1) * 32
	tooLongProof := common.Bytes2Hex(bytes.Repeat([]byte{0x01}, tooLongSize))

	req := &types.QueryVerifyAccountProofRequest{
		Address: AccountHash,
		Proof:   "0x" + tooLongProof,
	}

	res, err := queryClient.VerifyAccountProofByAddress(ctx, req)
	require.Error(err)
	require.Nil(res)
	require.Contains(err.Error(), "proof exceeds maximum allowed size of")
}

func (s *KeeperTestSuite) TestGRPCGetAccountProofByAddress_Success() {
	ctx, tk, queryClient, require, contractCaller := s.ctx, s.keeper, s.queryClient, s.Require(), &s.contractCaller

	var accountRoot [32]byte
	stakingInfo := &stakinginfo.Stakinginfo{}
	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)
	dividendAccounts, err := tk.GetAllDividendAccounts(ctx)
	require.NoError(err)
	accRoot, err := hTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(err)
	copy(accountRoot[:], accRoot)

	contractCaller.On("GetStakingInfoInstance", mock.Anything).Return(stakingInfo, nil)
	contractCaller.On("CurrentAccountStateRoot", stakingInfo).Return(accountRoot, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryAccountProofRequest{
		Address: AccountHash,
	}

	res, err := queryClient.GetAccountProofByAddress(ctx, req)
	require.NoError(err)
	require.NotNil(res.Proof)
}
