package listener

import (
	"encoding/json"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagerTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func TestConfirmedHeadBlock(t *testing.T) {
	rl := &RootChainListener{}
	rl.BaseListener.Logger = log.NewNopLogger()
	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{MainChainTxConfirmations: 10},
	}

	t.Run("a finalized header is returned as-is, confirmations are irrelevant", func(t *testing.T) {
		header := &blockHeader{header: &types.Header{Number: big.NewInt(5)}, isFinalized: true}
		got, ok := rl.confirmedHeadBlock(rootChainContext, header)
		require.True(t, ok)
		require.Equal(t, big.NewInt(5), got)
	})

	t.Run("a non-finalized header below the confirmation requirement is rejected", func(t *testing.T) {
		header := &blockHeader{header: &types.Header{Number: big.NewInt(9)}, isFinalized: false}
		_, ok := rl.confirmedHeadBlock(rootChainContext, header)
		require.False(t, ok)
	})

	t.Run("a non-finalized header exactly at the confirmation requirement is rejected (boundary)", func(t *testing.T) {
		header := &blockHeader{header: &types.Header{Number: big.NewInt(10)}, isFinalized: false}
		_, ok := rl.confirmedHeadBlock(rootChainContext, header)
		require.False(t, ok)
	})

	t.Run("a non-finalized header one past the confirmation requirement is accepted (boundary)", func(t *testing.T) {
		header := &blockHeader{header: &types.Header{Number: big.NewInt(11)}, isFinalized: false}
		got, ok := rl.confirmedHeadBlock(rootChainContext, header)
		require.True(t, ok)
		require.Equal(t, big.NewInt(1), got)
	})

	t.Run("a non-finalized header well past the confirmation requirement subtracts it", func(t *testing.T) {
		header := &blockHeader{header: &types.Header{Number: big.NewInt(100)}, isFinalized: false}
		got, ok := rl.confirmedHeadBlock(rootChainContext, header)
		require.True(t, ok)
		require.Equal(t, big.NewInt(90), got)
	})
}

func TestFromBlockAfterLastPersisted(t *testing.T) {
	t.Run("no last block persisted returns headerNumber itself", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)

		got, ok := rl.fromBlockAfterLastPersisted(big.NewInt(50))
		require.True(t, ok)
		require.Equal(t, big.NewInt(50), got)
	})

	t.Run("a last block below headerNumber returns last+1", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)
		require.NoError(t, rl.storageClient.Put([]byte(lastRootBlockKey), []byte("40"), nil))

		got, ok := rl.fromBlockAfterLastPersisted(big.NewInt(50))
		require.True(t, ok)
		require.Equal(t, big.NewInt(41), got)
	})

	t.Run("a last block exactly at headerNumber has nothing new (boundary)", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)
		require.NoError(t, rl.storageClient.Put([]byte(lastRootBlockKey), []byte("50"), nil))

		_, ok := rl.fromBlockAfterLastPersisted(big.NewInt(50))
		require.False(t, ok)
	})

	t.Run("a last block past headerNumber has nothing new", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)
		require.NoError(t, rl.storageClient.Put([]byte(lastRootBlockKey), []byte("51"), nil))

		_, ok := rl.fromBlockAfterLastPersisted(big.NewInt(50))
		require.False(t, ok)
	})

	t.Run("an unparseable last block falls back to headerNumber", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)
		require.NoError(t, rl.storageClient.Put([]byte(lastRootBlockKey), []byte("not-a-number"), nil))

		got, ok := rl.fromBlockAfterLastPersisted(big.NewInt(50))
		require.True(t, ok)
		require.Equal(t, big.NewInt(50), got)
	})
}

func TestRootChainBlockRangeToProcess(t *testing.T) {
	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{MainChainTxConfirmations: 10},
	}

	t.Run("rejects a header too recent to be confirmed", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)

		header := &blockHeader{header: &types.Header{Number: big.NewInt(5)}, isFinalized: false}
		_, _, ok := rl.rootChainBlockRangeToProcess(rootChainContext, header)
		require.False(t, ok)
	})

	t.Run("rejects a header with nothing new past the last persisted block", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)
		require.NoError(t, rl.storageClient.Put([]byte(lastRootBlockKey), []byte("100"), nil))

		header := &blockHeader{header: &types.Header{Number: big.NewInt(100)}, isFinalized: true}
		_, _, ok := rl.rootChainBlockRangeToProcess(rootChainContext, header)
		require.False(t, ok)
	})

	t.Run("with no last persisted block, from and to both equal the confirmed head", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)

		header := &blockHeader{header: &types.Header{Number: big.NewInt(100)}, isFinalized: true}
		from, to, ok := rl.rootChainBlockRangeToProcess(rootChainContext, header)
		require.True(t, ok)
		require.Equal(t, big.NewInt(100), from)
		require.Equal(t, big.NewInt(100), to)
	})

	t.Run("with a last persisted block, from starts right after it", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)
		require.NoError(t, rl.storageClient.Put([]byte(lastRootBlockKey), []byte("90"), nil))

		header := &blockHeader{header: &types.Header{Number: big.NewInt(100)}, isFinalized: true}
		from, to, ok := rl.rootChainBlockRangeToProcess(rootChainContext, header)
		require.True(t, ok)
		require.Equal(t, big.NewInt(91), from)
		require.Equal(t, big.NewInt(100), to)
	})
}

