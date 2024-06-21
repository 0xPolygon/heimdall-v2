package types

import (
	"context"
)

// CheckpointKeeper defines the checkpoint keeper contract used by x/stake module
type CheckpointKeeper interface {
	GetACKCount(ctx context.Context) uint64
}
