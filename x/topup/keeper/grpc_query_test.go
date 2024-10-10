package keeper_test

import (
	"math/big"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/simulation"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

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
		User:      AccountHash,
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

func (s *KeeperTestSuite) TestGRPCVerifyAccountProof_Success() {
	ctx, tk, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)

	// TODO HV2: double check this
	AccountHashProof := "44ad89ba62b98ff34f51403ac22759b55759460c0bb5521eb4b6ee3cff49cf83"

	req := &types.QueryVerifyAccountProofRequest{
		Address: AccountHash,
		Proof:   AccountHashProof,
	}
	res, err := queryClient.VerifyAccountProofByAddress(ctx, req)
	require.NoError(err)
	require.True(res.IsVerified)
}

func (s *KeeperTestSuite) TestGRPCGetDividendAccountProof_Success() {
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
