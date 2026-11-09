package heimdalld

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	sdklog "cosmossdk.io/log"
	"github.com/0xPolygon/heimdall-v2/peerpolicy"
	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func servingContext(t *testing.T, args []string) (*cobra.Command, *server.Context, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "start"}
	addPeerServingFlags(cmd)
	cmd.Flags().Bool("with-comet", true, "")
	cmd.Flags().Bool("grpc-only", false, "")
	require.NoError(t, cmd.ParseFlags(args))
	v := viper.New()
	require.NoError(t, v.BindPFlags(cmd.Flags()))
	output := &bytes.Buffer{}
	ctx := &server.Context{Config: config.DefaultConfig(), Viper: v, Logger: sdklog.NewLogger(output, sdklog.OutputJSONOption())}
	return cmd, ctx, output
}

func TestPeerServingStartup(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		mode string
	}{
		{"default", nil, "observe"},
		{"scoring_disabled", []string{"--peer-reputation=false"}, "disabled"},
		{"serving_disabled", []string{"--peer-serving=false"}, ""},
		{"external", []string{"--with-comet=false"}, ""},
		{"query", []string{"--grpc-only"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, ctx, output := servingContext(t, tc.args)
			require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
			if tc.mode == "" {
				require.Nil(t, ctx.Config.P2P.ServingPolicy)
				require.Empty(t, output.String())
				return
			}
			g := ctx.Config.P2P.ServingPolicy.(*peerpolicy.Governor)
			const id = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			require.Equal(t, tc.mode, g.Snapshot(id).Mode)
			g.Observe(id, servebudget.InvalidResponse)
			expectedRisk := uint64(60)
			if tc.mode == "disabled" {
				expectedRisk = 0
			}
			require.Equal(t, expectedRisk, g.Snapshot(id).Risk)
			var record map[string]interface{}
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, tc.mode, record["reputation_mode"])
		})
	}
}

func TestPeerServingStartupErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--with-comet=false", "--peer-serving=true"},
		{"--grpc-only", "--peer-serving=true"},
		{"--peer-serving-inflight=0"},
	} {
		t.Run(args[0], func(t *testing.T) {
			cmd, ctx, output := servingContext(t, args)
			require.Error(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
			require.Nil(t, ctx.Config.P2P.ServingPolicy)
			require.Empty(t, output.String())
		})
	}
	t.Run("non_start", func(t *testing.T) {
		cmd, ctx, _ := servingContext(t, nil)
		cmd.Use = "status"
		require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
		require.Nil(t, ctx.Config.P2P.ServingPolicy)
	})
	t.Run("registration", func(t *testing.T) {
		cmd, ctx, _ := servingContext(t, nil)
		registry := prometheus.NewRegistry()
		_, err := peerpolicy.New(peerpolicy.DefaultConfig(), registry)
		require.NoError(t, err)
		require.ErrorContains(t, installPeerServing(cmd, ctx, registry), "configure peer serving")
		require.Nil(t, ctx.Config.P2P.ServingPolicy)
	})
}

func TestPeerServingConfiguredHeadroom(t *testing.T) {
	cmd, ctx, _ := servingContext(t, nil)
	const configured = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const ordinaryA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const ordinaryB = "cccccccccccccccccccccccccccccccccccccccc"
	ctx.Config.P2P.PersistentPeers = configured + "@127.0.0.1:26656"
	require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
	policy := ctx.Config.P2P.ServingPolicy
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 128 << 20}
	for _, id := range []string{ordinaryA, ordinaryB} {
		lease, ok := policy.Admit(id, req)
		require.True(t, ok)
		require.True(t, lease.Prepare(req.MaxBytes))
		lease.Finish(true)
	}
	lease, ok := policy.Admit(configured, req)
	require.True(t, ok, "configured peer must retain reserved headroom")
	if lease != nil {
		lease.Finish(false)
	}
}

func TestPeerServingProtectedIDs(t *testing.T) {
	const a = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const b = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const c = "cccccccccccccccccccccccccccccccccccccccc"
	for _, tc := range []struct {
		name, persistent, private string
		additional                []string
		want                      []string
		fails                     bool
	}{
		{name: "none", want: []string{}},
		{name: "configured", persistent: a + "@127.0.0.1:26656", private: b, want: []string{b, a}},
		{name: "scheme", persistent: "tcp://" + a + "@127.0.0.1:26656", want: []string{a}},
		{name: "scheme_ipv6", persistent: "tcp://" + b + "@[::1]:26656", want: []string{b}},
		{name: "deduplicate", persistent: " " + a + "@node:26656, ", private: " " + a + "," + b, additional: []string{strings.ToUpper(a), c}, want: []string{a, c, b}},
		{name: "missing_id", persistent: "node:26656", fails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ctx, _ := servingContext(t, nil)
			ctx.Config.P2P.PersistentPeers = tc.persistent
			ctx.Config.P2P.PrivatePeerIDs = tc.private
			ids, err := protectedPeerIDs(ctx, tc.additional)
			if tc.fails {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, ids)
		})
	}
}

func TestPeerServingReservationFloors(t *testing.T) {
	for _, tc := range []struct {
		flag    string
		minimum uint64
	}{
		{"peer-serving-memory", 8 * blocksync.MaxMsgSize},
		{"peer-serving-byte-burst", 4 * blocksync.MaxMsgSize},
		{"peer-serving-peer-byte-burst", blocksync.MaxMsgSize},
	} {
		for _, value := range []uint64{tc.minimum - 1, tc.minimum, tc.minimum + 1} {
			t.Run(fmt.Sprintf("%s/%d", tc.flag, value), func(t *testing.T) {
				cmd, ctx, _ := servingContext(t, []string{fmt.Sprintf("--%s=%d", tc.flag, value)})
				id := strings.Repeat("a", 40)
				ctx.Config.P2P.PrivatePeerIDs = id
				err := installPeerServing(cmd, ctx, prometheus.NewRegistry())
				if value < tc.minimum {
					require.ErrorContains(t, err, tc.flag)
					require.Nil(t, ctx.Config.P2P.ServingPolicy)
					return
				}
				require.NoError(t, err)
				lease, admitted := ctx.Config.P2P.ServingPolicy.Admit(id, servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: blocksync.MaxMsgSize})
				require.True(t, admitted)
				lease.Finish(false)
			})
		}
	}
}

func TestPeerServingLargeConfiguredPeerSet(t *testing.T) {
	cmd, ctx, _ := servingContext(t, nil)
	ids := make([]string, 300)
	for i := range ids {
		ids[i] = fmt.Sprintf("%040x", i+1)
	}
	ctx.Config.P2P.PrivatePeerIDs = strings.Join(ids, ",")
	require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
	require.NotNil(t, ctx.Config.P2P.ServingPolicy)
}

func TestPeerServingInvalidConfiguredPrivateID(t *testing.T) {
	cmd, ctx, output := servingContext(t, nil)
	ctx.Config.P2P.PrivatePeerIDs = "invalid"
	require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
	require.Contains(t, output.String(), "Ignoring invalid private peer ID")
	cmd, ctx, _ = servingContext(t, []string{"--peer-serving-protected-ids=invalid"})
	require.Error(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
}
