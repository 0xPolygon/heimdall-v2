package broadcaster

import (
	"os"
	"strconv"
	"testing"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Parallel test - to check BroadcastToHeimdall synchronisation
func TestBroadcastToHeimdall(t *testing.T) {
	t.Parallel()

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// cli context
	CometBFTNode := "tcp://localhost:26657"
	viper.Set(helper.CometBFTNodeFlag, CometBFTNode)
	viper.Set("log_level", "info")

	helper.InitHeimdallConfig(os.ExpandEnv("$HOME/.heimdalld"))

	_txBroadcaster := NewTxBroadcaster(cdc)

	testData := []checkpointTypes.MsgCheckpoint{
		{Proposer: string(helper.GetAddress()[:]), StartBlock: 0, EndBlock: 63, RootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}, AccountRootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}},
		{Proposer: string(helper.GetAddress()[:]), StartBlock: 64, EndBlock: 1024, RootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}, AccountRootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}},
		{Proposer: string(helper.GetAddress()[:]), StartBlock: 1025, EndBlock: 2048, RootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}, AccountRootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}},
		{Proposer: string(helper.GetAddress()[:]), StartBlock: 2049, EndBlock: 3124, RootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}, AccountRootHash: hmTypes.HeimdallHash{Hash: []byte("0x5bd83f679c8ce7c48d6fa52ce41532fcacfbbd99d5dab415585f397bf44a0b6e")}},
	}

	for index, test := range testData { //nolint
		index := index
		test := test

		t.Run(strconv.Itoa(index), func(t *testing.T) {
			t.Parallel()

			// create and send checkpoint message
			msg := checkpointTypes.NewMsgCheckpointBlock(
				test.Proposer,
				test.StartBlock,
				test.EndBlock,
				test.RootHash,
				test.AccountRootHash,
				test.BorChainID,
			)

			err := _txBroadcaster.BroadcastToHeimdall(&msg, nil)
			assert.Empty(t, err, "Error broadcasting tx to heimdall", err)
		})
	}
}
