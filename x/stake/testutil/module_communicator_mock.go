package testutil

import "context"

type ModuleCommunicatorMock struct {
	AckCount uint64
}

func (m ModuleCommunicatorMock) GetACKCount(ctx context.Context) uint64 {
	return m.AckCount
}
