package engine

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	testState = ForkChoiceState{
		HeadHash:           common.HexToHash("0x10"),
		SafeBlockHash:      common.HexToHash("0x01"),
		FinalizedBlockHash: common.HexToHash("0x11"),
	}
	testWithdrawals = []*Withdrawal{
		{
			Index:     "0x1",
			Validator: "0x2",
			Address:   common.HexToAddress("0x12340000").Hex(),
			Amount:    "0x3",
		},
		{
			Index:     "0x2",
			Validator: "0x3",
			Address:   common.HexToAddress("0x98760000").Hex(),
			Amount:    "0x4",
		},
	}
	testPayload = PayloadAttributes{
		Timestamp:             0,
		PrevRandao:            common.HexToHash("0x2222"),
		SuggestedFeeRecipient: common.HexToAddress("0x1337"),
		Withdrawals:           testWithdrawals,
	}
	expectedStateJson   = `{"headBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000010","safeBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000001","finalizedBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000011"}`
	expectedPayloadJson = `{"timestamp":"0x0","prevRandao":"0x0000000000000000000000000000000000000000000000000000000000002222","suggestedFeeRecipient":"0x0000000000000000000000000000000000001337","withdrawals":[{"index":"0x1","validatorIndex":"0x2","address":"0x0000000000000000000000000000000012340000","amount":"0x3"},{"index":"0x2","validatorIndex":"0x3","address":"0x0000000000000000000000000000000098760000","amount":"0x4"}],"parentBeaconBlockRoot":"0x0000000000000000000000000000000000000000000000000000000000000000"}`
)

func TestJsonRpcTypeSerialization(t *testing.T) {
	bytes, err := json.Marshal(testState)
	require.NoError(t, err)
	require.Equal(t, expectedStateJson, string(bytes))

	bytes, err = json.Marshal(testPayload)
	require.NoError(t, err)
	require.Equal(t, expectedPayloadJson, string(bytes))

	payload := []interface{}{testState, testPayload}
	expectedPayload := fmt.Sprintf("[%s,%s]", expectedStateJson, expectedPayloadJson)

	bytes, err = json.Marshal(payload)
	require.NoError(t, err)
	require.Equal(t, expectedPayload, string(bytes))
}

func TestEthRequestSerialization(t *testing.T) {
	params := []interface{}{testState, testPayload}
	er := JsonrpcRequest{
		ID:      1,
		JSONRPC: "2.0",
		Method:  "eth_someRpcMethod",
		Params:  params,
	}
	bytes, err := json.Marshal(er)
	require.NoError(t, err)

	require.Equal(t,
		fmt.Sprintf(`{"id":1,"jsonrpc":"2.0","method":"eth_someRpcMethod","params":[%s,%s]}`,
			expectedStateJson, expectedPayloadJson), string(bytes))
}
