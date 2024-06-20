package types

import (
	"context"
)

type CheckpointKeeper interface {
	GetACKCount(ctx context.Context) uint64
}
