#!/bin/bash
handle_error() {
    local step_number=$1
    local message=$2
    echo -e "\n[ERROR] Step $step_number failed: $message"
    exit 1
}

print_step() {
    echo ""
    local step_number=$1
    local message=$2
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "\n[$timestamp] [STEP $step_number] $message"
}

echo "Creating required directory for output"
sudo mkdir -p config/heimdallv2/config/
sudo mkdir -p data/heimdallv2/
STEP=1
print_step $STEP "UPDATE priv_validator_key.json"
PRIV_VALIDATOR_FILE="config/priv_validator_key.json"
TEMP_PRIV_FILE="temp_priv_validator_key.json"
TEMP_FILES+=("$TEMP_PRIV_FILE")

if [ -f "$PRIV_VALIDATOR_FILE" ]; then
    echo "[INFO] Creating backup of priv_validator_key.json..."
    sudo cp "$PRIV_VALIDATOR_FILE" "$PRIV_VALIDATOR_FILE.bak" || handle_error $STEP "Failed to backup priv_validator_key.json"
    echo "[INFO] Backup saved at: $PRIV_VALIDATOR_FILE.bak"
else
    handle_error $STEP "priv_validator_key.json not found in Heimdall config directory!"
fi
ADDRESS=$(jq -r '.address' "config/priv_validator_key.json")
PUB_KEY_VALUE=$(jq -r '.pub_key.value' "config/priv_validator_key.json")
PRIV_KEY_VALUE=$(jq -r '.priv_key.value' "config/priv_validator_key.json")
if jq --arg addr "$ADDRESS" \
      --arg pub "$PUB_KEY_VALUE" \
      --arg priv "$PRIV_KEY_VALUE" \
      '.address = $addr | .pub_key.value = $pub | .priv_key.value = $priv' \
      "$PRIV_VALIDATOR_FILE" > "$TEMP_PRIV_FILE"; then
    if [[ ! -s "$TEMP_PRIV_FILE" ]]; then
        handle_error $STEP "Updated priv_validator_key.json is empty or invalid!"
    fi
    mv "$TEMP_PRIV_FILE" "config/heimdallv2/$PRIV_VALIDATOR_FILE" || handle_error $STEP "Failed to move updated priv_validator_key.json into place"
else
    handle_error $STEP "Failed to update priv_validator_key.json"
fi
echo "[INFO] Updated priv_validator_key.json file saved as $PRIV_VALIDATOR_FILE"


STEP=2
print_step $STEP "UPDATE node_key.json"
NODE_KEY_FILE="config/node_key.json"
TEMP_NODE_KEY_FILE="temp_node_key.json"
TEMP_FILES+=("$TEMP_NODE_KEY_FILE")
if [ -f "$NODE_KEY_FILE" ]; then
    echo "[INFO] Creating backup of node_key.json..."
    cp "$NODE_KEY_FILE" "$NODE_KEY_FILE.bak" || handle_error $STEP "Failed to backup node_key.json"
    echo "[INFO] Backup saved at: $NODE_KEY_FILE.bak"
else
    handle_error $STEP "node_key.json not found in Heimdall config directory!"
fi
NODE_KEY=$(jq -r '.priv_key.value' "config/node_key.json") || handle_error $STEP "Failed to extract priv_key.value from backup node_key.json"
if jq --arg nodekey "$NODE_KEY" \
      '.priv_key.value = $nodekey' \
      "$NODE_KEY_FILE" > "$TEMP_NODE_KEY_FILE"; then
    if [[ ! -s "$TEMP_NODE_KEY_FILE" ]]; then
        handle_error $STEP "Updated node_key.json is empty or invalid!"
    fi
    mv "$TEMP_NODE_KEY_FILE" "config/heimdallv2/$NODE_KEY_FILE" || handle_error $STEP "Failed to move updated node_key.json into place"
else
    handle_error $STEP "Failed to update node_key.json"
fi
echo "[INFO] Updated node_key.json file saved as $NODE_KEY_FILE"


STEP=3
print_step $STEP "UPDATE priv_validator_state.json"
PRIV_VALIDATOR_STATE="data/priv_validator_state.json"
TEMP_STATE_FILE="temp_priv_validator_state.json"
TEMP_FILES+=("$TEMP_STATE_FILE")
if [ ! -f "$PRIV_VALIDATOR_STATE" ]; then
    handle_error $STEP "priv_validator_state.json not found in data/"
fi
echo "[INFO] Creating backup of priv_validator_state.json..."
cp "$PRIV_VALIDATOR_STATE" "$PRIV_VALIDATOR_STATE.bak" || handle_error $STEP "Failed to backup priv_validator_state.json"
echo "[INFO] Backup saved at: $PRIV_VALIDATOR_STATE.bak"
# Validate the file has proper JSON
jq empty "$PRIV_VALIDATOR_STATE" || handle_error $STEP "Invalid JSON detected in priv_validator_state.json"
# Apply transformations:
#   1. Convert "round" from string to int
#   2. Set "height" to string value of $V2_INITIAL_HEIGHT
if jq --arg height "$V2_INITIAL_HEIGHT" '.round |= tonumber | .height = $height' "$PRIV_VALIDATOR_STATE" > "$TEMP_STATE_FILE"; then
    if [[ ! -s "$TEMP_STATE_FILE" ]]; then
        handle_error $STEP "Updated priv_validator_state.json is empty or invalid!"
    fi
    mv "$TEMP_STATE_FILE" "data/heimdallv2/$PRIV_VALIDATOR_STATE" || handle_error $STEP "Failed to move updated priv_validator_state.json into place"
