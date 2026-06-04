package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBorFailover_AllHooksWired(t *testing.T) {
	m := BorFailover("unit")

	require.NotNil(t, m.Switch)
	require.NotNil(t, m.ProactiveSwitch)
	require.NotNil(t, m.ActiveIndex)
	require.NotNil(t, m.HealthyCount)

	require.NotPanics(t, func() {
		m.Switch()
		m.ProactiveSwitch()
		m.ActiveIndex(1)
		m.HealthyCount(2)
	})
}
