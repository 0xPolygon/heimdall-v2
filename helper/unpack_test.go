package helper

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const testFooEventABI = `[{"anonymous":false,"inputs":[{"indexed":false,"name":"value","type":"uint256"}],"name":"Foo","type":"event"}]`

func mustParseTestFooABI(t *testing.T) abi.ABI {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(testFooEventABI))
	require.NoError(t, err)
	return parsed
}

// TestUnpackLog_NoTopics verifies a log with zero topics (a valid anonymous
// Ethereum event, just one this decoder can't identify) is rejected here,
// not by rejecting the whole receipt it came from.
func TestUnpackLog_NoTopics(t *testing.T) {
	fooABI := mustParseTestFooABI(t)

	var out struct{ Value *big.Int }
	err := UnpackLog(&fooABI, &out, "Foo", &types.Log{Topics: nil})

	require.Error(t, err)
	require.Contains(t, err.Error(), "no topics")
}

// TestUnpackLog_MatchingEvent verifies the success path is unaffected by the
// new topics-length check: a real log with its event-signature topic set
// still decodes correctly.
func TestUnpackLog_MatchingEvent(t *testing.T) {
	fooABI := mustParseTestFooABI(t)

	fooEventID := fooABI.Events["Foo"].ID
	data, err := fooABI.Events["Foo"].Inputs.NonIndexed().Pack(big.NewInt(42))
	require.NoError(t, err)

	var out struct{ Value *big.Int }
	err = UnpackLog(&fooABI, &out, "Foo", &types.Log{
		Topics: []common.Hash{fooEventID},
		Data:   data,
	})

	require.NoError(t, err)
	require.Equal(t, big.NewInt(42), out.Value)
}

// TestUnpackLog_UnrelatedAnonymousLogDoesNotBlockMatchedLog verifies a
// receipt containing both a zero-topic anonymous log and the actual target
// log still lets the target log decode, mirroring the real Decode*Event
// matching pattern.
func TestUnpackLog_UnrelatedAnonymousLogDoesNotBlockMatchedLog(t *testing.T) {
	fooABI := mustParseTestFooABI(t)

	fooEventID := fooABI.Events["Foo"].ID
	data, err := fooABI.Events["Foo"].Inputs.NonIndexed().Pack(big.NewInt(7))
	require.NoError(t, err)

	contractAddress := common.HexToAddress("0xabc")
	logs := []*types.Log{
		{Topics: nil, Index: 0, Address: contractAddress}, // unrelated anonymous log, same contract, different index
		{Topics: []common.Hash{fooEventID}, Data: data, Index: 1, Address: contractAddress},
	}

	const targetLogIndex = 1
	var matched *types.Log
	for _, vLog := range logs {
		if uint64(vLog.Index) == targetLogIndex && vLog.Address == contractAddress {
			matched = vLog
			break
		}
	}
	require.NotNil(t, matched, "matching loop must reach the target log despite the unrelated anonymous log")

	var out struct{ Value *big.Int }
	require.NoError(t, UnpackLog(&fooABI, &out, "Foo", matched))
	require.Equal(t, big.NewInt(7), out.Value)
}
