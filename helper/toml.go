package helper

import (
	"text/template"
)

// Note: any changes to the comments/variables/mapstructure
// must be reflected in the appropriate struct in helper/config.go

const DefaultConfigTemplate = `
[custom]
# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

##### RPC and REST configs #####

# RPC endpoint for ethereum chain
eth_rpc_url = "{{ .Custom.EthRPCUrl }}"

# RPC endpoint for bor chain
bor_rpc_url = "{{ .Custom.BorRPCUrl }}"

# GRPC flag for bor chain
bor_grpc_flag = "{{ .Custom.BorGRPCFlag }}"

# GRPC endpoint for bor chain
bor_grpc_url = "{{ .Custom.BorGRPCUrl }}"

# RPC endpoint for cometBFT
comet_bft_rpc_url = "{{ .Custom.CometBFTRPCUrl }}"

# Polygon Sub Graph URL for self-heal mechanism (optional)
sub_graph_url = "{{ .Custom.SubGraphUrl }}"

#### Bridge configs ####

# AMQP endpoint
amqp_url = "{{ .Custom.AmqpURL }}"

## Poll intervals
checkpoint_poll_interval = "{{ .Custom.CheckpointPollInterval }}"
syncer_poll_interval = "{{ .Custom.SyncerPollInterval }}"
noack_poll_interval = "{{ .Custom.NoACKPollInterval }}"
clerk_poll_interval = "{{ .Custom.ClerkPollInterval }}"
span_poll_interval = "{{ .Custom.SpanPollInterval }}"
milestone_poll_interval = "{{ .Custom.MilestonePollInterval }}"
enable_self_heal = "{{ .Custom.EnableSH }}"
sh_state_synced_interval = "{{ .Custom.SHStateSyncedInterval }}"
sh_stake_update_interval = "{{ .Custom.SHStakeUpdateInterval }}"
sh_checkpoint_ack_interval = "{{ .Custom.SHCheckpointAckInterval }}"
sh_max_depth_duration = "{{ .Custom.SHMaxDepthDuration }}"

#### gas price configs (EIP-1559) ####
main_chain_gas_fee_cap = "{{ .Custom.MainChainGasFeeCap }}"
main_chain_gas_tip_cap = "{{ .Custom.MainChainGasTipCap }}"

##### Timeout Config #####
no_ack_wait_time = "{{ .Custom.NoACKWaitTime }}"

chain = "{{ .Custom.Chain }}"

#### Health check configs ####
# Maximum number of goroutines before heimdall health check fails (0 = disabled)
max_goroutine_threshold = "{{ .Custom.MaxGoRoutineThreshold }}"

# Maximum number of goroutines before heimdall health check warns (0 = disabled)
warn_goroutine_threshold = "{{ .Custom.WarnGoRoutineThreshold }}"

# Minimum number of peers before heimdall health check fails (0 = disabled)
min_peer_threshold = "{{ .Custom.MinPeerThreshold }}"

# Minimum number of peers before heimdall health check warns (0 = disabled)
warn_peer_threshold = "{{ .Custom.WarnPeerThreshold }}"
`

var _ *template.Template

func init() {
	var err error

	tmpl := template.New("appConfigFileTemplate")
	if _, err = tmpl.Parse(DefaultConfigTemplate); err != nil {
		panic(err)
	}
}
