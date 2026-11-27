package processor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestConfirmationTasksSurviveAPIOutage(t *testing.T) {
	var applied atomic.Bool
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !applied.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(req.URL.Path, "/stake/") {
			_, _ = w.Write([]byte(`{"is_old":true}`))
			return
		}
		id := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
		_, _ = fmt.Fprintf(w, `{"record":{"id":%q}}`, id)
	}))
	t.Cleanup(api.Close)
	previous := helper.Client
	helper.Client = api.Client()
	t.Cleanup(func() { helper.Client = previous })
	cfg := helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: helper.GetDefaultHeimdallConfig()}
	cfg.Config.API.Address = api.URL
	helper.SetTestConfig(cfg)
	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)
	stateABI, err := statesender.StatesenderMetaData.GetAbi()
	require.NoError(t, err)
	sp, cp := NewStakingProcessor(stakingABI), NewClerkProcessor(stateABI)
	sp.Logger, cp.Logger = log.NewNopLogger(), log.NewNopLogger()
	sp.cliCtx = client.Context{}.WithCodec(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()))
	cp.cliCtx = sp.cliCtx
	tests := []struct {
		event   string
		payload string
		run     func(context.Context, string, string) error
	}{
		{"StakeUpdate", stakeEventLog(t, stakingABI, "StakeUpdate", 10, 5), sp.sendStakeUpdateToHeimdall},
		{"UnstakeInit", stakeEventLog(t, stakingABI, "UnstakeInit", 10, 5), sp.sendUnstakeInitToHeimdall},
		{"SignerChange", stakeEventLog(t, stakingABI, "SignerChange", 10, 5), sp.sendSignerChangeToHeimdall},
		{"Staked", stakeEventLog(t, stakingABI, "Staked", 10, 5), sp.sendValidatorJoinToHeimdall},
	}
	for _, tc := range tests {
		t.Run(tc.event, func(t *testing.T) {
			signature := &tasks.Signature{RetryCount: 0, Headers: tasks.Headers{eventRetryStartHeader: strconv.FormatInt(time.Now().Add(-48*time.Hour).Unix(), 10)}}
			task, err := tasks.NewWithSignature(tc.run, signature)
			require.NoError(t, err)
			var later tasks.ErrRetryTaskLater
			require.ErrorAs(t, tc.run(task.Context, tc.event, tc.payload), &later)
			require.Positive(t, later.RetryIn())
		})
	}
	applied.Store(true)
	for _, tc := range tests {
		t.Run(tc.event+" confirmed", func(t *testing.T) { require.NoError(t, tc.run(context.Background(), tc.event, tc.payload)) })
		t.Run(tc.event+" malformed", func(t *testing.T) {
			err := tc.run(context.Background(), tc.event, "invalid json")
			require.Error(t, err)
			var later tasks.ErrRetryTaskLater
			require.NotErrorAs(t, err, &later)
		})
	}

}
