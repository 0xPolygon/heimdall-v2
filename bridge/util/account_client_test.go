package util

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestAccountClientFollowsConfiguredURL(t *testing.T) {
	// Not parallel: points the global helper config at different URLs.
	setURL := func(url string) {
		conf := helper.GetDefaultHeimdallConfig()
		conf.CometBFTRPCUrl = url
		helper.SetTestConfig(helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: conf})
	}

	setURL("http://127.0.0.1:26657")
	first, err := accountClient()
	require.NoError(t, err)
	again, err := accountClient()
	require.NoError(t, err)
	require.Same(t, first, again, "same URL reuses the client")

	setURL("http://127.0.0.1:36657")
	moved, err := accountClient()
	require.NoError(t, err)
	require.NotSame(t, first, moved, "a new URL builds a new client")

	setURL("http://%zz")
	broken, err := accountClient()
	require.Error(t, err)
	require.Nil(t, broken)
	account, err := GetAccount(context.Background(), client.Context{}, "0x00000000000000000000000000000000000000aa")
	require.Error(t, err)
	require.Nil(t, account)

	setURL("http://127.0.0.1:36657")
	recovered, err := accountClient()
	require.NoError(t, err)
	require.Same(t, moved, recovered, "a failed rebuild keeps the last good client")
}