// emptyLogsListener builds a RootChainListener wired against a mock
// eth_getLogs endpoint that always succeeds with zero logs, plus a real
// in-memory storage client, so the chunk-loop's own boundaries can be
// exercised without any log-validation noise. requestCount is incremented on
// every eth_getLogs call the chunk loop makes.
func emptyLogsListener(t *testing.T) (*RootChainListener, *RootChainListenerContext, *int) {
	t.Helper()

	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}

	rl := &RootChainListener{
		eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
		eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
	}
	rl.BaseListener.Logger = log.NewNopLogger()
	rl.storageClient = newMemLevelDB(t)

	var requestCount int
	emptyLogs, err := json.Marshal([]*types.Log{})
	require.NoError(t, err)
	rpcURL := mockEthGetLogs(t, string(emptyLogs), &requestCount)
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client
	rl.contractCaller.MainChainTimeout = 5_000_000_000 // 5s, as time.Duration nanoseconds

	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{
			ChainParams: chainmanagerTypes.ChainParams{
				RootChainAddress:   "0x1111111111111111111111111111111111111111",
				StakingInfoAddress: "0x2222222222222222222222222222222222222222",
				StateSenderAddress: "0x3333333333333333333333333333333333333333",
			},
		},
	}

	return rl, rootChainContext, &requestCount
}

func TestProcessRootChainBlockRangeInChunks(t *testing.T) {
	t.Run("a range exactly one chunk wide makes a single query and persists its end", func(t *testing.T) {
		rl, rootChainContext, requestCount := emptyLogsListener(t)

		from := big.NewInt(1000)
		to := new(big.Int).Add(from, big.NewInt(maxRootChainBlockRange-1))
		rl.processRootChainBlockRangeInChunks(rootChainContext, from, to)

		require.Equal(t, 1, *requestCount)
		lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
		require.NoError(t, err)
		require.Equal(t, to.String(), string(lastBlockBytes))
	})

	t.Run("a range one block wider than a chunk makes two queries and persists the true end", func(t *testing.T) {
		rl, rootChainContext, requestCount := emptyLogsListener(t)

		from := big.NewInt(1000)
		to := new(big.Int).Add(from, big.NewInt(maxRootChainBlockRange))
		rl.processRootChainBlockRangeInChunks(rootChainContext, from, to)

		require.Equal(t, 2, *requestCount)
		lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
		require.NoError(t, err)
		require.Equal(t, to.String(), string(lastBlockBytes))
	})

	t.Run("a from equal to to is a single-block chunk", func(t *testing.T) {
		rl, rootChainContext, requestCount := emptyLogsListener(t)

		rl.processRootChainBlockRangeInChunks(rootChainContext, big.NewInt(1000), big.NewInt(1000))

		require.Equal(t, 1, *requestCount)
		lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
		require.NoError(t, err)
		require.Equal(t, "1000", string(lastBlockBytes))
	})
}

func TestProcessRootChainBlockRange_RightHalfNotAttemptedWhenLeftFails(t *testing.T) {
	rl, rootChainContext := newQuarantineTestListener(t)

	// [100,100] always fails (the bad log); [101,101] is never queried
	// because processRootChainBlockRange gives up on the whole range as soon
	// as its left half fails, without trying the right half.
	err := rl.processRootChainBlockRange(rootChainContext, big.NewInt(100), big.NewInt(101), newRootChainRejectionState())
	require.Error(t, err)

	_, err = rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.Error(t, err, "the cursor must not advance when the range as a whole fails")
}

func TestPersistLastRootBlock(t *testing.T) {
	t.Run("writes the block number as a readable string", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		rl.storageClient = newMemLevelDB(t)

		require.NoError(t, rl.persistLastRootBlock(big.NewInt(12345)))

		got, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
		require.NoError(t, err)
		require.Equal(t, "12345", string(got))
	})

	t.Run("returns an error when the storage client can't be written to", func(t *testing.T) {
		rl := &RootChainListener{}
		rl.BaseListener.Logger = log.NewNopLogger()
		db := newMemLevelDB(t)
		require.NoError(t, db.Close())
		rl.storageClient = db

		require.Error(t, rl.persistLastRootBlock(big.NewInt(1)))
	})
}
