package keeper_test

import (
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func (s *KeeperTestSuite) TestInitExportGenesis() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	lastNoACK := simulation.RandIntBetween(r1, 1, 5)
	ackCount := simulation.RandIntBetween(r1, 1, 5)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := hmTypes.HeimdallHash{Hash: testutil.RandomBytes()}

	proposerAddress := common.Address{}.String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"

	bufferedCheckpoint := types.CreateCheckpoint(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		borChainId,
		timestamp,
	)

	checkpoints := make([]types.Checkpoint, 0, ackCount)
	params := types.DefaultParams()
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
	require.Equal(genesisState.LastNoAck, actualParams.LastNoAck)
	require.Equal(genesisState.Params, actualParams.Params)
	require.LessOrEqual(len(actualParams.Checkpoints), len(genesisState.Checkpoints))
}
