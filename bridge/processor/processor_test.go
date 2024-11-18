package processor

import (
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/helper"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestBroadcastWhenTxInMempool(t *testing.T) {
	// TODO HV2: fix this test as it currently depends on the config file
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")

	t.Parallel()

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	viper.Set(helper.CometBFTNodeFlag, "http://localhost:26657")
	viper.Set("log_level", "info")

	helper.InitHeimdallConfig(os.ExpandEnv("$HOME/.heimdalld"))

	_txBroadcaster := broadcaster.NewTxBroadcaster(cdc)

	defaultMessage := clerkTypes.MsgEventRecordRequest{
		From:            string(helper.GetAddress()[:]),
		TxHash:          "0xa86a2f8f30d5fab1bb858e572ecf3f24d691276f833f06fc90a745cea20f4fcb",
		LogIndex:        71,
		BlockNumber:     14475337,
		ContractAddress: "0x401f6c983ea34274ec46f84d70b31c151321188b",
		Data:            []byte{},
		Id:              1897091,
		ChainId:         "15001",
	}

	// adding clerk messages and errors for testing
	var (
		testData       []clerkTypes.MsgEventRecordRequest
		expectedStatus []bool
	)

	// keep the first 2 messages same (default)
	testData = append(testData, defaultMessage)
	expectedStatus = append(expectedStatus, false) // 1st message should go through as it's unique in mempool
	testData = append(testData, defaultMessage)
	expectedStatus = append(expectedStatus, true) // 2nd message should fail, as it's duplicate

	// change the txhash in the 3rd message
	defaultMessage2 := defaultMessage
	defaultMessage2.TxHash = "0x8a83aa78a400fe959b44ccf70d926c967af4e451ba630a849b2e1dedc7e30c07"
	testData = append(testData, defaultMessage2)
	expectedStatus = append(expectedStatus, false) // 3rd message should go through as the txhash is different

	// change the log index in the 4th message
	defaultMessage2 = defaultMessage
	defaultMessage2.LogIndex = 72
	testData = append(testData, defaultMessage2)
	expectedStatus = append(expectedStatus, false) // 4th message should go through as the log index is different

	// create a mock clerk processor
	contractCaller, err := helper.NewContractCaller()
	if err != nil {
		t.Fatal(err)
	}

	cp := NewClerkProcessor(&contractCaller.StateSenderABI)
	cp.BaseProcessor = *NewBaseProcessor(cdc, nil, nil, nil, "clerk", cp)

	for index, tx := range testData { //nolint
		index := index
		tx := tx

		t.Run(string(rune(index)), func(t *testing.T) {
			t.Parallel()

			inMempool, err := cp.checkTxAgainstMempool(&tx, nil)
			t.Log("Done checking tx against mempool", "in mempool", inMempool)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, inMempool, expectedStatus[index])
			if !inMempool {
				t.Log("Tx not in mempool, broadcasting")
				_, err = _txBroadcaster.BroadcastToHeimdall(&tx, nil)
				assert.Empty(t, err, "Error broadcasting tx to heimdall", err)
			} else {
				t.Log("Tx is already in mempool, not broadcasting")
			}
		})
	}
}
