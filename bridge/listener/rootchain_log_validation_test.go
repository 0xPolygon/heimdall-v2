package listener

import (
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestRootChainListener_ValidateLogAgainstQuery(t *testing.T) {
	t.Parallel()

	// Must be a name in rootChainEvents (a real queried event), not an
	// arbitrary string — validateLogAgainstQuery checks membership there.
	// Belongs to the RootChain contract.
	rootChainTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	rootChainEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}

	// A second real, queried event, belonging to a different contract
	// (StakingInfo) — used to test cross-contract topic/address mismatches.
	stakingInfoTopic := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	stakingInfoEvent := &abi.Event{Name: helper.StakedEvent}

	// In eventMap (so the topic resolves) but its event name isn't in
	// rootChainEvents — the gap the query-scope check exists to close.
	unqueriedTopic := common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
	unqueriedEvent := &abi.Event{Name: "OwnershipTransferred"}

	rootChainAddress := common.HexToAddress("0xaaaa000000000000000000000000000000aaaa")
	stakingInfoAddress := common.HexToAddress("0xbbbb000000000000000000000000000000bbbb")
	stateSenderAddress := common.HexToAddress("0xcccc000000000000000000000000000000cccc")

	contractAddresses := map[rootChainContract]common.Address{
		rootChainContractRootChain:   rootChainAddress,
		rootChainContractStakingInfo: stakingInfoAddress,
		rootChainContractStateSender: stateSenderAddress,
	}

	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)

	newListener := func() *RootChainListener {
		rl := &RootChainListener{
			eventMap: map[common.Hash]*abi.Event{
				rootChainTopic:   rootChainEvent,
				stakingInfoTopic: stakingInfoEvent,
				unqueriedTopic:   unqueriedEvent,
			},
			eventContract: map[common.Hash]rootChainContract{
				rootChainTopic:   rootChainContractRootChain,
				stakingInfoTopic: rootChainContractStakingInfo,
				unqueriedTopic:   rootChainContractRootChain,
			},
		}
		rl.BaseListener.Logger = log.NewNopLogger()
		return rl
	}

	t.Run("accepts a log matching its event's contract, range, and topic", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		event, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.True(t, ok)
		require.Same(t, rootChainEvent, event)
	})

	t.Run("accepts a log from a different watched contract when its topic matches", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     stakingInfoAddress,
			Topics:      []common.Hash{stakingInfoTopic},
			BlockNumber: 150,
		}

		event, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.True(t, ok)
		require.Same(t, stakingInfoEvent, event)
	})

	t.Run("accepts a log at the exact range boundaries", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		lowerBound := types.Log{Address: rootChainAddress, Topics: []common.Hash{rootChainTopic}, BlockNumber: fromBlock.Uint64()}
		_, ok := rl.validateLogAgainstQuery(lowerBound, contractAddresses, fromBlock, toBlock)
		require.True(t, ok)

		upperBound := types.Log{Address: rootChainAddress, Topics: []common.Hash{rootChainTopic}, BlockNumber: toBlock.Uint64()}
		_, ok = rl.validateLogAgainstQuery(upperBound, contractAddresses, fromBlock, toBlock)
		require.True(t, ok)
	})

	t.Run("rejects a log from an address outside all watched contracts", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     common.HexToAddress("0xdddd000000000000000000000000000000dddd"),
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a log whose topic belongs to a different contract than its address", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		// A real, watched contract address (StakingInfo) paired with a topic
		// that belongs to RootChain — right address, wrong topic-family.
		vLog := types.Log{
			Address:     stakingInfoAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a log whose address belongs to a different contract than its topic", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		// The topic resolves to a StakingInfo event, but the log claims to
		// come from RootChainAddress — right topic, wrong address.
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{stakingInfoTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a log below the queried block range", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: fromBlock.Uint64() - 1,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a log above the queried block range", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: toBlock.Uint64() + 1,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a log with no topics", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a log with a topic outside the configured event set", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a log whose event resolves in eventMap but isn't in the query's topic set", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{unqueriedTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})

	t.Run("rejects a removed (reorg'd) log", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
			Removed:     true,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		require.False(t, ok)
	})
}

func TestRootChainListener_ValidateAndHandleLogs(t *testing.T) {
	t.Parallel()

	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.StakedEvent}
	rootChainAddress := common.HexToAddress("0xaaaa000000000000000000000000000000aaaa")
	contractAddresses := map[rootChainContract]common.Address{
		rootChainContractRootChain: rootChainAddress,
	}
	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)

	newListener := func() *RootChainListener {
		rl := &RootChainListener{
			eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
			eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
		}
		rl.BaseListener.Logger = log.NewNopLogger()
		return rl
	}

	t.Run("a bad log anywhere in the batch fails the whole batch and dispatches nothing", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		// A valid log ahead of an invalid one: if dispatch ran before the whole
		// batch was validated, this would call handleLog for the valid entry
		// (which needs ABI/codec plumbing this test deliberately doesn't set up,
		// so it would panic) before returning the error for the invalid one.
		logs := []types.Log{
			{Address: rootChainAddress, Topics: []common.Hash{knownTopic}, BlockNumber: 150},
			{Address: common.HexToAddress("0xcccc000000000000000000000000000000cccc"), Topics: []common.Hash{knownTopic}, BlockNumber: 150},
		}

		var err error
		require.NotPanics(t, func() {
			err = rl.validateAndHandleLogs(logs, contractAddresses, fromBlock, toBlock)
		})
		require.ErrorIs(t, err, errUnexpectedRootChainLog)
	})

	t.Run("an empty batch succeeds with no dispatch", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		err := rl.validateAndHandleLogs(nil, contractAddresses, fromBlock, toBlock)
		require.NoError(t, err)
	})
}
