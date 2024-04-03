package keeper_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (suite *KeeperTestSuite) TestGRPCGetTopupTxSequence() {
	queryClient := suite.queryClient
	// TODO HV2: enable when contractCaller and chainManager are implemented
	// suite.contractCaller = mocks.IContractCaller{}
	// suite.chainParams = suite.app.ChainKeeper.GetParams(suite.ctx)

	var req *types.QueryTopupSequenceRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res *types.QueryTopupSequenceResponse)
	}{
		{
			"success",
			func() {
				s1 := rand.NewSource(time.Now().UnixNano())
				r1 := rand.New(s1)
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}
				logIndex := uint64(simulation.RandIntBetween(r1, 0, 100))
				txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
				sequence := new(big.Int).Mul(txReceipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
				err := suite.app.TopupKeeper.SetTopupSequence(suite.ctx, sequence.String())
				suite.Require().NoError(err)
				// TODO HV2: enable when contractCaller is implemented
				// suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)

				req = &types.QueryTopupSequenceRequest{
					TxHash:   hash.String(),
					LogIndex: logIndex,
				}
			},
			true,
			"",
			func(res *types.QueryTopupSequenceResponse) {
				txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
				sequence := new(big.Int).Mul(txReceipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
				suite.Require().NotNil(res.Sequence)
				suite.Require().Equal(sequence.String(), res.Sequence)
			},
		},
		{
			"not found",
			func() {
				s1 := rand.NewSource(time.Now().UnixNano())
				r1 := rand.New(s1)
				logIndex := r1.Uint64()
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				req = &types.QueryTopupSequenceRequest{
					TxHash:   hash.String(),
					LogIndex: logIndex,
				}
			},
			false,
			"not found",
			func(res *types.QueryTopupSequenceResponse) {
			},
		},
		{
			"empty request",
			func() {
				req = &types.QueryTopupSequenceRequest{}
			},
			false,
			"empty request",
			func(res *types.QueryTopupSequenceResponse) {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res, err := queryClient.GetTopupTxSequence(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			}

			tc.posttests(res)
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCIsTopupTxOld() {
	queryClient := suite.queryClient
	var req *types.QueryTopupSequenceRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res *types.QueryIsTopupTxOldResponse)
	}{
		{
			"oldTx",
			func() {
				s1 := rand.NewSource(time.Now().UnixNano())
				r1 := rand.New(s1)
				logIndex := r1.Uint64()
				blockNumber := r1.Uint64()
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				blockN := new(big.Int).SetUint64(blockNumber)
				sequence := new(big.Int).Mul(blockN, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
				_ = suite.app.TopupKeeper.SetTopupSequence(suite.ctx, sequence.String())

				req = &types.QueryTopupSequenceRequest{
					TxHash:   hash.String(),
					LogIndex: logIndex,
				}
			},
			true,
			"",
			func(res *types.QueryIsTopupTxOldResponse) {
				suite.Require().True(res.IsOld)
			},
		},
		{
			"notOldTx",
			func() {
				s1 := rand.NewSource(time.Now().UnixNano())
				r1 := rand.New(s1)
				logIndex := r1.Uint64()
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}
				req = &types.QueryTopupSequenceRequest{
					TxHash:   hash.String(),
					LogIndex: logIndex,
				}
			},
			false,
			"not found",
			func(res *types.QueryIsTopupTxOldResponse) {
			},
		},
		{
			"empty request",
			func() {
				req = &types.QueryTopupSequenceRequest{}
			},
			false,
			"empty request",
			func(res *types.QueryIsTopupTxOldResponse) {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res, err := queryClient.IsTopupTxOld(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
				suite.Require().Nil(res)
			}

			tc.posttests(res)
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountByAddress() {
	queryClient := suite.queryClient

	var req *types.QueryDividendAccountRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res *types.QueryDividendAccountResponse)
	}{
		{
			"success",
			func() {
				accountHash, err := address.HexCodec{}.StringToBytes("0x000000000000000000000000000000000000dEaD")
				suite.Require().NoError(err)
				hash := hTypes.HeimdallHash{Hash: accountHash}
				dividendAccount := hTypes.DividendAccount{
					User:      hash.String(),
					FeeAmount: big.NewInt(0).String(),
				}
				err = suite.app.TopupKeeper.SetDividendAccount(suite.ctx, dividendAccount)
				require.NoError(suite.T(), err)
				req = &types.QueryDividendAccountRequest{
					Address: hash.String(),
				}
			},
			true,
			"",
			func(res *types.QueryDividendAccountResponse) {
				accountHash, err := address.HexCodec{}.StringToBytes("0x000000000000000000000000000000000000dEaD")
				suite.Require().NoError(err)
				hash := hTypes.HeimdallHash{Hash: accountHash}
				dividendAccount := hTypes.DividendAccount{
					User:      hash.String(),
					FeeAmount: big.NewInt(0).String(),
				}
				suite.Require().NotNil(res.DividendAccount)
				suite.Require().Equal(dividendAccount, res.DividendAccount)
			},
		},
		{
			"not found",
			func() {
				accountHash, err := address.HexCodec{}.StringToBytes("0x000000000000000000000000000000000000dEaD")
				suite.Require().NoError(err)
				hash := hTypes.HeimdallHash{Hash: accountHash}
				req = &types.QueryDividendAccountRequest{
					Address: hash.String(),
				}
			},
			false,
			"not found",
			func(res *types.QueryDividendAccountResponse) {
			},
		},
		{
			"empty request",
			func() {
				req = &types.QueryDividendAccountRequest{}
			},
			false,
			"empty request",
			func(res *types.QueryDividendAccountResponse) {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res, err := queryClient.GetDividendAccountByAddress(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
				suite.Require().Nil(res)
			}

			tc.posttests(res)
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountRootHash() {
	queryClient := suite.queryClient

	var req *types.QueryDividendAccountRootHashRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res *types.QueryDividendAccountRootHashResponse)
	}{
		{
			"success",
			func() {
				accountHash, err := address.HexCodec{}.StringToBytes("0x000000000000000000000000000000000000dEaD")
				suite.Require().NoError(err)
				hash := hTypes.HeimdallHash{Hash: accountHash}
				dividendAccount := hTypes.DividendAccount{
					User:      hash.String(),
					FeeAmount: big.NewInt(0).String(),
				}
				err = suite.app.TopupKeeper.SetDividendAccount(suite.ctx, dividendAccount)
				require.NoError(suite.T(), err)
				req = &types.QueryDividendAccountRootHashRequest{}
			},
			true,
			"",
			func(res *types.QueryDividendAccountRootHashResponse) {
				suite.Require().NotNil(res.AccountRootHash)
			},
		},
		{
			"no accounts set",
			func() {
				req = &types.QueryDividendAccountRootHashRequest{}
			},
			false,
			// TODO HV2: expect a particular message when checkpoint is implemented?
			"",
			func(res *types.QueryDividendAccountRootHashResponse) {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res, err := queryClient.GetDividendAccountRootHash(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
				suite.Require().Nil(res)
			}

			tc.posttests(res)
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCVerifyAccountProof() {
	queryClient := suite.queryClient

	var req *types.QueryVerifyAccountProofRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res *types.QueryVerifyAccountProofResponse)
	}{
		{
			"success",
			func() {
				accountHash, err := address.HexCodec{}.StringToBytes("0x000000000000000000000000000000000000dEaD")
				suite.Require().NoError(err)
				hash := hTypes.HeimdallHash{Hash: accountHash}
				dividendAccount := hTypes.DividendAccount{
					User:      hash.String(),
					FeeAmount: big.NewInt(0).String(),
				}
				err = suite.app.TopupKeeper.SetDividendAccount(suite.ctx, dividendAccount)
				require.NoError(suite.T(), err)
				req = &types.QueryVerifyAccountProofRequest{
					Address: hash.String(),
					Proof:   "",
				}
			},
			true,
			"",
			func(res *types.QueryVerifyAccountProofResponse) {
				suite.Require().True(res.IsVerified)
			},
		},
		{
			"empty request",
			func() {
				req = &types.QueryVerifyAccountProofRequest{}
			},
			false,
			"empty request",
			func(res *types.QueryVerifyAccountProofResponse) {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res, err := queryClient.VerifyAccountProof(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
				suite.Require().Nil(res)
			}

			tc.posttests(res)
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCGetDividendAccountProof() {
	queryClient := suite.queryClient

	var req *types.QueryDividendAccountProofRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res *types.QueryDividendAccountProofResponse)
	}{
		{
			"success",
			func() {
				var accountRoot [32]byte
				// TODO HV2: enable this when contractCaller is implemented in heimdall-v2
				// stakingInfo := &stakinginfo.Stakinginfo{}
				accountHash, err := address.HexCodec{}.StringToBytes("0x000000000000000000000000000000000000dEaD")
				suite.Require().NoError(err)
				hash := hTypes.HeimdallHash{Hash: accountHash}
				dividendAccount := hTypes.DividendAccount{
					User:      hash.String(),
					FeeAmount: big.NewInt(0).String(),
				}
				err = suite.app.TopupKeeper.SetDividendAccount(suite.ctx, dividendAccount)
				require.NoError(suite.T(), err)
				// TODO HV2: replace _ with dividendAccounts when checkpoint is implemented in heimdall-v2
				_, err = suite.app.TopupKeeper.GetAllDividendAccounts(suite.ctx)
				require.NoError(suite.T(), err)
				// TODO HV2: enable this when checkpoint is implemented in heimdall-v2 and deleted the fake accRoot
				// accRoot, err := checkpointTypes.GetAccountRootHash(dividendAccounts)
				require.NoError(suite.T(), err)
				accRoot := []byte("accRoot")
				copy(accountRoot[:], accRoot)

				// TODO HV2: enable this when contractCaller is implemented in heimdall-v2
				// suite.contractCaller.On("GetStakingInfoInstance", mock.Anything).Return(stakingInfo, nil)
				// suite.contractCaller.On("CurrentAccountStateRoot", stakingInfo).Return(accountRoot, nil)

				req = &types.QueryDividendAccountProofRequest{
					Address: hash.String(),
				}
			},
			true,
			"",
			func(res *types.QueryDividendAccountProofResponse) {
				suite.Require().NotNil(res.Proof)
			},
		},
		{
			"empty request",
			func() {
				req = &types.QueryDividendAccountProofRequest{}
			},
			false,
			"empty request",
			func(res *types.QueryDividendAccountProofResponse) {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res, err := queryClient.GetDividendAccountProof(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
				suite.Require().Nil(res)
			}

			tc.posttests(res)
		})
	}
}
