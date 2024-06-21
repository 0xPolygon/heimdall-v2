package testutil

import "context"

type CheckpointKeeperMock struct {
	AckCount uint64
}

func (m CheckpointKeeperMock) GetACKCount(_ context.Context) uint64 {
	return m.AckCount
}
