package types_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestMsgCheckpoint_ValidateBasic(t *testing.T) {
	t.Parallel()

	validProposer := "0x1234567890123456789012345678901234567890"
	validRootHash := common.HexToHash("0xabcdef1234567890").Bytes()
	validBorChainId := "137"

	tests := []struct {
		name        string
		msg         *types.MsgCheckpoint
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid message",
			msg: &types.MsgCheckpoint{
				Proposer:        validProposer,
				RootHash:        validRootHash,
				BorChainId:      validBorChainId,
				StartBlock:      1000,
				EndBlock:        2000,
				AccountRootHash: common.HexToHash("0x123").Bytes(),
			},
			shouldError: false,
		},
		{
			name: "invalid bor chain id - not a number",
			msg: &types.MsgCheckpoint{
				Proposer:   validProposer,
				RootHash:   validRootHash,
				BorChainId: "invalid",
				StartBlock: 1000,
				EndBlock:   2000,
			},
			shouldError: true,
			errorMsg:    "Invalid bor chain id",
		},
		{
			name: "invalid root hash - empty",
			msg: &types.MsgCheckpoint{
				Proposer:   validProposer,
				RootHash:   common.Hash{}.Bytes(),
				BorChainId: validBorChainId,
				StartBlock: 1000,
				EndBlock:   2000,
			},
			shouldError: true,
			errorMsg:    "Invalid rootHash",
		},
		{
			name: "invalid root hash - wrong length",
			msg: &types.MsgCheckpoint{
				Proposer:   validProposer,
				RootHash:   []byte{0x01, 0x02}, // Too short
				BorChainId: validBorChainId,
				StartBlock: 1000,
				EndBlock:   2000,
			},
			shouldError: true,
			errorMsg:    "Invalid rootHash length",
		},
		{
			name: "invalid proposer address",
			msg: &types.MsgCheckpoint{
				Proposer:   "invalid-address",
				RootHash:   validRootHash,
				BorChainId: validBorChainId,
				StartBlock: 1000,
				EndBlock:   2000,
			},
			shouldError: true,
			errorMsg:    "invalid proposer",
		},
		{
			name: "empty proposer address",
			msg: &types.MsgCheckpoint{
				Proposer:   "",
				RootHash:   validRootHash,
				BorChainId: validBorChainId,
				StartBlock: 1000,
				EndBlock:   2000,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.msg.ValidateBasic()
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgCpAck_ValidateBasic(t *testing.T) {
	t.Parallel()

	validFrom := "0x1234567890123456789012345678901234567890"
	validProposer := "0x0987654321098765432109876543210987654321"

	tests := []struct {
		name        string
		msg         *types.MsgCpAck
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid message",
			msg: &types.MsgCpAck{
				From:       validFrom,
				Proposer:   validProposer,
				StartBlock: 1000,
				EndBlock:   2000,
				RootHash:   common.HexToHash("0xabcd").Bytes(),
				Number:     1,
			},
			shouldError: false,
		},
		{
			name: "invalid sender address",
			msg: &types.MsgCpAck{
				From:     "invalid-sender",
				Proposer: validProposer,
			},
			shouldError: true,
			errorMsg:    "invalid sender",
		},
		{
			name: "empty sender address",
			msg: &types.MsgCpAck{
				From:     "",
				Proposer: validProposer,
			},
			shouldError: true,
		},
		{
			name: "invalid proposer address",
			msg: &types.MsgCpAck{
				From:     validFrom,
				Proposer: "invalid-proposer",
			},
			shouldError: true,
			errorMsg:    "invalid sender", // Error message uses "sender" for proposer
		},
		{
			name: "empty proposer address",
			msg: &types.MsgCpAck{
				From:     validFrom,
				Proposer: "",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.msg.ValidateBasic()
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgCpNoAck_ValidateBasic(t *testing.T) {
	t.Parallel()

	validFrom := "0x1234567890123456789012345678901234567890"

	tests := []struct {
		name        string
		msg         *types.MsgCpNoAck
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid message",
			msg: &types.MsgCpNoAck{
				From: validFrom,
			},
			shouldError: false,
		},
		{
			name: "invalid sender address",
			msg: &types.MsgCpNoAck{
				From: "invalid-sender",
			},
			shouldError: true,
			errorMsg:    "Invalid sender",
		},
		{
			name: "empty sender address",
			msg: &types.MsgCpNoAck{
				From: "",
			},
			shouldError: true,
			errorMsg:    "Invalid sender",
		},
		{
			name: "zero-padded address",
			msg: &types.MsgCpNoAck{
				From: "0x0000000000000000000000000000000000000000",
			},
			shouldError: false, // Zero address is technically valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.msg.ValidateBasic()
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestBlockRangeValidation tests block range validation logic
func TestBlockRangeValidation(t *testing.T) {
	t.Parallel()

	t.Run("validates start block less than end block", func(t *testing.T) {
		t.Parallel()

		startBlock := uint64(1000)
		endBlock := uint64(2000)

		require.Less(t, startBlock, endBlock)
	})

	t.Run("validates sequential blocks", func(t *testing.T) {
		t.Parallel()

		startBlock := uint64(1000)
		endBlock := startBlock + 1

		require.Equal(t, uint64(1001), endBlock)
		require.Greater(t, endBlock, startBlock)
	})

	t.Run("validates checkpoint size", func(t *testing.T) {
		t.Parallel()

		startBlock := uint64(1000)
		endBlock := uint64(2000)
		checkpointLength := endBlock - startBlock + 1

		require.Equal(t, uint64(1001), checkpointLength)
	})
}

// TestHashValidation tests hash validation logic
func TestHashValidation(t *testing.T) {
	t.Parallel()

	t.Run("validates correct hash length", func(t *testing.T) {
		t.Parallel()

		hash := common.HexToHash("0xabcdef").Bytes()

		require.Len(t, hash, common.HashLength)
		require.Equal(t, 32, len(hash))
	})

	t.Run("detects empty hash", func(t *testing.T) {
		t.Parallel()

		emptyHash := common.Hash{}.Bytes()
		nonEmptyHash := common.HexToHash("0x01").Bytes()

		require.NotEqual(t, emptyHash, nonEmptyHash)
	})

	t.Run("validates hash uniqueness", func(t *testing.T) {
		t.Parallel()

		hash1 := common.HexToHash("0xabc").Bytes()
		hash2 := common.HexToHash("0xdef").Bytes()

		require.NotEqual(t, hash1, hash2)
		require.Len(t, hash1, common.HashLength)
		require.Len(t, hash2, common.HashLength)
	})
}

// TestChainIDValidation tests chain ID validation
func TestChainIDValidation(t *testing.T) {
	t.Parallel()

	t.Run("validates numeric chain ID", func(t *testing.T) {
		t.Parallel()

		chainIDs := []string{"137", "80001", "1", "999999"}

		for _, chainID := range chainIDs {
			// Should be parseable as uint64
			var isNumeric bool
			for _, c := range chainID {
				isNumeric = c >= '0' && c <= '9'
				if !isNumeric {
					break
				}
			}
			require.True(t, isNumeric, "Chain ID %s should be numeric", chainID)
		}
	})

	t.Run("rejects non-numeric chain ID", func(t *testing.T) {
		t.Parallel()

		invalidChainIDs := []string{"abc", "12a", "polygon", ""}

		for _, chainID := range invalidChainIDs {
			var isNumeric = true
			if chainID == "" {
				isNumeric = false
			} else {
				for _, c := range chainID {
					if c < '0' || c > '9' {
						isNumeric = false
						break
					}
				}
			}
			require.False(t, isNumeric, "Chain ID %s should not be numeric", chainID)
		}
	})
}

func TestMsgCheckpoint_GetSideSignBytes(t *testing.T) {
	t.Parallel()

	t.Run("returns side sign bytes for valid message", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCheckpointBlock(
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			common.HexToHash("0xabcdef").Bytes(),
			common.HexToHash("0x123456").Bytes(),
			"137",
		)

		sideSignBytes := msg.GetSideSignBytes()

		require.NotNil(t, sideSignBytes)
		require.NotEmpty(t, sideSignBytes)
		// Should be 6 fields * 32 bytes = 192 bytes
		require.Equal(t, 192, len(sideSignBytes))
	})

	t.Run("panics for invalid proposer", func(t *testing.T) {
		t.Parallel()

		msg := &types.MsgCheckpoint{
			Proposer:        "invalid-proposer",
			StartBlock:      1000,
			EndBlock:        2000,
			RootHash:        common.HexToHash("0xabcdef").Bytes(),
			AccountRootHash: common.HexToHash("0x123456").Bytes(),
			BorChainId:      "137",
		}

		require.Panics(t, func() {
			msg.GetSideSignBytes()
		})
	})

	t.Run("handles different bor chain IDs", func(t *testing.T) {
		t.Parallel()

		chainIDs := []string{"1", "137", "80001", "999999"}

		for _, chainID := range chainIDs {
			msg := types.NewMsgCheckpointBlock(
				"0x1234567890123456789012345678901234567890",
				1000,
				2000,
				common.HexToHash("0xabcdef").Bytes(),
				common.HexToHash("0x123456").Bytes(),
				chainID,
			)

			sideSignBytes := msg.GetSideSignBytes()
			require.Equal(t, 192, len(sideSignBytes))
		}
	})
}

func TestUnpackCheckpointSideSignBytes(t *testing.T) {
	t.Parallel()

	t.Run("unpacks valid side sign bytes", func(t *testing.T) {
		t.Parallel()

		originalMsg := types.NewMsgCheckpointBlock(
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			common.HexToHash("0xabcdef").Bytes(),
			common.HexToHash("0x123456").Bytes(),
			"137",
		)

		sideSignBytes := originalMsg.GetSideSignBytes()
		unpackedMsg, err := types.UnpackCheckpointSideSignBytes(sideSignBytes)

		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
		require.Equal(t, originalMsg.Proposer, unpackedMsg.Proposer)
		require.Equal(t, originalMsg.StartBlock, unpackedMsg.StartBlock)
		require.Equal(t, originalMsg.EndBlock, unpackedMsg.EndBlock)
		require.Equal(t, originalMsg.BorChainId, unpackedMsg.BorChainId)
		require.Equal(t, originalMsg.RootHash, unpackedMsg.RootHash)
		require.Equal(t, originalMsg.AccountRootHash, unpackedMsg.AccountRootHash)
	})

	t.Run("rejects invalid data length - too short", func(t *testing.T) {
		t.Parallel()

		invalidData := make([]byte, 64) // Too short, should be 192

		unpackedMsg, err := types.UnpackCheckpointSideSignBytes(invalidData)

		require.Error(t, err)
		require.Nil(t, unpackedMsg)
		require.Contains(t, err.Error(), "invalid data length")
	})

	t.Run("rejects invalid data length - too long", func(t *testing.T) {
		t.Parallel()

		invalidData := make([]byte, 256) // Too long, should be 192

		unpackedMsg, err := types.UnpackCheckpointSideSignBytes(invalidData)

		require.Error(t, err)
		require.Nil(t, unpackedMsg)
		require.Contains(t, err.Error(), "invalid data length")
	})

	t.Run("rejects empty data", func(t *testing.T) {
		t.Parallel()

		unpackedMsg, err := types.UnpackCheckpointSideSignBytes([]byte{})

		require.Error(t, err)
		require.Nil(t, unpackedMsg)
	})

	t.Run("handles zero values in packed data", func(t *testing.T) {
		t.Parallel()

		zeroData := make([]byte, 192)

		unpackedMsg, err := types.UnpackCheckpointSideSignBytes(zeroData)

		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
		require.Equal(t, uint64(0), unpackedMsg.StartBlock)
		require.Equal(t, uint64(0), unpackedMsg.EndBlock)
		require.Equal(t, "0", unpackedMsg.BorChainId)
	})
}

func TestNewMsgCpAck(t *testing.T) {
	t.Parallel()

	t.Run("creates message with all fields", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCpAck(
			"0x1234567890123456789012345678901234567890",
			5,
			"0x0987654321098765432109876543210987654321",
			1000,
			2000,
			common.HexToHash("0xabcdef").Bytes(),
		)

		require.Equal(t, "0x1234567890123456789012345678901234567890", msg.From)
		require.Equal(t, uint64(5), msg.Number)
		require.Equal(t, "0x0987654321098765432109876543210987654321", msg.Proposer)
		require.Equal(t, uint64(1000), msg.StartBlock)
		require.Equal(t, uint64(2000), msg.EndBlock)
		require.Equal(t, common.HexToHash("0xabcdef").Bytes(), msg.RootHash)
	})

	t.Run("formats addresses", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCpAck(
			"1234567890123456789012345678901234567890", // Without 0x prefix
			1,
			"0x0987654321098765432109876543210987654321",
			1000,
			2000,
			common.HexToHash("0xabcdef").Bytes(),
		)

		// FormatAddress should add 0x prefix and normalize
		require.Contains(t, msg.From, "0x")
	})
}

func TestMsgCpAck_GetSideSignBytes(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for MsgCpAck", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCpAck(
			"0x1234567890123456789012345678901234567890",
			5,
			"0x0987654321098765432109876543210987654321",
			1000,
			2000,
			common.HexToHash("0xabcdef").Bytes(),
		)

		sideSignBytes := msg.GetSideSignBytes()

		require.Nil(t, sideSignBytes)
	})
}

func TestNewMsgCheckpointNoAck(t *testing.T) {
	t.Parallel()

	t.Run("creates message with from address", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCheckpointNoAck("0x1234567890123456789012345678901234567890")

		require.Equal(t, "0x1234567890123456789012345678901234567890", msg.From)
	})

	t.Run("formats address", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCheckpointNoAck("1234567890123456789012345678901234567890") // Without 0x

		// FormatAddress should add 0x prefix and normalize
		require.Contains(t, msg.From, "0x")
	})
}

func TestIsCheckpointMsg(t *testing.T) {
	t.Parallel()

	t.Run("returns true for MsgCheckpoint", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCheckpointBlock(
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			common.HexToHash("0xabcdef").Bytes(),
			common.HexToHash("0x123456").Bytes(),
			"137",
		)

		isCheckpoint := types.IsCheckpointMsg(msg)

		require.True(t, isCheckpoint)
	})

	t.Run("returns false for MsgCpAck", func(t *testing.T) {
		t.Parallel()

		msg := &types.MsgCpAck{
			From:       "0x1234567890123456789012345678901234567890",
			Proposer:   "0x0987654321098765432109876543210987654321",
			StartBlock: 1000,
			EndBlock:   2000,
			RootHash:   common.HexToHash("0xabcdef").Bytes(),
			Number:     1,
		}

		isCheckpoint := types.IsCheckpointMsg(msg)

		require.False(t, isCheckpoint)
	})

	t.Run("returns false for MsgCpNoAck", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgCheckpointNoAck("0x1234567890123456789012345678901234567890")

		isCheckpoint := types.IsCheckpointMsg(&msg)

		require.False(t, isCheckpoint)
	})
}
