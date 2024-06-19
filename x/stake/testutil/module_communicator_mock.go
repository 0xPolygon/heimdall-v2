package testutil

import "context"

type CheckpointKeeperMock struct {
	AckCount uint64
}

func (m CheckpointKeeperMock) GetACKCount(ctx context.Context) uint64 {
	return m.AckCount
}
