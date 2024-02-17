package types

import (
	"context"
)

type ModuleCommunicator interface {
	GetACKCount(ctx context.Context) uint64
}
