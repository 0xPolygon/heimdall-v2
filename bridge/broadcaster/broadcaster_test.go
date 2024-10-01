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
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var (
	privKey                 = secp256k1.GenPrivKey()
	pubKey                  = privKey.PubKey()
	address                 = pubKey.Address()
	heimdallAddress         = address.String()
	heimdallAddressBytes, _ = addressCodec.NewHexCodec().StringToBytes(heimdallAddress)
	defaultBalance          = math.NewIntFromBigInt(big.NewInt(10).Exp(big.NewInt(10), big.NewInt(18), nil))
	testChainId             = "testChainId"
	dummyCometBFTNodeUrl    = "http://localhost:26657"
	dummyHeimdallServerUrl  = "https://dummy-heimdall-api-testnet.polygon.technology"
	getAccountUrl           = dummyHeimdallServerUrl + "/auth/accounts/" + address.String()
	getAccountResponse      = fmt.Sprintf(`
	{
		"height": "11384869",
		"result": {
		  "type": "auth/Account",
		  "value": {
			"address": "0x%s",
			"coins": [
			  {
				"denom": "matic",
				"amount": "10000000000000000000"
			  }
			],
			"public_key": {
				"type": "cometbft/PubKeySecp256k1",
				"value": "BE/WIL+R3P+8YlGBfxqPdb+jWlWdAiocPOBYNXoXqYOlQ0+QiJudDIMLhDqovssOvS9REFaUYn6pXE0YGD3nb5k="
			  },
			"account_number": "0",
			"sequence": "0"
		  }
		}
	  }
	  `, address.String())

	getAccountUpdatedResponse = fmt.Sprintf(`
	{
		"height": "11384869",
		"result": {
		  "type": "auth/Account",
		  "value": {
			"address": "0x%s",
			"coins": [
			  {
				"denom": "matic",
				"amount": "10000000000000000000"
			  }
			],
			"public_key": {
				"type": "cometbft/PubKeySecp256k1",
				"value": "BE/WIL+R3P+8YlGBfxqPdb+jWlWdAiocPOBYNXoXqYOlQ0+QiJudDIMLhDqovssOvS9REFaUYn6pXE0YGD3nb5k="
			  },
			"account_number": "0",
			"sequence": "1"
		  }
		}
	  }
	  `, address.String())

	msgs = []sdk.Msg{
		checkpointTypes.NewMsgCheckpointBlock(
			heimdallAddress,
			0,
			63,
			hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")},
			hmTypes.HeimdallHash{Hash: []byte("0xd10b5c16c25efe0b0f5b3d75038834223934ae8c2ec2b63a62bbe42aa21e2d2d")},
			"borChainID",
		),
		milestoneTypes.NewMsgMilestoneBlock(
			heimdallAddress,
			0,
			63,
			hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")},
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

	// TODO HV2 - this says HeimdallApp doesnot implement abci.Application
	// testOpts.SetApplication(heimdallApp)

	txBroadcaster := NewTxBroadcaster(heimdallApp.AppCodec())
	txBroadcaster.CliCtx.Simulate = true

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
				// TODO HV2 - not sure how to set balance to 0
				/*
					// reduce account balance to 0
					if err := acc.SetCoins(sdk.Coins{}); err != nil {
						return err
					}
				*/
				hApp.AccountKeeper.SetAccount(sdkCtx, acc)
				return nil
			},
			expResCode: 5,
			expErr:     true,
			tearDown: func(hApp *app.HeimdallApp) error {
				acc := hApp.AccountKeeper.GetAccount(sdkCtx, sdk.AccAddress(heimdallAddressBytes))
				// TODO HV2 - not sure how to reset account balance
				/*
					// reset account balance
					if err := acc.SetCoins(sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: defaultBalance}}); err != nil {
						return err
					}
				*/
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
	hApp.CheckpointKeeper.SetParams(ctx, checkpointTypes.DefaultParams())
	hApp.BorKeeper.SetParams(ctx, borTypes.DefaultParams())

	// TODO HV2 - this is unused, remove it?
	// coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: defaultBalance}}

	/*
		TODO HV2 - resolve the issue with this
		impossible type assertion: no type can implement both
		github.com/cometbft/cometbft/crypto.PubKey and
		github.com/cosmos/cosmos-sdk/crypto/types.PubKey
		(conflicting types for Equals method)
	*/
	acc := authTypes.NewBaseAccount(sdk.AccAddress(heimdallAddressBytes), pubKey.(cryptotypes.PubKey), 0, 0)

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
