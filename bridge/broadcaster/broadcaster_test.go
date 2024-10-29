package broadcaster

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"

	"cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/helper"
	helperMocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	topuptypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var (
	privKey                 = secp256k1.GenPrivKey()
	cosmosPrivKey           = &cosmossecp256k1.PrivKey{Key: privKey}
	pubKey                  = privKey.PubKey()
	address                 = pubKey.Address()
	heimdallAddress         = address.String()
	heimdallAddressBytes, _ = addressCodec.NewHexCodec().StringToBytes(heimdallAddress)
	defaultBalance          = math.NewIntFromBigInt(big.NewInt(10).Exp(big.NewInt(10), big.NewInt(18), nil))
	testChainId             = "testChainId"
	dummyCometBFTNodeUrl    = "http://localhost:26657"
	dummyHeimdallServerUrl  = "https://dummy-heimdall-api-testnet.polygon.technology"
	getAccountUrl           = dummyHeimdallServerUrl + "/cosmos/auth/v1beta1/accounts/" + common.BytesToAddress(address).String()
	getAccountResponse      = fmt.Sprintf(`
	{
		"address": "%s",
		"pub_key": null,
		"account_number": "0",
		"sequence": "0"
	  }
	  `, address.String())

	getAccountUpdatedResponse = fmt.Sprintf(`
	{
		"address": "%s",
		"pub_key": null,
		"account_number": "0",
		"sequence": "1"
	  }
	  `, address.String())

	msgs = []sdk.Msg{
		checkpointTypes.NewMsgCheckpointBlock(
			heimdallAddress,
			0,
			63,
			[]byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e"),
			[]byte("0xd10b5c16c25efe0b0f5b3d75038834223934ae8c2ec2b63a62bbe42aa21e2d2d"),
			"borChainID",
		),
		milestoneTypes.NewMsgMilestoneBlock(
			heimdallAddress,
			0,
			63,
			[]byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e"),
			"testBorChainID",
			"testMilestoneID",
		),
		milestoneTypes.NewMsgMilestoneTimeout(
			heimdallAddress,
		),
		borTypes.NewMsgProposeSpanRequest(
			1,
			heimdallAddress,
			0,
			63,
			"testBorChainID",
			[]byte("randseed"),
		),
	}
)

// Parallel test - to check BroadcastToHeimdall synchronisation
func TestBroadcastToHeimdall(t *testing.T) {
	/*
		TODO HV2 - find a way to simulate txBroadcaster.CliCtx.AccountRetriever as without
		it, we cannot test BroadcastTx function.
	*/
	t.Skip()
	t.Parallel()

	viper.Set(helper.CometBFTNodeFlag, dummyCometBFTNodeUrl)
	viper.Set("log_level", "info")

	configuration := helper.GetDefaultHeimdallConfig()
	configuration.CometBFTRPCUrl = dummyCometBFTNodeUrl
	configuration.HeimdallServerURL = dummyHeimdallServerUrl
	helper.SetTestConfig(configuration)
	helper.SetTestPrivPubKey(privKey)

	mockCtrl := prepareMockData(t)
	defer mockCtrl.Finish()

	testOpts := helper.NewTestOpts(nil, testChainId)
	heimdallApp, sdkCtx, _ := createTestApp(t)

	encodingConfig := moduletestutil.MakeTestEncodingConfig()
	txConfig := encodingConfig.TxConfig

	txBroadcaster := NewTxBroadcaster(heimdallApp.AppCodec())
	txBroadcaster.CliCtx.Simulate = true
	txBroadcaster.CliCtx.TxConfig = txConfig

	testCases := []struct {
		name       string
		msg        sdk.Msg
		op         func(*app.HeimdallApp) error
		expResCode uint32
		expErr     bool
		tearDown   func(*app.HeimdallApp) error
	}{
		{
			name: "successful broadcast",
			msg:  msgs[0],

			op:         nil,
			expResCode: 0,
			expErr:     false,
		},
		{
			name: "failed broadcast (insufficient funds for fees)",
			msg:  msgs[1],
			op: func(hApp *app.HeimdallApp) error {
				acc := hApp.AccountKeeper.GetAccount(sdkCtx, sdk.AccAddress(heimdallAddressBytes))

				accountBalance := hApp.BankKeeper.GetBalance(sdkCtx, sdk.AccAddress(heimdallAddressBytes), authTypes.FeeToken)
				err := hApp.BankKeeper.SendCoinsFromAccountToModule(sdkCtx, sdk.AccAddress(heimdallAddressBytes), topuptypes.ModuleName, sdk.Coins{accountBalance})
				require.NoError(t, err)

				err = hApp.BankKeeper.BurnCoins(sdkCtx, authTypes.FeeToken, sdk.Coins{accountBalance})
				require.NoError(t, err)

				hApp.AccountKeeper.SetAccount(sdkCtx, acc)
				return nil
			},
			expResCode: 5,
			expErr:     true,
			tearDown: func(hApp *app.HeimdallApp) error {
				acc := hApp.AccountKeeper.GetAccount(sdkCtx, sdk.AccAddress(heimdallAddressBytes))

				coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: defaultBalance}}

				err := hApp.BankKeeper.SendCoinsFromAccountToModule(sdkCtx, sdk.AccAddress(heimdallAddressBytes), topuptypes.ModuleName, coins)
				require.NoError(t, err)

				err = hApp.BankKeeper.BurnCoins(sdkCtx, authTypes.FeeToken, coins)
				require.NoError(t, err)

				hApp.AccountKeeper.SetAccount(sdkCtx, acc)
				return nil
			},
		},
		{
			name: "failed broadcast (invalid sequence number)",
			msg:  msgs[2],
			op: func(hApp *app.HeimdallApp) error {
				acc := hApp.AccountKeeper.GetAccount(sdkCtx, sdk.AccAddress(heimdallAddressBytes))
				txBroadcaster.lastSeqNo = acc.GetSequence() + 1
				return nil
			},
			expResCode: 4,
			expErr:     true,
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		if tc.expErr {
			updateMockData(t)
		}
		t.Run(tc.name, func(t *testing.T) {
			if tc.op != nil {
				err := tc.op(heimdallApp)
				require.NoError(t, err)
			}
			txRes, err := txBroadcaster.BroadcastToHeimdall(tc.msg, nil, testOpts)
			require.NoError(t, err)
			require.Equal(t, tc.expResCode, txRes.Code)
			accSeq, err := heimdallApp.AccountKeeper.GetSequence(sdkCtx, sdk.AccAddress(heimdallAddressBytes))
			require.NoError(t, err)
			require.Equal(t, txBroadcaster.lastSeqNo, accSeq)

			if tc.tearDown != nil {
				err := tc.tearDown(heimdallApp)
				require.NoError(t, err)
			}
		})
	}
}

