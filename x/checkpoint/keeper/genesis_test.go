package keeper_test

import (
	"math/rand"
	"time"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"
)

func (s *KeeperTestSuite) TestInitExportGenesis() {
	ctx, _, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	lastNoACK := simulation.RandIntBetween(r1, 1, 5)
	ackCount := simulation.RandIntBetween(r1, 1, 5)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := hmTypes.HexToHeimdallHash("123")

	proposerAddress := common.Address{}.String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"

	bufferedCheckpoint := types.CreateBlock(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		borChainId,
		timestamp,
	)

	checkpoints := make([]types.Checkpoint, ackCount)

	for i := range checkpoints {
		checkpoints[i] = bufferedCheckpoint
	}

	params := types.Params{}
	genesisState := types.NewGenesisState(
		params,
		&bufferedCheckpoint,
		uint64(lastNoACK),
		uint64(ackCount),
		checkpoints,
	)

	keeper.InitGenesis(ctx, &genesisState)

	actualParams := keeper.ExportGenesis(ctx)

	require.Equal(genesisState.AckCount, actualParams.AckCount)
	require.Equal(genesisState.BufferedCheckpoint, actualParams.BufferedCheckpoint)
	require.Equal(genesisState.LastNoACK, actualParams.LastNoACK)
	require.Equal(genesisState.Params, actualParams.Params)
	require.LessOrEqual(len(actualParams.Checkpoints), len(genesisState.Checkpoints))
}
