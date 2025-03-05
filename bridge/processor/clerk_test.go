package processor

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"testing"
	"time"

	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/listener"
	"github.com/0xPolygon/heimdall-v2/bridge/queue"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	helperMocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
)

func BenchmarkSendStateSyncedToHeimdall(b *testing.B) {
	b.Skip("to be enabled")
	b.ReportAllocs()
	b.ResetTimer()
	b.StopTimer()

	for i := 0; i < b.N; i++ {
		func() {
			b.Logf("Executing iteration '%d' out of '%d'", i, b.N)

			// given
			mockCtrl := prepareMockData(b)
			defer mockCtrl.Finish()

			cp, err := prepareClerkProcessor()
			if err != nil {
				b.Fatal("Error initializing test clerk processor")
			}

			dlb, err := prepareDummyLogBytes()
			if err != nil {
				b.Fatal("Error creating test data")
			}

			// when
			b.StartTimer()

			if err = cp.sendStateSyncedToHeimdall("StateSynced", dlb.String()); err != nil {
				b.Fatal(err)
			}

			b.StopTimer()

			b.Log("StateSynced sent to heimdall successfully")
		}()
	}
}

func BenchmarkIsOldTx(b *testing.B) {
	b.Skip("to be enabled")
	b.ReportAllocs()
	b.ResetTimer()
	b.StopTimer()

	for i := 0; i < b.N; i++ {
		func() {
			b.Logf("Executing iteration '%d' out of '%d'", i, b.N)

			// given
			mockCtrl := prepareMockData(b)
			mockCtrl.Finish()

			cp, err := prepareClerkProcessor()
			if err != nil {
				b.Fatal("Error initializing test clerk processor")
			}

			// when
			b.StartTimer()

			status, err := cp.isOldTx(
				cp.cliCtx, "0x6d428739815d7c84cf89db055158861b089e0fd649676a0243a2a2d204c1d854",
				0, util.ClerkEvent, nil)
			if err != nil {
				b.Fatal(err)
			}

			b.StopTimer()

			b.Logf("isTxOld tested successfully with result: '%t'", status)
		}()
	}
}

func BenchmarkSendTaskWithDelay(b *testing.B) {
	ts := make([]time.Duration, 0, b.N)
	for i := 0; i < b.N; i++ {
		ts = append(ts, time.Duration(generateRandNumber(60)))
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.StopTimer()

	for i := 0; i < b.N; i++ {
		func() {
			b.Logf("Executing iteration '%d' out of '%d'", i, b.N)

			// given
			mockCtrl := prepareMockData(b)
			defer mockCtrl.Finish()

			logs, err := prepareDummyLogBytes()
			if err != nil {
				b.Fatal("Error creating test data")
			}

			rcl, stopFn, err := prepareRootChainListener()
			if err != nil {
				b.Fatal("Error initializing test listener")
			}

			defer stopFn()

			// when
			b.StartTimer()
			// This will trigger error="Set state pending error: dial tcp 127.0.0.1:6379: connect: connection refused"
			// it's fine as long as we don't want to test the actual sendTask to rabbitmq
			rcl.SendTaskWithDelay(
				"sendStateSyncedToHeimdall", "StateSynced",
				logs.Bytes(), ts[i], nil)
			b.StopTimer()

			// then
			b.Logf("SendTaskWithDelay tested successfully")
		}()
	}
}

func BenchmarkCalculateTaskDelay(b *testing.B) {
	b.Skip("to be enabled")
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	clerkTypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	b.ReportAllocs()
	b.ResetTimer()
	b.StopTimer()

	for i := 0; i < b.N; i++ {
		func() {
			b.Logf("Executing iteration '%d' out of '%d'", i, b.N)

			// given
			mockCtrl := prepareMockData(b)
			defer mockCtrl.Finish()

			_, err := prepareClerkProcessor()
			if err != nil {
				b.Fatal("Error initializing test clerk processor")
			}

			// when
			b.StartTimer()

			isCurrentValidator, timeDuration := util.CalculateTaskDelay(nil, cdc)

			b.StopTimer()

			b.Logf("CalculateTaskDelay tested successfully. Results: isCurrentValidator: '%t', timeDuration: '%s'",
				isCurrentValidator, timeDuration.String())
		}()
	}
}

func BenchmarkGetUnconfirmedTxnCount(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	b.StopTimer()

	for i := 0; i < b.N; i++ {
		func() {
			b.Logf("Executing iteration '%d' out of '%d'", i, b.N)

			// given
			mockCtrl := prepareMockData(b)
			defer mockCtrl.Finish()

			_, stopFn, err := prepareRootChainListener()
			if err != nil {
				b.Fatal("Error initializing test listener")
			}

			defer stopFn()

			// when
			b.StartTimer()
			util.GetUnconfirmedTxnCount(nil)
			b.StopTimer()

			// then
			b.Logf("GetUnconfirmedTxnCount tested successfully")
		}()
	}
}

func prepareMockData(b *testing.B) *gomock.Controller {
	b.Helper()

	mockCtrl := gomock.NewController(b)

	mockHttpClient := helperMocks.NewMockHTTPClient(mockCtrl)

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(chainManagerParamsUrl).Return(prepareResponse(chainManagerParamsResponse), nil).AnyTimes()

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(getAccountUrl).Return(prepareResponse(getAccountResponse), nil).AnyTimes()

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(getAccountUrl2).Return(prepareResponse(getAccountResponse), nil).AnyTimes()

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(isOldTxUrl).Return(prepareResponse(isOldTxResponse), nil).AnyTimes()

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(checkpointCountUrl).Return(prepareResponse(checkpointCountResponse), nil).AnyTimes()

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(unconfirmedTxsUrl).Return(prepareResponse(unconfirmedTxsResponse), nil).AnyTimes()

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(getUnconfirmedTxnCountUrl).Return(prepareResponse(getUnconfirmedTxnCountResponse), nil).AnyTimes()

	//nolint: bodyclose
	mockHttpClient.EXPECT().Get(getValidatorSetUrl).Return(prepareResponse(getValidatorSetResponse), nil).AnyTimes()

	helper.Client = mockHttpClient

	return mockCtrl
}

func prepareClerkProcessor() (*ClerkProcessor, error) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	viper.Set(helper.CometBFTNodeFlag, dummyCometBFTNode)
	viper.Set(flags.FlagLogLevel, "debug")
	helper.InitTestHeimdallConfig("")

	srvconf := serverconfig.DefaultConfig()
	configuration := helper.GetDefaultHeimdallConfig()
	srvconf.API.Enable = true
	srvconf.API.Address = dummyHeimdallServerUrl
	configuration.CometBFTRPCUrl = dummyCometBFTNode
	customAppConf := helper.CustomAppConfig{
		Config: *srvconf,
		Custom: configuration,
	}
	helper.SetTestConfig(customAppConf)

	txBroadcaster := broadcaster.NewTxBroadcaster(cdc, client.Context{}, nil)
	txBroadcaster.CliCtx.Simulate = true
	txBroadcaster.CliCtx.SkipConfirm = true

	contractCaller, err := helper.NewContractCaller()
	if err != nil {
		return nil, err
	}

	cp := NewClerkProcessor(&contractCaller.StateSenderABI)
	cp.cliCtx.Simulate = true
	cp.cliCtx.SkipConfirm = true
	cp.BaseProcessor = *NewBaseProcessor(cdc, nil, nil, txBroadcaster, "clerk", cp)

	return cp, nil
}

