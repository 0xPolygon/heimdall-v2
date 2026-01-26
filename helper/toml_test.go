package helper_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestDefaultConfigTemplate_NotEmpty(t *testing.T) {
	require.NotEmpty(t, helper.DefaultConfigTemplate)
}

func TestDefaultConfigTemplate_ContainsRequiredSections(t *testing.T) {
	// Test that the template contains all expected configuration sections
	requiredSections := []string{
		"[custom]",
		"eth_rpc_url",
		"bor_rpc_url",
		"bor_grpc_flag",
		"bor_grpc_url",
		"comet_bft_rpc_url",
		"sub_graph_url",
		"amqp_url",
		"checkpoint_poll_interval",
		"syncer_poll_interval",
		"noack_poll_interval",
		"clerk_poll_interval",
		"span_poll_interval",
		"milestone_poll_interval",
		"enable_self_heal",
		"sh_state_synced_interval",
		"sh_stake_update_interval",
		"sh_checkpoint_ack_interval",
		"sh_max_depth_duration",
		"main_chain_gas_fee_cap",
		"main_chain_gas_tip_cap",
		"no_ack_wait_time",
		"chain",
		"max_goroutine_threshold",
		"warn_goroutine_threshold",
		"min_peer_threshold",
		"warn_peer_threshold",
	}

	for _, section := range requiredSections {
		require.Contains(t, helper.DefaultConfigTemplate, section,
			"template should contain section: %s", section)
	}
}

func TestDefaultConfigTemplate_ContainsTemplateVariables(t *testing.T) {
	// Test that template variables are properly formatted
	templateVariables := []string{
		"{{ .Custom.EthRPCUrl }}",
		"{{ .Custom.BorRPCUrl }}",
		"{{ .Custom.BorGRPCFlag }}",
		"{{ .Custom.BorGRPCUrl }}",
		"{{ .Custom.CometBFTRPCUrl }}",
		"{{ .Custom.SubGraphUrl }}",
		"{{ .Custom.AmqpURL }}",
		"{{ .Custom.CheckpointPollInterval }}",
		"{{ .Custom.SyncerPollInterval }}",
		"{{ .Custom.NoACKPollInterval }}",
		"{{ .Custom.ClerkPollInterval }}",
		"{{ .Custom.SpanPollInterval }}",
		"{{ .Custom.MilestonePollInterval }}",
		"{{ .Custom.EnableSH }}",
		"{{ .Custom.SHStateSyncedInterval }}",
		"{{ .Custom.SHStakeUpdateInterval }}",
		"{{ .Custom.SHCheckpointAckInterval }}",
		"{{ .Custom.SHMaxDepthDuration }}",
		"{{ .Custom.MainChainGasFeeCap }}",
		"{{ .Custom.MainChainGasTipCap }}",
		"{{ .Custom.NoACKWaitTime }}",
		"{{ .Custom.Chain }}",
		"{{ .Custom.MaxGoRoutineThreshold }}",
		"{{ .Custom.WarnGoRoutineThreshold }}",
		"{{ .Custom.MinPeerThreshold }}",
		"{{ .Custom.WarnPeerThreshold }}",
	}

	for _, variable := range templateVariables {
		require.Contains(t, helper.DefaultConfigTemplate, variable,
			"template should contain variable: %s", variable)
	}
}

func TestDefaultConfigTemplate_IsTOMLFormat(t *testing.T) {
	// Test that the template is in TOML format
	require.Contains(t, helper.DefaultConfigTemplate, "[custom]")
	require.Contains(t, helper.DefaultConfigTemplate, "This is a TOML config file")
}

func TestDefaultConfigTemplate_ContainsRPCConfigs(t *testing.T) {
	// Test RPC and REST configuration section
	require.Contains(t, helper.DefaultConfigTemplate, "##### RPC and REST configs #####")
	require.Contains(t, helper.DefaultConfigTemplate, "eth_rpc_url")
	require.Contains(t, helper.DefaultConfigTemplate, "bor_rpc_url")
	require.Contains(t, helper.DefaultConfigTemplate, "comet_bft_rpc_url")
}

