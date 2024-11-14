package broadcaster

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"testing"

	"cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/app"
	addressUtil "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/helper"
	helperMocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmosTestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	tx "github.com/cosmos/cosmos-sdk/types/tx"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var (
	privKey                 = secp256k1.GenPrivKey()
	cosmosPrivKey           = &cosmossecp256k1.PrivKey{Key: privKey}
	pubKey                  = privKey.PubKey()
	address                 = pubKey.Address()
	heimdallAddress         = addressUtil.FormatAddress(common.BytesToAddress(address).String())
	heimdallAddressBytes, _ = addressCodec.NewHexCodec().StringToBytes(heimdallAddress)
	defaultBalance          = math.NewIntFromBigInt(big.NewInt(10).Exp(big.NewInt(10), big.NewInt(18), nil))
	testChainId             = "testChainId"
	dummyCometBFTNodeUrl    = "http://localhost:26657"
	dummyHeimdallServerUrl  = "https://dummy-heimdall-api-testnet.polygon.technology"
	getAccountUrl           = dummyHeimdallServerUrl + "/cosmos/auth/v1beta1/accounts/" + heimdallAddress
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

func TestBroadcastToHeimdall(t *testing.T) {
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
	txBroadcaster.CliCtx.FromAddress = sdk.AccAddress(heimdallAddressBytes)
	txBroadcaster.CliCtx.ChainID = testChainId
	txBroadcaster.CliCtx.Client = cosmosTestutil.NewMockCometRPC(abci.ResponseQuery{})
	// nolint:staticcheck
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer()), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	txBroadcaster.CliCtx.GRPCClient = conn

	mockAccountRetriever := &mockAccountRetriever{
		AccountKeeper: heimdallApp.AccountKeeper,
		Ctx:           sdkCtx,
	}
	txBroadcaster.CliCtx.AccountRetriever = mockAccountRetriever

	updateMockData(t)

	testCases := []struct {
		name       string
		msg        sdk.Msg
		expResCode uint32
		expErr     bool
		expLastSeq uint64
	}{
		{
			name: "successful broadcast",
			msg:  msgs[0],

			expResCode: 0,
			expErr:     false,
			expLastSeq: 1,
		},
		{
			name: "failed broadcast",
			msg:  msgs[0],

			expResCode: 1,
			expErr:     true,
			expLastSeq: 1,
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expErr {
				shouldFailSimulate = true
			} else {
				shouldFailSimulate = false
			}

			txRes, err := txBroadcaster.BroadcastToHeimdall(tc.msg, nil, testOpts)
			if tc.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expResCode, txRes.Code)
			require.Equal(t, tc.expLastSeq, txBroadcaster.lastSeqNo)
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

	acc := authTypes.NewBaseAccount(sdk.AccAddress(heimdallAddressBytes), cosmosPrivKey.PubKey(), 1337, 0)
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

type mockAccountRetriever struct {
	AccountKeeper authkeeper.AccountKeeper
	Ctx           sdk.Context
}

func (mar *mockAccountRetriever) GetAccount(clientCtx client.Context, addr sdk.AccAddress) (client.Account, error) {
	acc := mar.AccountKeeper.GetAccount(mar.Ctx, addr)
	if acc == nil {
		return nil, fmt.Errorf("account not found")
	}
	return acc, nil
}

func (mar *mockAccountRetriever) GetAccountWithHeight(clientCtx client.Context, addr sdk.AccAddress) (client.Account, int64, error) {
	acc := mar.AccountKeeper.GetAccount(mar.Ctx, addr)
	if acc == nil {
		return nil, 0, fmt.Errorf("account not found")
	}
	return acc, mar.Ctx.BlockHeight(), nil
}

func (mar *mockAccountRetriever) EnsureExists(clientCtx client.Context, addr sdk.AccAddress) error {
	acc := mar.AccountKeeper.GetAccount(mar.Ctx, addr)
	if acc == nil {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (mar *mockAccountRetriever) GetAccountNumberSequence(clientCtx client.Context, addr sdk.AccAddress) (uint64, uint64, error) {
	acc := mar.AccountKeeper.GetAccount(mar.Ctx, addr)
	if acc == nil {
		return 0, 0, fmt.Errorf("account not found")
	}
	return acc.GetAccountNumber(), acc.GetSequence(), nil
}

const bufSize = 1024 * 1024

func dialer() func(context.Context, string) (net.Conn, error) {
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()

	mockTxService := &mockTxServiceServer{}
	tx.RegisterServiceServer(srv, mockTxService)

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	return func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}
}

type mockTxServiceServer struct {
	tx.UnimplementedServiceServer

	BroadcastTxFunc func(ctx context.Context, req *tx.BroadcastTxRequest) (*tx.BroadcastTxResponse, error)
}

func (m *mockTxServiceServer) Simulate(ctx context.Context, req *tx.SimulateRequest) (*tx.SimulateResponse, error) {
	if shouldFailSimulate {
		return nil, fmt.Errorf("simulate failed")
	}
	return &tx.SimulateResponse{
		GasInfo: &sdk.GasInfo{
			GasWanted: 200000,
			GasUsed:   150000,
		},
		Result: &sdk.Result{
			Data: []byte("simulation data"),
			Log:  "simulation log",
		},
	}, nil
}

var shouldFailSimulate bool