func prepareRootChainListener() (*listener.RootChainListener, func(), error) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	viper.Set(helper.CometBFTNodeFlag, dummyCometBFTNode)
	viper.Set(flags.FlagLogLevel, "debug")

	srvconf := serverconfig.DefaultConfig()
	configuration := helper.GetDefaultHeimdallConfig()
	srvconf.API.Enable = true
	srvconf.API.Address = dummyHeimdallServerUrl
	configuration.CometBFTRPCUrl = dummyCometBFTNode
	customAppConf := helper.CustomAppConfig{
		Config: *srvconf,
		Custom: configuration,
	}
	helper.SetTestConfig(customAppConf)

	stopFn := func() {}

	rcl := listener.NewRootChainListener()

	server, err := getTestServer()
	if err != nil {
		return nil, stopFn, err
	}

	rcl.BaseListener = *listener.NewBaseListener(
		cdc, &queue.QueueConnector{Server: server}, nil, helper.GetMainClient(), "rootchain", rcl)

	stopFn = func() {
		rcl.Stop()

		if helper.GetMainClient() != nil {
			helper.GetMainClient().Close()
		}

		rcl.BaseListener.Stop()
	}

	return rcl, stopFn, nil
}

func prepareDummyLogBytes() (*bytes.Buffer, error) {
	topics := append([]common.Hash{},
		common.HexToHash("0x103fed9db65eac19c4d870f49ab7520fe03b99f1838e5996caf47e9e43308392"),
		common.HexToHash("0x00000000000000000000000000000000000000000000000000000000001ef6e0"),
		common.HexToHash("0x000000000000000000000000a6fa4fb5f76172d178d61b04b0ecd319c5d1c0aa"))

	log := types.Log{
		Address:     common.HexToAddress("0x28e4f3a7f651294b9564800b2d01f35189a5bfbe"),
		Topics:      topics,
		Data:        common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000010087a7811f4bfedea3d341ad165680ae306b01aaeacc205d227629cf157dd9f821000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000004aa11c9581573571f963bda7a41b28d90c36027c000000000000000000000000eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee0000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000b1a2bc2ec50000"),
		BlockNumber: 14702845,
		TxHash:      common.HexToHash("0x6d428739815d7c84cf89db055158861b089e0fd649676a0243a2a2d204c1d854"),
		TxIndex:     0,
		BlockHash:   common.HexToHash("0xe8370360b861be304ef4144e33a3803cf6d4e31524832444ada797e16f859438"),
		Index:       0,
		Removed:     false,
	}

	reqBodyBytes := new(bytes.Buffer)
	if err := json.NewEncoder(reqBodyBytes).Encode(log); err != nil {
		return nil, err
	}

	return reqBodyBytes, nil
}

func prepareResponse(body string) *http.Response {
	return &http.Response{
		Status:           "200 OK",
		StatusCode:       200,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           nil,
		Body:             io.NopCloser(bytes.NewReader([]byte(body))),
		ContentLength:    0,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          nil,
		TLS:              nil,
	}
}

func getTestServer() (*machinery.Server, error) {
	return machinery.NewServer(&config.Config{
		Broker:        "amqp://guest:guest@localhost:5672/",
		DefaultQueue:  "machinery_tasks",
		ResultBackend: "redis://127.0.0.1:6379",
		AMQP: &config.AMQPConfig{
			Exchange:      "machinery_exchange",
			ExchangeType:  "direct",
			BindingKey:    "machinery_task",
			PrefetchCount: 1,
		},
	})
}

func generateRandNumber(maxValue int64) uint64 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(maxValue))
	if err != nil {
		return 1
	}

	return nBig.Uint64()
}