func TestDefaultConfigTemplate_ContainsBridgeConfigs(t *testing.T) {
	// Test Bridge configuration section
	require.Contains(t, helper.DefaultConfigTemplate, "#### Bridge configs ####")
	require.Contains(t, helper.DefaultConfigTemplate, "amqp_url")
	require.Contains(t, helper.DefaultConfigTemplate, "checkpoint_poll_interval")
	require.Contains(t, helper.DefaultConfigTemplate, "syncer_poll_interval")
}

func TestDefaultConfigTemplate_ContainsGasConfigs(t *testing.T) {
	// Test gas configuration section
	require.Contains(t, helper.DefaultConfigTemplate, "#### gas price configs (EIP-1559) ####")
	require.Contains(t, helper.DefaultConfigTemplate, "main_chain_gas_fee_cap")
	require.Contains(t, helper.DefaultConfigTemplate, "main_chain_gas_tip_cap")
}

func TestDefaultConfigTemplate_ContainsHealthCheckConfigs(t *testing.T) {
	// Test health check configuration section
	require.Contains(t, helper.DefaultConfigTemplate, "#### Health check configs ####")
	require.Contains(t, helper.DefaultConfigTemplate, "max_goroutine_threshold")
	require.Contains(t, helper.DefaultConfigTemplate, "warn_goroutine_threshold")
	require.Contains(t, helper.DefaultConfigTemplate, "min_peer_threshold")
	require.Contains(t, helper.DefaultConfigTemplate, "warn_peer_threshold")
}

func TestDefaultConfigTemplate_ContainsSelfHealConfigs(t *testing.T) {
	// Test self-heal configuration section
	require.Contains(t, helper.DefaultConfigTemplate, "enable_self_heal")
	require.Contains(t, helper.DefaultConfigTemplate, "sh_state_synced_interval")
	require.Contains(t, helper.DefaultConfigTemplate, "sh_stake_update_interval")
	require.Contains(t, helper.DefaultConfigTemplate, "sh_checkpoint_ack_interval")
	require.Contains(t, helper.DefaultConfigTemplate, "sh_max_depth_duration")
}

func TestDefaultConfigTemplate_NoTrailingWhitespace(t *testing.T) {
	// Test that template lines don't have trailing whitespace
	lines := strings.Split(helper.DefaultConfigTemplate, "\n")

	for i, line := range lines {
		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		trimmed := strings.TrimRight(line, " \t")
		require.Equal(t, trimmed, line,
			"line %d should not have trailing whitespace", i+1)
	}
}

func TestDefaultConfigTemplate_ContainsTOMLLink(t *testing.T) {
	// Test that the template has a reference to TOML documentation
	require.Contains(t, helper.DefaultConfigTemplate, "https://github.com/toml-lang/toml")
}

func TestDefaultConfigTemplate_AllVariablesUnderCustom(t *testing.T) {
	// Test that all template variables reference .Custom
	lines := strings.Split(helper.DefaultConfigTemplate, "\n")

	for _, line := range lines {
		if strings.Contains(line, "{{") && strings.Contains(line, "}}") {
			// If line contains template variable, it should reference .Custom
			require.Contains(t, line, ".Custom.",
				"template variable should be under .Custom: %s", line)
		}
	}
}

func TestDefaultConfigTemplate_ValidFormatting(t *testing.T) {
	// Test that the template has consistent formatting
	// All key-value pairs should use " = " format
	lines := strings.Split(helper.DefaultConfigTemplate, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comments and section headers
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Should contain " = " separator
		if !strings.Contains(trimmed, "=") {
			continue // Skip lines without assignments
		}

		parts := strings.Split(trimmed, "=")
		require.Len(t, parts, 2, "config line should have key = value format: %s", trimmed)
	}
}

func TestDefaultConfigTemplate_ChainVariable(t *testing.T) {
	// Test that the chain variable is present
	require.Contains(t, helper.DefaultConfigTemplate, "chain =")
	require.Contains(t, helper.DefaultConfigTemplate, "{{ .Custom.Chain }}")
}

func TestDefaultConfigTemplate_TimeoutConfigs(t *testing.T) {
	// Test timeout configuration section
	require.Contains(t, helper.DefaultConfigTemplate, "##### Timeout Config #####")
	require.Contains(t, helper.DefaultConfigTemplate, "no_ack_wait_time")
}