func createTestApp(t *testing.T) (*app.HeimdallApp, sdk.Context, client.Context) {
	hApp, _, _ := app.SetupApp(t, 1)
	ctx := hApp.BaseApp.NewContext(true)
	hApp.BankKeeper.SetSendEnabled(ctx, "", true)
	err := hApp.CheckpointKeeper.SetParams(ctx, checkpointTypes.DefaultParams())
	require.NoError(t, err)
	err = hApp.BorKeeper.SetParams(ctx, borTypes.DefaultParams())
	require.NoError(t, err)

	// TODO HV2 - this is unused, remove it?
	// coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: defaultBalance}}

	acc := authTypes.NewBaseAccount(heimdallAddressBytes, cosmosPrivKey.PubKey(), 0, 0)

	hApp.AccountKeeper.SetAccount(ctx, acc)

	// create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	return hApp, ctx, client.Context{}.WithCodec(cdc)
}

func prepareMockData(t *testing.T) *gomock.Controller {
	t.Helper()

	mockCtrl := gomock.NewController(t)

	mockHttpClient := helperMocks.NewMockHTTPClient(mockCtrl)
	res := prepareResponse(getAccountResponse)
	defer res.Body.Close()
	mockHttpClient.EXPECT().Get(getAccountUrl).Return(res, nil).AnyTimes()
	helper.Client = mockHttpClient
	return mockCtrl
}

func updateMockData(t *testing.T) *gomock.Controller {
	t.Helper()

	mockCtrl := gomock.NewController(t)

	mockHttpClient := helperMocks.NewMockHTTPClient(mockCtrl)
	res := prepareResponse(getAccountUpdatedResponse)
	defer res.Body.Close()
	mockHttpClient.EXPECT().Get(getAccountUrl).Return(res, nil).AnyTimes()
	helper.Client = mockHttpClient
	return mockCtrl
}

func prepareResponse(body string) *http.Response {
	return &http.Response{
		Status:           "200 OK",
		StatusCode:       200,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           nil,
		Body:             newResettableReadCloser(body),
		ContentLength:    0,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          nil,
		TLS:              nil,
	}
}

// resettableReadCloser resets the reader to the beginning of the data when Close is called.
// this is useful for reusing the response body more than once in tests.
type resettableReadCloser struct {
	data []byte
	r    io.Reader
}

func newResettableReadCloser(body string) *resettableReadCloser {
	return &resettableReadCloser{
		data: []byte(body),
		r:    bytes.NewReader([]byte(body)),
	}
}

func (r *resettableReadCloser) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *resettableReadCloser) Close() error {
	r.r = bytes.NewReader(r.data)
	return nil
}
