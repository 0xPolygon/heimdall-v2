package helper

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
)

func TestGetValidatorInfo(t *testing.T) {
	t.Parallel()

	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)

	oneToken := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	signer := common.HexToAddress("0x00000000000000000000000000000000000000b2")

	tests := []struct {
		name              string
		amount            *big.Int
		deactivationEpoch int64
		rpcErr            bool
		callerCtxDone     bool
		wantErr           bool
	}{
		{name: "staked", amount: new(big.Int).Mul(oneToken, big.NewInt(5))},
		{name: "exit started", amount: new(big.Int).Mul(oneToken, big.NewInt(5)), deactivationEpoch: 9},
		{name: "stake claimed", amount: big.NewInt(0), deactivationEpoch: 9, wantErr: true},
		{name: "main chain unavailable", rpcErr: true, wantErr: true},
		{name: "caller context expired", amount: new(big.Int).Mul(oneToken, big.NewInt(5)), callerCtxDone: true, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
				if tc.rpcErr || req.Method != "eth_call" {
					resp["error"] = map[string]any{"code": -32000, "message": "unavailable"}
				} else {
					out, err := stakingABI.Methods["getStakerDetails"].Outputs.Pack(
						tc.amount, big.NewInt(0), big.NewInt(3), big.NewInt(tc.deactivationEpoch), signer, big.NewInt(1),
					)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					resp["result"] = hexutil.Encode(out)
				}
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client, err := ethclient.Dial(server.URL)
			require.NoError(t, err)
			defer client.Close()

			instance, err := stakinginfo.NewStakinginfo(common.HexToAddress("0xa1"), client)
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.callerCtxDone {
				cancel()
			}

			caller := ContractCaller{MainChainTimeout: 5 * time.Second}
			validator, err := caller.GetValidatorInfo(ctx, 202, instance)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, uint64(202), validator.ValId)
			require.Equal(t, int64(5), validator.VotingPower)
			require.Equal(t, uint64(3), validator.StartEpoch)
			require.Equal(t, uint64(tc.deactivationEpoch), validator.EndEpoch)
			require.Equal(t, signer.String(), validator.Signer)
		})
	}
}