else
    handle_error $STEP "Failed to update priv_validator_state.json"
fi
echo "[INFO] Successfully updated priv_validator_state.json"

STEP=4
print_step $STEP "V1 -> V2 CONFIG PORTING"
# 1. Set chain-id in client.toml
CLIENT_TOML="config/client.toml"
echo "[INFO] Setting chain-id in client.toml..."
set_toml_key "$CLIENT_TOML" "chain-id" "$V2_CHAIN_ID"
actual_chain_id=$(grep -E '^chain-id\s*=' "$CLIENT_TOML" | cut -d'=' -f2 | tr -d ' "')
if [[ "$actual_chain_id" != "$V2_CHAIN_ID" ]]; then
    echo "[WARN] Validation failed: expected chain-id = $V2_CHAIN_ID, found $actual_chain_id"
fi
echo "[OK]   client.toml: chain-id = $V2_CHAIN_ID"
# 2. Migrate config.toml keys
OLD_CONFIG_TOML="config/config.toml"
NEW_CONFIG_TOML="config/heimdallv2/config.toml"
CONFIG_KEYS=(
    "moniker"
    "external_address"
    "seeds"
    "persistent_peers"
    "max_num_inbound_peers"
    "max_num_outbound_peers"
    "proxy_app"
    "addr_book_strict"
)
echo "[INFO] Copying selected values from v1 config.toml to v2..."
for key in "${CONFIG_KEYS[@]}"; do
    value=$(grep -E "^$key\s*=" "$OLD_CONFIG_TOML" | cut -d'=' -f2- | sed 's/^ *//;s/^"//;s/"$//' || true)
    if [[ -n "$value" ]]; then
        set_toml_key "$NEW_CONFIG_TOML" "$key" "$value"
        echo "[OK]   config.toml: $key = $value"
    else
        echo "[WARN] config.toml: key '$key' not found or empty in v1, skipping"
    fi
done
for key in "${CONFIG_KEYS[@]}"; do
    expected=$(grep -E "^$key\s*=" "$OLD_CONFIG_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    actual=$(grep -E "^$key\s*=" "$NEW_CONFIG_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    if [[ "$expected" != "$actual" ]]; then
        echo "[WARN] Validation failed for '$key' in config.toml: expected '$expected', got '$actual'"
    fi
done
echo "[INFO] config.toml values migrated successfully."
# 3. Set static log parameters in config.toml
echo "[INFO] Setting static logging parameters in config.toml..."
set_toml_key "$NEW_CONFIG_TOML" "log_level" "info"
set_toml_key "$NEW_CONFIG_TOML" "log_format" "plain"
echo "[OK]   config.toml: log_level = info"
echo "[OK]   config.toml: log_format = plain"
# 4. Migrate heimdall-config.toml → app.toml
OLD_HEIMDALL_CONFIG_TOML="config/heimdall-config.toml"
NEW_APP_TOML="config/heimdallv2/app.toml"
APP_KEYS=(
    "eth_rpc_url"
    "bor_rpc_url"
    "bor_grpc_flag"
    "bor_grpc_url"
    "amqp_url"
)
echo "[INFO] Copying selected values from v1 heimdall-config.toml to app.toml..."
for key in "${APP_KEYS[@]}"; do
    value=$(grep -E "^$key\s*=" "$OLD_HEIMDALL_CONFIG_TOML" | cut -d'=' -f2- | sed 's/^ *//;s/^"//;s/"$//' || true)
    if [[ -n "$value" ]]; then
        set_toml_key "$NEW_APP_TOML" "$key" "$value"
        echo "[OK]   app.toml: $key = $value"
    else
        echo "[WARN] app.toml: key '$key' not found or empty in v1, skipping"
    fi
done
# 5. Validate
for key in "${APP_KEYS[@]}"; do
    expected=$(grep -E "^$key\s*=" "$OLD_HEIMDALL_CONFIG_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    actual=$(grep -E "^$key\s*=" "$NEW_APP_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    if [[ "$expected" != "$actual" ]]; then
        echo "[WARN] Validation failed for '$key' in app.toml: expected '$expected', got '$actual'"
    fi
done
# 6. Set static bor_grpc_flag=false in app.toml (recommended for the migration)
echo "[INFO] Setting bor_grpc_flag param to false in app.toml..."
set_toml_key "app.toml" "bor_grpc_flag" "false"
echo "[OK]   app.toml: bor_grpc_flag = false"
echo "[INFO] app.toml values migrated and updated successfully."
# 6. Set static bor_rpc_timeout=1s in app.toml (recommended for the migration)
echo "[INFO] Setting bor_rpc_timeout param to 1s in config.toml..."
set_toml_key "app.toml" "bor_rpc_timeout" "1s"
echo "[OK]   app.toml: bor_rpc_timeout = 1s"
echo "[INFO] app.toml values migrated and updated successfully."