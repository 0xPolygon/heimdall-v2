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

func (suite *KeeperTestSuite) TestGRPCGetTopupTxSequence_Success() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	hash := hTypes.TxHash{Hash: []byte(TxHash)}
	logIndex := uint64(simulation.RandIntBetween(r1, 0, 100))
	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
	sequence := new(big.Int).Mul(txReceipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	err := tk.SetTopupSequence(ctx, sequence.String())
	require.NoError(err)
	suite.contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   hash.String(),
		LogIndex: logIndex,
	}

	res, err := queryClient.GetTopupTxSequence(ctx, req)
	require.NoError(err)
	require.NotNil(res.Sequence)
	require.Equal(sequence.String(), res.Sequence)
}

func (suite *KeeperTestSuite) TestGRPCGetTopupTxSequence_NotFound() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	logIndex := r1.Uint64()
	hash := hTypes.TxHash{Hash: []byte(TxHash)}
	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

	suite.contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   hash.String(),
		LogIndex: logIndex,
	}

	res, err := queryClient.GetTopupTxSequence(ctx, req)
	require.Error(err)
	require.Nil(res)
}

func (suite *KeeperTestSuite) TestGRPCIsTopupTxOld_IsOld() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	logIndex := r1.Uint64()
	blockNumber := r1.Uint64()
	hash := hTypes.TxHash{Hash: []byte(TxHash)}
	blockN := new(big.Int).SetUint64(blockNumber)
	sequence := new(big.Int).Mul(blockN, big.NewInt(types.DefaultLogIndexUnit))
	txReceipt := &ethTypes.Receipt{BlockNumber: blockN}
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
	err := tk.SetTopupSequence(ctx, sequence.String())
	require.NoError(err)
	suite.contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   hash.String(),
		LogIndex: logIndex,
	}

	res, err := queryClient.IsTopupTxOld(ctx, req)
	require.NoError(err)
	require.True(res.IsOld)
}

func (suite *KeeperTestSuite) TestGRPCIsTopupTxOld_IsNotOld() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	logIndex := r1.Uint64()
	hash := hTypes.TxHash{Hash: []byte(TxHash)}
	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

	suite.contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryTopupSequenceRequest{
		TxHash:   hash.String(),
		LogIndex: logIndex,
	}

	res, err := queryClient.IsTopupTxOld(ctx, req)
	require.NoError(err)
	require.False(res.IsOld)
}

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountByAddress_Success() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

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

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountByAddress_NotFound() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

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

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountRootHash_Success() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)

	req := &types.QueryDividendAccountRootHashRequest{}

	_, err = queryClient.GetDividendAccountRootHash(ctx, req)
	require.NoError(err)
	// TODO HV2: enable this when `GetDividendAccountRootHash` is fully functional in grpc_query.go
	// require.NotNil(res.AccountRootHash)
	// require.NotEmpty(res.AccountRootHash)
}

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountRootHash_NotFound() {
	ctx, queryClient, require := suite.ctx, suite.queryClient, suite.Require()

	req := &types.QueryDividendAccountRootHashRequest{}

	res, _ := queryClient.GetDividendAccountRootHash(ctx, req)
	// TODO HV2: enable this when `GetDividendAccountRootHash` is fully functional in grpc_query.go
	// require.Error(err)
	// require.Contains(err.Error(), "not found")
	require.Empty(res.AccountRootHash)
}

func (suite *KeeperTestSuite) TestGRPCVerifyAccountProof_Success() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

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
	_, err = queryClient.VerifyAccountProof(ctx, req)
	require.NoError(err)
	// TODO HV2: enable this when `VerifyAccountProof` is fully functional in grpc_query.go
	// require.True(res.IsVerified)
}

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountProof_Success() {
	ctx, tk, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

	var accountRoot [32]byte
	stakingInfo := &stakinginfo.Stakinginfo{}
	dividendAccount := hTypes.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)
	// TODO HV2: replace `_` with `dividendAccounts` when checkpoint is implemented in heimdall-v2
	_, err = tk.GetAllDividendAccounts(ctx)
	require.NoError(err)
	// TODO HV2: enable this when checkpoint is implemented in heimdall-v2 and deleted the mocked `accRoot`
	// accRoot, err := checkpointTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(err)
	accRoot := []byte("accRoot")
	copy(accountRoot[:], accRoot)

	suite.contractCaller.On("GetStakingInfoInstance", mock.Anything).Return(stakingInfo, nil)
	suite.contractCaller.On("CurrentAccountStateRoot", stakingInfo).Return(accountRoot, nil)
	tk.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

	req := &types.QueryAccountProofRequest{
		Address: AccountHash,
	}

	_, err = queryClient.GetAccountProof(ctx, req)
	require.NoError(err)
	// TODO HV2: enable this when `GetAccountProof` is fully functional in grpc_query.go
	// require.NotNil(res.Proof)
}
