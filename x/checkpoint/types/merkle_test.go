package types_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestIsValidCheckpoint_BorQueryFailures_WrapDistinctSentinels(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*helpermocks.IContractCaller)
		// Unique (end, confirmations) per row avoids the package-level existsCache.
		start, end, checkpointLength, confirmations uint64
		wantSentinel                                error
	}{
		{
			name: "ethereum.NotFound or nil header -> (false, nil)",
			setupMock: func(c *helpermocks.IContractCaller) {
				c.On("CheckIfBlocksExist", mock.Anything).Return(false, nil)
			},
			start: 100_000, end: 200_000, checkpointLength: 256, confirmations: 10,
			wantSentinel: borTypes.ErrBorBlockNotFound,
		},
		{
			name: "JSON-RPC failure -> (false, err)",
			setupMock: func(c *helpermocks.IContractCaller) {
				c.On("CheckIfBlocksExist", mock.Anything).
					Return(false, fmt.Errorf("bor unreachable"))
			},
			start: 300_000, end: 400_000, checkpointLength: 256, confirmations: 10,
			wantSentinel: borTypes.ErrFailedToQueryBor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caller := new(helpermocks.IContractCaller)
			tt.setupMock(caller)

			ok, err := checkpointTypes.IsValidCheckpoint(
				tt.start,
				tt.end,
				[]byte{0xde, 0xad},
				tt.checkpointLength,
				caller,
				tt.confirmations,
			)

			require.False(t, ok)
			require.Error(t, err)
			require.True(t,
				errors.Is(err, tt.wantSentinel),
				"expected errors.Is(err, %v); got %v", tt.wantSentinel, err,
			)
			require.True(t,
				strings.Contains(err.Error(), fmt.Sprintf("target=%d", tt.end+tt.confirmations)),
				"expected target=%d in error; got %v", tt.end+tt.confirmations, err,
			)
		})
	}
}
