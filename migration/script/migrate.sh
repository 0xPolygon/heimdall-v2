#!/bin/bash

umask 0022

# -------------------- Env variables, to be adjusted before rolling out --------------------
APOCALYPSE_TAG="1.2.3-27-g74c8af58"
REQUIRED_BOR_VERSION="2.0.0"
CHECKSUM="07d8634fd2c14bf3ad1b1f6f6646ee632b9279339c8f77ccc1cea0f2e64b389a97d2c443f42e345210be59a58e574bdfb4e425e8e998f83dd8383239b031dd03"
MIGRATED_CHECKSUM="bf7a2a4b99c96eaa4246c1932bfdae28a821b6f90b68209ccbc1da49d5689e28f8bbd6433939523d5d362551d6baa56d1c448a178fc8ee82061177e3b7539060"
HEIMDALL_V2_VERSION="0.1.15"
CHAIN_ID="devnet"
GENESIS_TIME="2025-05-15T14:15:00Z"
APOCALYPSE_HEIGHT=200
INITIAL_HEIGHT=$(( APOCALYPSE_HEIGHT + 1 ))
VERIFY_DATA=true
DUMP_V1_GENESIS_FILE_NAME="dump-genesis.json"
DRY_RUN=false
TRUSTED_GENESIS_URL="https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/e2e-test/migration/networks/devnet/dump-genesis.json"

START_TIME=$(date +%s)
SCRIPT_PATH=$(realpath "$0")

if ! tail -n 10 "$SCRIPT_PATH" | grep -q "# End of script"; then
  echo "[ERROR] Script appears to be incomplete or partially downloaded."
  exit 1
fi

if [[ "$(id -u)" -ne 0 ]]; then
  echo "[ERROR] This script must be run as root. Use sudo."
  exit 1
fi

# CLI-provided values
HEIMDALL_HOME=""
HEIMDALL_CLI_PATH=""
HEIMDALLD_PATH=""
BOR_PATH=""
NETWORK=""
NODETYPE=""
BACKUP_DIR=""
MONIKER_NODE_NAME=""
HEIMDALL_SERVICE_USER=""
GENERATE_GENESIS=""

show_help() {
  echo "Usage: sudo bash migrate.sh --heimdall-home=<PATH_TO_HEIMDALL_HOME> --cli-path=<PATH_TO_HEIMDALLCLI_BINARY> --d-path=<PATH_TO_HEIMDALLD_BINARY> \\
            --network=mainnet|amoy --nodetype=sentry|validator \\
            --backup-dir=<PATH_TO_BACKUP_DIR> --moniker=<MONIKER_STRING> --service-user=<HEIMDALL_SERVICE_USER> \\
            --generate-genesis=true|false [--bor-path=<PATH_TO_BOR_BINARY>]"
  echo "Required arguments:"
  echo "  --heimdall-home=PATH          Absolute path to Heimdall home directory (must contain 'config' and 'data')"
  echo "  --cli-path=PATH               Path to the heimdallcli binary (must be >= v1.0.10)"
  echo "  --d-path=PATH                 Path to the heimdalld binary (must be apocalypse tag: 1.2.0-41-*)"
  echo "  --network=mainnet|amoy       Network this node is part of (use 'mainnet' or 'amoy')"
  echo "  --nodetype=sentry|validator  Whether this node is a sentry or validator"
  echo "  --backup-dir=PATH            Directory where a backup of Heimdall v1 will be stored"
  echo "  --moniker=NAME               The node's moniker (must match 'moniker' in config.toml)"
  echo "  --service-user=USER          System user that runs the Heimdall service"
  echo "                                (typically 'heimdall'; check systemd with 'systemctl status heimdalld')"
  echo "  --generate-genesis=true|false Whether to export genesis from heimdalld (recommended: true)"
  echo "Optional arguments:"
  echo "  --bor-path=PATH              Path to 'bor' binary (only needed if Bor runs on the same machine)"
  echo "Example:"
  echo "  sudo bash migrate.sh \\
    --heimdall-home=/var/lib/heimdall \\
    --cli-path=/usr/bin/heimdallcli \\
    --d-path=/usr/bin/heimdalld \\
    --network=mainnet \\
    --nodetype=validator \\
    --backup-dir=/var/lib/heimdall.backup \\
    --moniker=my-node-01 \\
    --service-user=heimdall \\
    --generate-genesis=true"
  exit 0
}

# Parse args
for arg in "$@"; do
  case $arg in
    --heimdall-home=*) HEIMDALL_HOME="${arg#*=}" ;;
    --cli-path=*) HEIMDALL_CLI_PATH="${arg#*=}" ;;
    --d-path=*) HEIMDALLD_PATH="${arg#*=}" ;;
    --bor-path=*) BOR_PATH="${arg#*=}" ;;
    --network=*) NETWORK="${arg#*=}" ;;
    --nodetype=*) NODETYPE="${arg#*=}" ;;
    --backup-dir=*) BACKUP_DIR="${arg#*=}" ;;
    --moniker=*) MONIKER_NODE_NAME="${arg#*=}" ;;
    --service-user=*) HEIMDALL_SERVICE_USER="${arg#*=}" ;;
    --generate-genesis=*) GENERATE_GENESIS="${arg#*=}" ;;
    --help|-h) show_help ;;
    *) echo "[ERROR] Unknown argument: $arg"; exit 1 ;;
  esac
  shift || true
  done

# Check required
missing_args=()
[[ -z "$HEIMDALL_HOME" ]] && missing_args+=("--heimdall-home")
[[ -z "$HEIMDALL_CLI_PATH" ]] && missing_args+=("--cli-path")
[[ -z "$HEIMDALLD_PATH" ]] && missing_args+=("--d-path")
[[ -z "$NETWORK" ]] && missing_args+=("--network")
[[ -z "$NODETYPE" ]] && missing_args+=("--nodetype")
[[ -z "$BACKUP_DIR" ]] && missing_args+=("--backup-dir")
[[ -z "$MONIKER_NODE_NAME" ]] && missing_args+=("--moniker")
[[ -z "$HEIMDALL_SERVICE_USER" ]] && missing_args+=("--service-user")
[[ -z "$GENERATE_GENESIS" ]] && missing_args+=("--generate-genesis")
if (( ${#missing_args[@]} > 0 )); then
  echo "[ERROR] Missing required arguments: ${missing_args[*]}"
  show_help
fi


# Initialize rollback actions
ROLLBACK_ACTIONS=()
# Track temp files to clean up on exit
TEMP_FILES=()
# Init last executed step
LAST_STEP_EXECUTED=0

# Function to print step information
print_step() {
    echo ""
    local step_number=$1
    local message=$2
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "\n[$timestamp] [STEP $step_number] $message"
    LAST_STEP_EXECUTED=$step_number
}

# Function to handle errors
handle_error() {
    local step_number=$1
    local message=$2
    echo -e "\n[ERROR] Step $step_number failed: $message"
    #rollback
    exit 1
}

# Function to rollback executed steps
rollback() {
    echo -e "\n[ROLLBACK] Reverting changes from step $LAST_STEP_EXECUTED down to step 1..."
    for (( i=LAST_STEP_EXECUTED; i>=1; i-- )); do
        if [[ -n "${ROLLBACK_ACTIONS[i]}" && "${ROLLBACK_ACTIONS[i]}" != ":" ]]; then
            echo "[ROLLBACK] Executing rollback for Step $i: ${ROLLBACK_ACTIONS[i]}"
            eval "${ROLLBACK_ACTIONS[i]}"
        else
            echo "[ROLLBACK] Step $i has no rollback action. Skipping."
        fi
    done
    echo "[ROLLBACK] Completed."
}

# Function to clean up temp files on script exit
cleanup_temp_files() {
    for f in "${TEMP_FILES[@]}"; do
        [[ -f "$f" ]] && rm -f "$f"
    done
}
trap cleanup_temp_files EXIT

# Function to validate absolute paths for user input
validate_absolute_path() {
    local path=$1
    local name=$2
    if [[ ! "$path" =~ ^/ ]]; then
        handle_error $STEP "$name must be an absolute path."
    fi
}

# Function to compare versions
version_ge() {
    [[ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" == "$2" ]]
}

# Normalize versions: strip leading 'v' if present
normalize_version() {
  local raw="$1"
  echo "${raw#v}"  # removes leading 'v' if it exists
}

# Helper to set or insert a key=value pair in a TOML file (top-level only)
set_toml_key() {
  local file="$1"
  local key="$2"
  local value="$3"
  local escaped_value

  # Escape double quotes in value
  escaped_value=$(printf '%s' "$value" | sed 's/"/\\"/g')

  if grep -qE "^$key\s*=" "$file"; then
    sed -i "s|^$key\s*=.*|$key = \"$escaped_value\"|" "$file"
  else
    echo "$key = \"$escaped_value\"" >> "$file"
  fi
}


# ------------------ Welcome Message ------------------
echo ""
echo -e "🔄  Welcome to the Polygon Heimdall v1 → v2 Migration Script 🔄"
echo -e "----------------------------------------------------------"
echo -e "This script will execute a multi-step migration"
echo -e "from Heimdall v1 to Heimdall v2, including:"
echo -e "  ✅ Exporting and verifying the v1 state"
echo -e "  ✅ Installing Heimdall v2 binaries"
echo -e "  ✅ Initializing and migrating configurations"
echo -e "  ✅ Updating genesis and validator keys"
echo -e "  ✅ Applying permissions and restarting the node"
echo -e "----------------------------------------------------------\n"
sleep 3


# Step 1: Check script dependencies
STEP=1
print_step $STEP "Checking for required dependencies"
ROLLBACK_ACTIONS["$STEP"]=":"
# Define base and new dependencies
DEPENDENCIES=("curl" "tar" "jq" "sha512sum")
MISSING_DEPS=()
# Check if commands are available
for dep in "${DEPENDENCIES[@]}"; do
    if ! command -v "$dep" &> /dev/null; then
        MISSING_DEPS+=("$dep")
    fi
done
# Fail if missing
if (( ${#MISSING_DEPS[@]} > 0 )); then
    handle_error $STEP "Missing dependencies: ${MISSING_DEPS[*]@Q}. Please install them and rerun the script."
fi
echo "[INFO] All required dependencies are installed."


# Step 2: Validate provided arguments
STEP=2
print_step $STEP "Validating provided arguments"
ROLLBACK_ACTIONS["$STEP"]=":"
# HEIMDALL_HOME
validate_absolute_path "$HEIMDALL_HOME" "HEIMDALL_HOME"
if [[ ! -d "$HEIMDALL_HOME/data" || ! -d "$HEIMDALL_HOME/config" ]]; then
    handle_error $STEP "Required directories (data, config) are missing in HEIMDALL_HOME: $HEIMDALL_HOME"
fi
# HEIMDALL_CLI_PATH
validate_absolute_path "$HEIMDALL_CLI_PATH" "HEIMDALL_CLI_PATH"
HEIMDALLCLI_VERSION=$("$HEIMDALL_CLI_PATH" version 2>/dev/null)
if [[ -z "$HEIMDALLCLI_VERSION" ]]; then
    handle_error $STEP "HEIMDALLCLI_PATH is invalid or heimdallcli is not executable."
fi
# Compare heimdallcli version
if [[ "$DRY_RUN" != "true" ]]; then
  NORMALIZED_HEIMDALLCLI_VERSION=$(normalize_version "$HEIMDALLCLI_VERSION")
  NORMALIZED_EXPECTED_VERSION=$(normalize_version "$APOCALYPSE_TAG")

  if [[ "$NORMALIZED_HEIMDALLCLI_VERSION" != "$NORMALIZED_EXPECTED_VERSION" ]]; then
    handle_error $STEP "heimdallcli version mismatch! Expected: $APOCALYPSE_TAG, Found: $HEIMDALLCLI_VERSION"
  fi
fi
# Validate heimdalld path and version
validate_absolute_path "$HEIMDALLD_PATH" "HEIMDALLD_PATH"
HEIMDALLD_VERSION=$("$HEIMDALLD_PATH" version 2>/dev/null)
if [[ -z "$HEIMDALLD_VERSION" ]]; then
    handle_error $STEP "HEIMDALLD_PATH is invalid or heimdalld is not executable."
fi

if [[ "$DRY_RUN" != "true" ]]; then
  NORMALIZED_HEIMDALLD_VERSION=$(normalize_version "$HEIMDALLD_VERSION")
  if [[ "$NORMALIZED_HEIMDALLD_VERSION" != "$NORMALIZED_EXPECTED_VERSION" ]]; then
    handle_error $STEP "heimdalld version mismatch! Expected: $APOCALYPSE_TAG, Found: $HEIMDALLD_VERSION"
  fi
fi
# BOR_PATH (optional)
if [[ -n "$BOR_PATH" ]]; then
    validate_absolute_path "$BOR_PATH" "BOR_PATH"
    if [[ "$DRY_RUN" != "true" ]]; then
        RAW_BOR_VERSION=$("$BOR_PATH" version 2>/dev/null)
        # Try to extract the version:
        # 1. If it's in "Version: x.y.z" format (multi-line), useful for commits/branches on unreleased versions
        # 2. Else fallback to assuming RAW_BOR_VERSION is the version string directly
        if echo "$RAW_BOR_VERSION" | grep -qi '^Version:'; then
            BOR_VERSION=$(echo "$RAW_BOR_VERSION" | grep -i '^Version:' | awk '{print $2}')
        else
            # Fallback: take first word in raw output
            BOR_VERSION=$(echo "$RAW_BOR_VERSION" | awk '{print $1}')
        fi
        if [[ -z "$BOR_VERSION" ]]; then
            handle_error $STEP "Could not parse bor version. Output: $RAW_BOR_VERSION"
        fi
        # Normalize both expected and actual versions (strip leading 'v')
        NORMALIZED_BOR_VERSION=$(normalize_version "$BOR_VERSION")
        NORMALIZED_REQUIRED_BOR_VERSION=$(normalize_version "$REQUIRED_BOR_VERSION")

        if [[ "$NORMALIZED_BOR_VERSION" != "$NORMALIZED_REQUIRED_BOR_VERSION" ]]; then
            handle_error $STEP "bor version mismatch! Expected: $REQUIRED_BOR_VERSION, Found: $BOR_VERSION"
        fi
    fi
fi
# NETWORK
if [[ "$NETWORK" != "amoy" && "$NETWORK" != "mainnet" ]]; then
    handle_error $STEP "Invalid network! Must be 'amoy' or 'mainnet'."
fi
# NODETYPE
if [[ "$NODETYPE" != "sentry" && "$NODETYPE" != "validator" ]]; then
    handle_error $STEP "Invalid node type! Must be 'sentry' or 'validator'."
fi
# BACKUP_DIR
validate_absolute_path "$BACKUP_DIR" "BACKUP_DIR"
if [[ "$BACKUP_DIR" == "$HEIMDALL_HOME" ]]; then
    handle_error $STEP "Invalid backup directory, it can't be equal to HEIMDALL_HOME."
fi
# MONIKER_NODE_NAME
if [[ -z "$MONIKER_NODE_NAME" ]]; then
    handle_error $STEP "MONIKER_NODE_NAME cannot be empty."
fi
# HEIMDALL_SERVICE_USER
if [[ -z "$HEIMDALL_SERVICE_USER" ]]; then
    handle_error $STEP "HEIMDALL_SERVICE_USER cannot be empty."
fi
if ! id "$HEIMDALL_SERVICE_USER" &>/dev/null; then
    handle_error $STEP "User '$HEIMDALL_SERVICE_USER' does not exist on this system."
fi
# GENERATE_GENESIS
if [[ "$GENERATE_GENESIS" != "true" && "$GENERATE_GENESIS" != "false" ]]; then
    handle_error $STEP "Invalid value for --generate-genesis. Must be 'true' or 'false'."
fi
# Summary
echo ""
echo "[INFO] Configuration summary:"
echo "       HEIMDALL_HOME:         $HEIMDALL_HOME"
echo "       HEIMDALL_CLI_PATH:     $HEIMDALL_CLI_PATH"
echo "       HEIMDALLD_PATH:        $HEIMDALLD_PATH"
echo "       BOR_PATH:              ${BOR_PATH:-<not used>}"
echo "       NETWORK:               $NETWORK"
echo "       NODETYPE:              $NODETYPE"
echo "       BACKUP_DIR:            $BACKUP_DIR"
echo "       MONIKER_NODE_NAME:     $MONIKER_NODE_NAME"
echo "       HEIMDALL_SERVICE_USER: $HEIMDALL_SERVICE_USER"
echo "       GENERATE_GENESIS:      $GENERATE_GENESIS"
echo ""


# Step 3: stop heimdall-v1. The apocalypse tag embeds the halt_height so heimdalld should be down already, running it for consistency/completeness
STEP=3
print_step $STEP "Stopping heimdall-v1"
ROLLBACK_ACTIONS["$STEP"]=":"
if systemctl list-units --type=service | grep -q heimdalld.service; then
    if systemctl is-active --quiet heimdalld; then
        systemctl stop heimdalld
    else
        echo "[INFO] heimdalld service is already stopped."
    fi
else
    if service heimdalld status &> /dev/null; then
        service heimdalld stop
    else
        echo "[INFO] heimdalld service is already stopped or not found."
    fi
fi

# Step 4: Ensure node has committed up to latest height
STEP=4
print_step $STEP "Checking that Heimdall v1 has committed the latest height"
ROLLBACK_ACTIONS["$STEP"]=":"

if [[ "$GENERATE_GENESIS" == "false" ]]; then
    echo "[INFO] Skipping committed height check since GENERATE_GENESIS=false was passed."
else
    # Get the last committed height from disk
    if ! COMMITTED_HEIGHT=$($HEIMDALL_CLI_PATH get-last-committed-height --home "$HEIMDALL_HOME" --quiet 2>/dev/null | tail -1); then
        handle_error $STEP "Unable to fetch committed height from disk with heimdallcli"
    fi

    if ! [[ "$COMMITTED_HEIGHT" =~ ^[0-9]+$ ]]; then
        handle_error $STEP "Invalid height value returned: $COMMITTED_HEIGHT"
    fi

    echo "[INFO] Latest committed height: $COMMITTED_HEIGHT"
    if [[ "$COMMITTED_HEIGHT" -lt "$APOCALYPSE_HEIGHT" ]]; then
        echo "[WARN] Node has not yet committed the apocalypse height."
        echo "       Expected: $APOCALYPSE_HEIGHT"
        echo "       Found:    $COMMITTED_HEIGHT"
        echo "       This node will NOT generate its own genesis file."
        GENERATE_GENESIS=false
    else
        echo "[INFO] Node has committed the apocalypse height."
        echo "       Expected: $APOCALYPSE_HEIGHT"
        echo "       Found:    $COMMITTED_HEIGHT"
        echo "       This node will generate its own genesis file."
        GENERATE_GENESIS=true
    fi
fi

# Step 5: Generate or download Heimdall v1 genesis JSON
STEP=5
print_step $STEP "Obtaining Heimdall v1 genesis JSON file"
GENESIS_FILE="$HEIMDALL_HOME/$DUMP_V1_GENESIS_FILE_NAME"
ROLLBACK_ACTIONS["$STEP"]="rm -f $GENESIS_FILE"
if $GENERATE_GENESIS; then
    echo "[INFO] Generating genesis file using heimdalld export..."
    if ! $HEIMDALL_CLI_PATH export-heimdall --home "$HEIMDALL_HOME" --chain-id "$CHAIN_ID"; then
        handle_error $STEP "Failed to generate Heimdall v1 genesis file $GENESIS_FILE"
    fi
    echo "[INFO] Genesis file generated to $GENESIS_FILE"
else
    echo "[INFO] Downloading genesis file from default source: $TRUSTED_GENESIS_URL"
    if ! curl -fsSL "$TRUSTED_GENESIS_URL" -o "$GENESIS_FILE"; then
        handle_error $STEP "Failed to download genesis file from $TRUSTED_GENESIS_URL"
    fi
    echo "[INFO] Genesis file downloaded to $GENESIS_FILE"
fi


# Step 6: Generate checksum of the genesis export
STEP=6
print_step $STEP "Generating checksum for Heimdall v1 genesis file, it will be saved in $HEIMDALL_HOME/$DUMP_V1_GENESIS_FILE_NAME.sha512"
CHECKSUM_FILE="$HEIMDALL_HOME/$DUMP_V1_GENESIS_FILE_NAME.sha512"
# Ensure the genesis file exists before computing checksum
if [[ ! -f "$GENESIS_FILE" ]]; then
    handle_error $STEP "Genesis file $GENESIS_FILE not found. Cannot generate checksum."
fi
ROLLBACK_ACTIONS["$STEP"]="rm -f $CHECKSUM_FILE"
# execute command
sha512sum "$GENESIS_FILE" | awk '{print $1}' > "$CHECKSUM_FILE"
# Verify checksum file exists and is not empty
if [[ ! -s "$CHECKSUM_FILE" ]]; then
    handle_error $STEP "Checksum file was not created or is empty."
fi
GENERATED_CHECKSUM=$(awk '{print $1}' "$CHECKSUM_FILE")
# Print checksum
echo "[INFO] Generated checksum: $GENERATED_CHECKSUM"


# Step 7: verify checksum
STEP=7
print_step $STEP "Verifying checksum"
ROLLBACK_ACTIONS["$STEP"]=":"
if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY-RUN] Skipping checksum verification"
else
    CHECKSUM_FILE="$HEIMDALL_HOME/$DUMP_V1_GENESIS_FILE_NAME.sha512"
    # Ensure checksum file exists before reading it
    if [[ ! -f "$CHECKSUM_FILE" ]]; then
        handle_error $STEP "Checksum file $CHECKSUM_FILE not found! Cannot verify checksum."
    fi
    # Read expected checksum from the file
    CHECKSUM=$(awk '{print $1}' "$CHECKSUM_FILE")
    # Verify checksum matches the generated one
    if [[ "$GENERATED_CHECKSUM" != "$CHECKSUM" ]]; then
        handle_error $STEP "Checksum mismatch! Expected: $CHECKSUM, Found: $GENERATED_CHECKSUM"
    fi
    echo "[INFO] Checksum verification passed."
fi

# Step 8: move heimdall-v1 to backup location
STEP=8
print_step $STEP "Moving $HEIMDALL_HOME to $BACKUP_DIR"
ROLLBACK_ACTIONS["$STEP"]="mv \"$BACKUP_DIR\" \"$HEIMDALL_HOME\" || (mkdir -p \"$HEIMDALL_HOME\" && cp -a \"$BACKUP_DIR/.\" \"$HEIMDALL_HOME\")"
# Create parent directory in case it doesn't exist
sudo mkdir -p "$(dirname "$BACKUP_DIR")" || handle_error $STEP "Failed to create parent directory for $BACKUP_DIR"
# Move Heimdall home to backup location
sudo mv "$HEIMDALL_HOME" "$BACKUP_DIR" || handle_error $STEP "Failed to move $HEIMDALL_HOME to $BACKUP_DIR"
echo "[INFO] Backup (move) completed successfully."

# Step 9 : select the proper heimdall-v2 binary package
STEP=9
print_step $STEP "Create temp directory for heimdall-v2 and target the right package based on current system"
tmpDir="/tmp/tmp-heimdall-v2"
ROLLBACK_ACTIONS["$STEP"]="rm -rf \"$tmpDir\""
sudo mkdir -p $tmpDir || handle_error $STEP "Cannot create $tmpDir directory for downloading files"
profileInfo=${NETWORK}-${NODETYPE}-config_v${HEIMDALL_V2_VERSION}
profileInforpm=${NETWORK}-${NODETYPE}-config-v${HEIMDALL_V2_VERSION}
baseUrl="https://github.com/0xPolygon/heimdall-v2/releases/download/v${HEIMDALL_V2_VERSION}"
case "$(uname -s).$(uname -m)" in
    Linux.x86_64)
        if command -v dpkg &> /dev/null; then
            type="deb"
            binary="heimdall-v${HEIMDALL_V2_VERSION}-amd64.deb"
            profile="heimdall-${profileInfo}-all.deb"
        elif command -v rpm &> /dev/null; then
            type="rpm"
            binary="heimdall-v${HEIMDALL_V2_VERSION}.x86_64.rpm"
            profile="heimdall-${profileInforpm}.noarch.rpm"
        elif command -v apk &> /dev/null; then
            handle_error $STEP "Sorry, there is no binary distribution for your platform"
        else
            handle_error $STEP "Sorry, there is no binary distribution for your platform"
        fi
        ;;
    Linux.aarch64)
        if command -v dpkg &> /dev/null; then
            type="deb"
            binary="heimdall-v${HEIMDALL_V2_VERSION}-arm64.deb"
            profile="heimdall-${profileInfo}-all.deb"
        elif command -v rpm &> /dev/null; then
            type="rpm"
            binary="heimdall-v${HEIMDALL_V2_VERSION}.aarch64.rpm"
            profile="heimdall-${profileInforpm}.noarch.rpm"
        elif command -v apk &> /dev/null; then
            handle_error $STEP "Sorry, there is no binary distribution for your platform"
        else
            handle_error $STEP "Sorry, there is no binary distribution for your platform"
        fi
        ;;
    Darwin.x86_64)
        handle_error $STEP "Sorry, there is no binary distribution for your platform"
        ;;
    Darwin.arm64|Darwin.aarch64)
        handle_error $STEP "Sorry, there is no binary distribution for your platform"
        ;;
    *) handle_error $STEP "Sorry, there is no binary distribution for your platform";;
esac
url="${baseUrl}/${binary}"
package="$tmpDir/$binary"


# Step 10: download heimdall-v2 binary package
STEP=10
print_step $STEP "Download heimdall-v2 binary package from $baseUrl to $tmpDir"
curl -L "$url" -o "$package" || handle_error $STEP "Failed to download binary from \"$url\""
ROLLBACK_ACTIONS["$STEP"]="rm -rf \"$tmpDir\""
if [ -n "$profile"  ]; then
    profileUrl="${baseUrl}/${profile}"
    profilePackage=$tmpDir/$profile
    curl -L "$profileUrl" -o "$profilePackage" || handle_error $STEP "Failed to download profile from \"$profileUrl\""
fi


# Step 11: unpack heimdall-v2 binary and install it
STEP=11
print_step $STEP "Unpack heimdall-v2 binary package and install it"
if [ "$type" = "tar.gz" ]; then
    unpack=$tmpDir/unpack
    ROLLBACK_ACTIONS["$STEP"]="rm -rf \"$unpack\""
    echo "[INFO] Creating unpack directory..."
    mkdir -p "$unpack" || handle_error $STEP "Failed to create unpack directory"
    echo "[INFO] Unpacking..."
    tar -xzf "$package" -C "$unpack" || handle_error $STEP "Failed to unpack '$package'"

    if [ -f "/usr/local/bin/heimdalld" ]; then
        echo "[INFO] Backing up existing heimdalld binary"
        sudo cp /usr/local/bin/heimdalld /usr/local/bin/heimdalld.bak || handle_error $STEP "Failed to backup existing heimdalld binary"
        ROLLBACK_ACTIONS["$STEP"]+=" && mv \"/usr/local/bin/heimdalld.bak\" \"/usr/local/bin/heimdalld\""
    fi
    echo "[INFO] Copying new binary from ${unpack}/heimdalld into /usr/local/bin/heimdalld"
    sudo cp "${unpack}/heimdalld" /usr/local/bin/heimdalld || handle_error $STEP "Failed to copy heimdalld binary to '/usr/local/bin/heimdalld'"
elif [ "$type" = "deb" ]; then
    echo "[INFO] Uninstalling any existing old binary"
    ROLLBACK_ACTIONS["$STEP"]=":"
    sudo dpkg -r heimdall heimdalld || handle_error $STEP "Failed to uninstall existing packages"
    sudo dpkg -i "$package" || handle_error $STEP "Failed to install $package"

    if [ -n "$profilePackage" ] && [ ! -d "$HEIMDALL_HOME/config" ]; then
        echo "[INFO] Installing v2 profile package"
        sudo dpkg -i "$profilePackage" || handle_error $STEP "Failed to install profile package"
    fi
elif [ "$type" = "rpm" ]; then
    ROLLBACK_ACTIONS["$STEP"]=":"
    sudo rpm -e heimdall || handle_error $STEP "Failed to uninstall old package"
    sudo rpm -i --force "$package" || handle_error $STEP "Failed to install $package"

    if [ -n "$profilePackage" ] && [ ! -d "$HEIMDALL_HOME/config" ]; then
        echo "[INFO] Installing v2 profile package"
        sudo rpm -i --force "$profilePackage" || handle_error $STEP "Failed to install profile package"
    fi
elif [ "$type" = "apk" ]; then
    echo "[INFO] Installing package"
    sudo apk add --allow-untrusted "$package" || handle_error $STEP "Failed to install $package"
    ROLLBACK_ACTIONS["$STEP"]=":"
fi
echo "[INFO] Heimdall-v2 installation completed."


# Step 12: move the heimdall-v2 binary to heimdall path
STEP=12
print_step $STEP "Moving the binary to $HEIMDALLD_PATH"
# Extract the directory path
dir_path=$(dirname "$HEIMDALLD_PATH")
# Ensure target directory exists
if [ ! -d "$dir_path" ]; then
    handle_error $STEP "Target directory $dir_path does not exist!"
fi
# Backup existing heimdalld binary if it exists
if [ -f "$HEIMDALLD_PATH" ]; then
    echo "[INFO] Backing up existing heimdalld binary..."
    sudo mv "$HEIMDALLD_PATH" "${HEIMDALLD_PATH}.bak" || handle_error $STEP "Failed to backup old heimdalld binary"
fi
ROLLBACK_ACTIONS["$STEP"]="if [ -f \"${HEIMDALLD_PATH}.bak\" ]; then mv \"${HEIMDALLD_PATH}.bak\" \"$HEIMDALLD_PATH\"; fi"
# Determine the actual location of the new binary
if [ -f "/usr/bin/heimdalld" ]; then
    NEW_BINARY="/usr/bin/heimdalld"
elif [ -f "$package" ]; then
    NEW_BINARY="$package"
else
    handle_error $STEP "Could not find the new heimdalld binary!"
fi
# Copy the new heimdalld binary
echo "[INFO] Resolved new binary at: $NEW_BINARY"
echo "[INFO] Copying new heimdalld binary from $NEW_BINARY to $HEIMDALLD_PATH..."
sudo cp "$NEW_BINARY" "$HEIMDALLD_PATH" || handle_error $STEP "Failed to copy new heimdalld binary"
# Ensure the new binary is executable
sudo chmod +x "$HEIMDALLD_PATH" || handle_error $STEP "Failed to set execution permissions on $HEIMDALLD_PATH"
echo "[INFO] heimdalld binary copied and set as executable successfully!"


# Step 13: verify heimdall-v2 version
STEP=13
print_step $STEP "Verifying Heimdall v2 version"
# Define rollback: Restore previous binary if something goes wrong
if [ -f "${HEIMDALLD_PATH}.bak" ]; then
    ROLLBACK_ACTIONS["$STEP"]="sudo mv \"${HEIMDALLD_PATH}.bak\" \"$HEIMDALLD_PATH\""
else
    ROLLBACK_ACTIONS["$STEP"]=":"
fi
# Check if heimdalld is installed and executable
if [[ ! -x "$HEIMDALLD_PATH" ]]; then
    handle_error $STEP "Heimdalld binary is missing or not executable: $HEIMDALLD_PATH"
fi
# Check heimdalld version
# Extract version from last non-empty line of heimdalld output
HEIMDALLD_V2_VERSION_RAW=$($HEIMDALLD_PATH version 2>/dev/null | awk 'NF' | tail -n 1)
if [[ -z "$HEIMDALLD_V2_VERSION_RAW" ]]; then
    handle_error $STEP "Failed to retrieve Heimdall v2 version. Installation may have failed."
fi
# Normalize actual and expected versions
NORMALIZED_HEIMDALLD_V2_VERSION=$(normalize_version "$HEIMDALLD_V2_VERSION_RAW")
NORMALIZED_EXPECTED_HEIMDALL_V2_VERSION=$(normalize_version "$HEIMDALL_V2_VERSION")
if [[ "$NORMALIZED_HEIMDALLD_V2_VERSION" != "$NORMALIZED_EXPECTED_HEIMDALL_V2_VERSION" ]]; then
    handle_error $STEP "Heimdall v2 version mismatch! Expected: $HEIMDALL_V2_VERSION, Found: $HEIMDALLD_V2_VERSION_RAW"
fi
# Ensure HEIMDALL_HOME exists
if [[ ! -d "$HEIMDALL_HOME" ]]; then
    handle_error $STEP "HEIMDALL_HOME does not exist after installation."
fi
echo "[INFO] heimdall-v2 is using the correct version $HEIMDALL_V2_VERSION"


# Step 14: migrate genesis file
STEP=14
print_step $STEP "Migrating Heimdall v1 genesis file $GENESIS_FILE to v2 format, the result file will be saved in $BACKUP_DIR/migrated_$DUMP_V1_GENESIS_FILE_NAME"
# Define the target output file
MIGRATED_GENESIS_FILE="$BACKUP_DIR/migrated_$DUMP_V1_GENESIS_FILE_NAME"
# Ensure the v1 genesis file exists before proceeding
if [[ ! -f "$BACKUP_DIR/$DUMP_V1_GENESIS_FILE_NAME" ]]; then
    handle_error $STEP "Genesis file $BACKUP_DIR/$DUMP_V1_GENESIS_FILE_NAME not found! Cannot proceed with migration."
fi
# Sanity check: warn if GENESIS_TIME is in the future
GENESIS_TIMESTAMP=$(date -d "$GENESIS_TIME" +%s)
NOW_TIMESTAMP=$(date +%s)
if (( GENESIS_TIMESTAMP > NOW_TIMESTAMP )); then
    echo "[WARNING] GENESIS_TIME is in the future: $GENESIS_TIME"
    echo "          This may cause Heimdall to sleep until that time on startup."
fi
# Run the migration command
if ! heimdalld migrate "$BACKUP_DIR/$DUMP_V1_GENESIS_FILE_NAME" --chain-id="$CHAIN_ID" --genesis-time="$GENESIS_TIME" --initial-height="$INITIAL_HEIGHT" --verify-data="$VERIFY_DATA"; then
    handle_error $STEP "Migration command failed."
fi
# Define rollback action only if the file was created successfully
if [ -f "$MIGRATED_GENESIS_FILE" ]; then
    ROLLBACK_ACTIONS["$STEP"]="rm -f \"$MIGRATED_GENESIS_FILE\""
fi
echo "[INFO] Genesis file migrated successfully from v1 to v2"
# ensure migrated genesis file exists
if [[ ! -f "$MIGRATED_GENESIS_FILE" ]]; then
    handle_error $STEP "Expected migrated genesis file not found at $MIGRATED_GENESIS_FILE"
fi
# Confirm initial_height in migrated genesis matches configured INITIAL_HEIGHT
echo "[INFO] Verifying initial_height in migrated genesis file..."
ACTUAL_INITIAL_HEIGHT=$(jq -r '.initial_height' "$MIGRATED_GENESIS_FILE")
[[ "$ACTUAL_INITIAL_HEIGHT" =~ ^[0-9]+$ ]] || handle_error $STEP "Failed to parse initial_height from migrated genesis."
if [[ "$ACTUAL_INITIAL_HEIGHT" != "$INITIAL_HEIGHT" ]]; then
    echo "[WARNING] Mismatch detected!"
    echo "          Configured INITIAL_HEIGHT: $INITIAL_HEIGHT"
    echo "          Genesis file contains:     $ACTUAL_INITIAL_HEIGHT"
    handle_error $STEP "INITIAL_HEIGHT mismatch detected"
else
    echo "[INFO] initial_height in genesis matches expected value: $INITIAL_HEIGHT"
fi


# Step 15: Generate checksum of the migrated genesis
STEP=15
print_step $STEP "Generating checksum for Heimdall v2 genesis file, it will be saved in $MIGRATED_GENESIS_FILE.sha512"
MIGRATED_CHECKSUM_FILE="$MIGRATED_GENESIS_FILE.sha512"
# Ensure the genesis file exists before computing checksum
if [[ ! -f "$MIGRATED_GENESIS_FILE" ]]; then
    handle_error $STEP "Migrated genesis file $MIGRATED_GENESIS_FILE not found. Cannot generate checksum."
fi
ROLLBACK_ACTIONS["$STEP"]="rm -f $MIGRATED_CHECKSUM_FILE"
# execute command
sha512sum "$MIGRATED_GENESIS_FILE" | awk '{print $1}' > "$MIGRATED_CHECKSUM_FILE"
# Verify checksum file exists and is not empty
if [[ ! -s "$MIGRATED_CHECKSUM_FILE" ]]; then
    handle_error $STEP "Checksum file was not created or is empty."
fi
GENERATED_MIGRATED_CHECKSUM=$(awk '{print $1}' "$MIGRATED_CHECKSUM_FILE")
# Print checksum
echo "[INFO] Generated checksum: $GENERATED_MIGRATED_CHECKSUM"


# Step 16: verify checksum of the migrated genesis
STEP=16
print_step $STEP "Verifying checksum of the migrated genesis file"
ROLLBACK_ACTIONS["$STEP"]=":"
if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY-RUN] Skipping checksum verification"
else
    MIGRATED_CHECKSUM_FILE="$MIGRATED_GENESIS_FILE.sha512"
    # Ensure checksum file exists before reading it
    if [[ ! -f "$MIGRATED_CHECKSUM_FILE" ]]; then
        handle_error $STEP "Checksum file $MIGRATED_CHECKSUM_FILE not found! Cannot verify checksum."
    fi
    # Read expected checksum from the file
    MIGRATED_CHECKSUM=$(awk '{print $1}' "$MIGRATED_CHECKSUM_FILE")
    # Verify checksum matches the generated one
    if [[ "$GENERATED_MIGRATED_CHECKSUM" != "$MIGRATED_CHECKSUM" ]]; then
        handle_error $STEP "Checksum mismatch! Expected: $MIGRATED_CHECKSUM, Found: $GENERATED_MIGRATED_CHECKSUM"
    fi
    echo "[INFO] Checksum verification passed."
fi


# Step 17: create temp heimdall-v2 home dir
STEP=17
print_step $STEP "Creating temp directory for heimdall-v2 in and applying proper permissions"
tmpDirV2Home="/tmp/tmp-heimdall-v2-home"
ROLLBACK_ACTIONS["$STEP"]="rm -rf \"$tmpDirV2Home\""
sudo mkdir -p "$tmpDirV2Home" || handle_error $STEP "Failed to create temporary directory"
# apply proper permissions for the current user
sudo chmod -R 755 "$tmpDirV2Home" || handle_error $STEP "Failed to set permissions"
sudo chown -R "$HEIMDALL_SERVICE_USER" "$tmpDirV2Home" || handle_error $STEP "Failed to change ownership"
echo "[INFO] $tmpDirV2Home created successfully"


# Step 18: init heimdall-v2
STEP=18
print_step $STEP "Initializing heimdalld for version v2 with moniker $MONIKER_NODE_NAME"
# Ensure Heimdall home exists before proceeding
if [[ ! -d "$HEIMDALL_HOME" ]]; then
    handle_error $STEP "HEIMDALL_HOME does not exist. Cannot proceed with initialization."
fi
# Backup old Heimdall home before clearing it
if [[ -d "$HEIMDALL_HOME" ]]; then
    echo "[INFO] Creating backup of existing Heimdall home..."
    sudo mv "$HEIMDALL_HOME" "$HEIMDALL_HOME.bak" || handle_error $STEP "Failed to backup old Heimdall home."
fi
ROLLBACK_ACTIONS["$STEP"]="if [ -d \"$HEIMDALL_HOME.bak\" ]; then mv \"$HEIMDALL_HOME.bak\" \"$HEIMDALL_HOME\"; fi"
# Init Heimdall v2
if ! heimdalld init "$MONIKER_NODE_NAME" --chain-id "$CHAIN_ID" --home="$tmpDirV2Home" &> /dev/null; then
    handle_error $STEP "Failed to initialize heimdalld."
fi
# Ensure Heimdall home directory exists before clearing it
if [[ ! -d "$HEIMDALL_HOME" ]]; then
    mkdir -p "$HEIMDALL_HOME" || handle_error $STEP "Failed to create Heimdall home directory."
fi
# Remove old Heimdall home content safely
find "$HEIMDALL_HOME" -mindepth 1 -delete || handle_error $STEP "Failed to clear old Heimdall home directory"
# Move the new Heimdall home
echo "[INFO] Moving new Heimdall home from temp location..."
if ! sudo cp -a "$tmpDirV2Home/." "$HEIMDALL_HOME"; then
    handle_error $STEP "Failed to move new Heimdall home."
fi
sudo rm -rf "$tmpDirV2Home"
echo "[INFO] heimdalld initialized successfully."


# Step 19: verify required directories exist
STEP=19
print_step $STEP "Verifying required directories and configuration files in $HEIMDALL_HOME"
# Define rollback: Restore old Heimdall home if needed
if [ -d "$HEIMDALL_HOME.bak" ]; then
    ROLLBACK_ACTIONS["$STEP"]="mv \"$HEIMDALL_HOME.bak\" \"$HEIMDALL_HOME\""
else
    ROLLBACK_ACTIONS["$STEP"]=":"
fi
# Check if required directories exist
REQUIRED_DIRS=("data" "config")
for dir in "${REQUIRED_DIRS[@]}"; do
    if [[ ! -d "$HEIMDALL_HOME/$dir" ]]; then
        handle_error $STEP "Required directory is missing: $HEIMDALL_HOME/$dir"
    fi
done
# Ensure config directory contains the necessary files
REQUIRED_CONFIG_FILES=("app.toml" "client.toml" "config.toml" "genesis.json" "node_key.json" "priv_validator_key.json")
for file in "${REQUIRED_CONFIG_FILES[@]}"; do
    if [[ ! -f "$HEIMDALL_HOME/config/$file" ]]; then
        handle_error $STEP "Missing required configuration file: $file"
    fi
done
# Ensure data directory contains the necessary files
REQUIRED_DATA_FILES=("priv_validator_state.json")
for file in "${REQUIRED_DATA_FILES[@]}"; do
    if [[ ! -f "$HEIMDALL_HOME/data/$file" ]]; then
        handle_error $STEP "Missing required data file: $file"
    fi
done
echo "[INFO] All required directories are present in $HEIMDALL_HOME"


# Step 20: Restore bridge directory from backup
STEP=20
print_step $STEP "Restoring bridge directory from backup if present"
ROLLBACK_ACTIONS["$STEP"]=":"  # No rollback needed for restore
BRIDGE_SRC="$BACKUP_DIR/bridge"
BRIDGE_DEST="$HEIMDALL_HOME/bridge"

if [[ -d "$BRIDGE_SRC" ]]; then
    echo "[INFO] Detected bridge directory in backup: $BRIDGE_SRC"
    echo "[INFO] Restoring it to: $BRIDGE_DEST"
    cp -a "$BRIDGE_SRC" "$BRIDGE_DEST" || handle_error $STEP "Failed to restore bridge directory"
    echo "[INFO] Bridge directory restored successfully."
else
    echo "[INFO] No bridge directory found in backup. Skipping restore."
fi


# Step 21: move genesis file to new heimdall home
STEP=21
print_step $STEP "Moving genesis file to the new $HEIMDALL_HOME"
TARGET_GENESIS_FILE="$HEIMDALL_HOME/config/genesis.json"
# Backup existing genesis file before replacing it
if [ -f "$TARGET_GENESIS_FILE" ]; then
    echo "[INFO] Backing up existing genesis file..."
    mv "$TARGET_GENESIS_FILE" "${TARGET_GENESIS_FILE}.bak"
    echo "[INFO] Backup saved at: $TARGET_GENESIS_FILE.bak"
    ROLLBACK_ACTIONS["$STEP"]="mv \"${TARGET_GENESIS_FILE}.bak\" \"$TARGET_GENESIS_FILE\""
else
    ROLLBACK_ACTIONS["$STEP"]="rm -f \"$TARGET_GENESIS_FILE\""
fi
# Replace with the migrated genesis
cp -p "$MIGRATED_GENESIS_FILE" "$TARGET_GENESIS_FILE" || handle_error $STEP "Failed to replace genesis file with migrated version."


# Step 22: edit priv_validator_key.json file according to v2 setup
STEP=22
print_step $STEP "Updating priv_validator_key.json file"
PRIV_VALIDATOR_FILE="$HEIMDALL_HOME/config/priv_validator_key.json"
TEMP_PRIV_FILE="temp_priv_validator_key.json"
TEMP_FILES+=("$TEMP_PRIV_FILE")

if [ -f "$PRIV_VALIDATOR_FILE" ]; then
    echo "[INFO] Creating backup of priv_validator_key.json..."
    sudo cp "$PRIV_VALIDATOR_FILE" "$PRIV_VALIDATOR_FILE.bak" || handle_error $STEP "Failed to backup priv_validator_key.json"
    echo "[INFO] Backup saved at: $PRIV_VALIDATOR_FILE.bak"
    ROLLBACK_ACTIONS["$STEP"]="if [ -f \"$PRIV_VALIDATOR_FILE.bak\" ]; then mv \"$PRIV_VALIDATOR_FILE.bak\" \"$PRIV_VALIDATOR_FILE\"; fi"
else
    handle_error $STEP "priv_validator_key.json not found in Heimdall config directory!"
fi
ADDRESS=$(jq -r '.address' "$BACKUP_DIR/config/priv_validator_key.json")
PUB_KEY_VALUE=$(jq -r '.pub_key.value' "$BACKUP_DIR/config/priv_validator_key.json")
PRIV_KEY_VALUE=$(jq -r '.priv_key.value' "$BACKUP_DIR/config/priv_validator_key.json")
if jq --arg addr "$ADDRESS" \
      --arg pub "$PUB_KEY_VALUE" \
      --arg priv "$PRIV_KEY_VALUE" \
      '.address = $addr | .pub_key.value = $pub | .priv_key.value = $priv' \
      "$PRIV_VALIDATOR_FILE" > "$TEMP_PRIV_FILE"; then
    if [[ ! -s "$TEMP_PRIV_FILE" ]]; then
        handle_error $STEP "Updated priv_validator_key.json is empty or invalid!"
    fi
    mv "$TEMP_PRIV_FILE" "$PRIV_VALIDATOR_FILE" || handle_error $STEP "Failed to move updated priv_validator_key.json into place"
else
    handle_error $STEP "Failed to update priv_validator_key.json"
fi
echo "[INFO] Updated priv_validator_key.json file saved as $PRIV_VALIDATOR_FILE"


# Step 23: edit node_key.json file according to v2 setup
STEP=23
print_step $STEP "Updating node_key.json file"
NODE_KEY_FILE="$HEIMDALL_HOME/config/node_key.json"
TEMP_NODE_KEY_FILE="temp_node_key.json"
TEMP_FILES+=("$TEMP_NODE_KEY_FILE")
if [ -f "$NODE_KEY_FILE" ]; then
    echo "[INFO] Creating backup of node_key.json..."
    cp "$NODE_KEY_FILE" "$NODE_KEY_FILE.bak" || handle_error $STEP "Failed to backup node_key.json"
    echo "[INFO] Backup saved at: $NODE_KEY_FILE.bak"
    ROLLBACK_ACTIONS["$STEP"]="if [ -f \"$NODE_KEY_FILE.bak\" ]; then mv \"$NODE_KEY_FILE.bak\" \"$NODE_KEY_FILE\"; fi"
else
    handle_error $STEP "node_key.json not found in Heimdall config directory!"
fi
NODE_KEY=$(jq -r '.priv_key.value' "$BACKUP_DIR/config/node_key.json") || handle_error $STEP "Failed to extract priv_key.value from backup node_key.json"
if jq --arg nodekey "$NODE_KEY" \
      '.priv_key.value = $nodekey' \
      "$NODE_KEY_FILE" > "$TEMP_NODE_KEY_FILE"; then
    if [[ ! -s "$TEMP_NODE_KEY_FILE" ]]; then
        handle_error $STEP "Updated node_key.json is empty or invalid!"
    fi
    mv "$TEMP_NODE_KEY_FILE" "$NODE_KEY_FILE" || handle_error $STEP "Failed to move updated node_key.json into place"
else
    handle_error $STEP "Failed to update node_key.json"
fi
echo "[INFO] Updated node_key.json file saved as $NODE_KEY_FILE"


# Step 24: Fix JSON formatting in priv_validator_state.json and set initial height
STEP=24
print_step $STEP "Fixing formatting in priv_validator_state.json and set initial height"
PRIV_VALIDATOR_STATE="$HEIMDALL_HOME/data/priv_validator_state.json"
TEMP_STATE_FILE="temp_priv_validator_state.json"
TEMP_FILES+=("$TEMP_STATE_FILE")
if [ ! -f "$PRIV_VALIDATOR_STATE" ]; then
    handle_error $STEP "priv_validator_state.json not found in $HEIMDALL_HOME/data/"
fi
echo "[INFO] Creating backup of priv_validator_state.json..."
cp "$PRIV_VALIDATOR_STATE" "$PRIV_VALIDATOR_STATE.bak" || handle_error $STEP "Failed to backup priv_validator_state.json"
echo "[INFO] Backup saved at: $PRIV_VALIDATOR_STATE.bak"
ROLLBACK_ACTIONS["$STEP"]="if [ -f \"$PRIV_VALIDATOR_STATE.bak\" ]; then mv \"$PRIV_VALIDATOR_STATE.bak\" \"$PRIV_VALIDATOR_STATE\"; fi"
# Validate the file has proper JSON
jq empty "$PRIV_VALIDATOR_STATE" || handle_error $STEP "Invalid JSON detected in priv_validator_state.json"
# Apply transformations:
#   1. Convert "round" from string to int
#   2. Set "height" to string value of $INITIAL_HEIGHT
if jq --arg height "$INITIAL_HEIGHT" '.round |= tonumber | .height = $height' "$PRIV_VALIDATOR_STATE" > "$TEMP_STATE_FILE"; then
    if [[ ! -s "$TEMP_STATE_FILE" ]]; then
        handle_error $STEP "Updated priv_validator_state.json is empty or invalid!"
    fi
    mv "$TEMP_STATE_FILE" "$PRIV_VALIDATOR_STATE" || handle_error $STEP "Failed to move updated priv_validator_state.json into place"
else
    handle_error $STEP "Failed to update priv_validator_state.json"
fi
echo "[INFO] Successfully updated priv_validator_state.json"


# Step 25: Restore addrbook.json from backup if it exists
STEP=25
print_step $STEP "Restoring addrbook.json from backup (if present)"
ADDRBOOK_FILE="$BACKUP_DIR/config/addrbook.json"
TARGET_ADDRBOOK_FILE="$HEIMDALL_HOME/config/addrbook.json"
if [ -f "$ADDRBOOK_FILE" ]; then
    # Backup current one (if any)
    if [ -f "$TARGET_ADDRBOOK_FILE" ]; then
        cp "$TARGET_ADDRBOOK_FILE" "$TARGET_ADDRBOOK_FILE.bak" || handle_error $STEP "Failed to backup existing addrbook.json"
        echo "[INFO] Backup saved at: $TARGET_ADDRBOOK_FILE.bak"
    fi
    cp "$ADDRBOOK_FILE" "$TARGET_ADDRBOOK_FILE" || handle_error $STEP "Failed to restore addrbook.json from backup"
    echo "[INFO] addrbook.json restored successfully."
    ROLLBACK_ACTIONS["$STEP"]="if [ -f \"$TARGET_ADDRBOOK_FILE.bak\" ]; then mv \"$TARGET_ADDRBOOK_FILE.bak\" \"$TARGET_ADDRBOOK_FILE\"; else rm -f \"$TARGET_ADDRBOOK_FILE\"; fi"
else
    echo "[INFO] No addrbook.json found in backup. Skipping restore."
    ROLLBACK_ACTIONS["$STEP"]=":"  # No-op rollback if nothing changed
fi


# Step 26: Configuration changes
STEP=26
print_step $STEP "Applying minimal v1 → v2 configuration migration"
echo -e "\n⚠️  [INFO] This step will automatically migrate a minimal and safe subset of configuration values from Heimdall v1 to Heimdall v2."
echo -e "   Only the following keys will be carried over:\n"
echo -e "📁 From v1 \033[1mconfig.toml\033[0m → v2 config.toml:"
echo -e "   - moniker"
echo -e "   - external_address"
echo -e "   - seeds"
echo -e "   - persistent_peers"
echo -e "   - max_num_inbound_peers"
echo -e "   - max_num_outbound_peers"
echo -e "   - proxy_app"
echo -e "   - addr_book_strict\n"
echo -e "📁 From v1 \033[1mheimdall-config.toml\033[0m → v2 app.toml:"
echo -e "   - eth_rpc_url"
echo -e "   - bor_rpc_url"
echo -e "   - bor_grpc_flag"
echo -e "   - bor_grpc_url"
echo -e "   - amqp_url\n"
echo -e "📁 Into \033[1mclient.toml\033[0m:"
echo -e "   - chain-id = \"$CHAIN_ID\"\n"
echo -e "💡 You may manually edit other parameters (e.g. ports, metrics, logging) after migration."
echo -e "\n📁 \033[1mBor Configuration Notice\033[0m:"
echo -e "   ⚠️  Please update your Bor's \033[1mbor/config.toml\033[0m manually to reflect v2-compatible settings."
echo -e "   💡 You can optionally add the following entry under the \033[1m[heimdall]\033[0m section to enable WebSocket support:"
echo -e "\n     [heimdall]"
echo -e "     ws-address = \"ws://localhost:26657/websocket\"\n"
echo -e "   ✅ This setting is recommended, as it improves performance by reducing the number of HTTP polling requests from Heimdall to Bor."
echo -e "   🔄 After updating the config, make sure to restart your Bor node for changes to take effect.\n"
# 1. Set chain-id in client.toml
CLIENT_TOML="$HEIMDALL_HOME/config/client.toml"
echo "[INFO] Setting chain-id in client.toml..."
set_toml_key "$CLIENT_TOML" "chain-id" "$CHAIN_ID"
actual_chain_id=$(grep -E '^chain-id\s*=' "$CLIENT_TOML" | cut -d'=' -f2 | tr -d ' "')
if [[ "$actual_chain_id" != "$CHAIN_ID" ]]; then
    handle_error $STEP "Validation failed: expected chain-id = $CHAIN_ID, found $actual_chain_id"
fi
echo "[OK]   client.toml: chain-id = $CHAIN_ID"
# 2. Migrate config.toml keys
OLD_CONFIG_TOML="$BACKUP_DIR/config/config.toml"
NEW_CONFIG_TOML="$HEIMDALL_HOME/config/config.toml"
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
# Validate
for key in "${CONFIG_KEYS[@]}"; do
    expected=$(grep -E "^$key\s*=" "$OLD_CONFIG_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    actual=$(grep -E "^$key\s*=" "$NEW_CONFIG_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    if [[ "$expected" != "$actual" ]]; then
        handle_error $STEP "Validation failed for '$key' in config.toml: expected '$expected', got '$actual'"
    fi
done
echo "[INFO] config.toml values migrated successfully."
# 3. Migrate heimdall-config.toml → app.toml
OLD_HEIMDALL_CONFIG_TOML="$BACKUP_DIR/config/heimdall-config.toml"
NEW_APP_TOML="$HEIMDALL_HOME/config/app.toml"
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
# Validate
for key in "${APP_KEYS[@]}"; do
    expected=$(grep -E "^$key\s*=" "$OLD_HEIMDALL_CONFIG_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    actual=$(grep -E "^$key\s*=" "$NEW_APP_TOML" | cut -d'=' -f2- | tr -d ' "' || true)
    if [[ "$expected" != "$actual" ]]; then
        handle_error $STEP "Validation failed for '$key' in app.toml: expected '$expected', got '$actual'"
    fi
done
echo "[INFO] app.toml values migrated successfully."


# Step 27: Assign correct ownership to Heimdall directories
STEP=27
print_step $STEP "Assigning correct ownership and permissions under $HEIMDALL_HOME as user: $HEIMDALL_SERVICE_USER"
ROLLBACK_ACTIONS["$STEP"]=":"  # No rollback needed
# Sanity check: avoid chowning critical paths
CRITICAL_PATHS=("/" "/usr" "/usr/bin" "/bin" "/lib" "/lib64" "/etc" "/boot")
for path in "${CRITICAL_PATHS[@]}"; do
    if [[ "$HEIMDALL_HOME" == "$path" ]]; then
        handle_error $STEP "Refusing to chown critical system path: $path"
    fi
done
echo "[INFO] Recursively setting ownership of all contents in $HEIMDALL_HOME to $HEIMDALL_SERVICE_USER"
sudo chown -R "$HEIMDALL_SERVICE_USER":"$HEIMDALL_SERVICE_USER" "$HEIMDALL_HOME" || handle_error $STEP "Failed to chown $HEIMDALL_HOME"
# Set 600 permissions for all files
echo "[INFO] Setting 600 permissions for all files under $HEIMDALL_HOME"
find "$HEIMDALL_HOME" -type f ! -name '.*' -exec chmod 600 {} \; || handle_error $STEP "Failed to chmod files"
# Set 700 permissions for all directories
echo "[INFO] Setting 700 permissions for all directories under $HEIMDALL_HOME"
find "$HEIMDALL_HOME" -type d ! -name '.*' -exec chmod 700 {} \; || handle_error $STEP "Failed to chmod directories"
echo "[INFO] Ownership and permissions successfully enforced under $HEIMDALL_HOME"

# Step 28: Automatically update the systemd unit file to set the correct user
STEP=28
print_step $STEP "Patching systemd service file to enforce user: $HEIMDALL_SERVICE_USER"
SERVICE_FILE=$(systemctl status heimdalld | grep 'Loaded:' | awk '{print $3}' | tr -d '();')
if [[ -z "$SERVICE_FILE" || ! -f "$SERVICE_FILE" ]]; then
    echo "[WARNING] Could not detect systemd unit file for heimdalld. Please update it manually to set the correct 'User=' value."
    handle_error $STEP "system unit not detected"
else
    echo "[INFO] Detected service file: $SERVICE_FILE"
    BACKUP_SERVICE_FILE="${SERVICE_FILE}.bak"
    echo "[INFO] Creating backup at: $BACKUP_SERVICE_FILE"
    sudo cp "$SERVICE_FILE" "$BACKUP_SERVICE_FILE" || handle_error $STEP "Failed to backup service file"

    echo "[INFO] Updating User= in [Service] block only if present"
    sudo sed -i "/^\[Service\]/,/^\[/{s/^\(\s*User=\).*/\1$HEIMDALL_SERVICE_USER/}" "$SERVICE_FILE"

    echo "[INFO] Reloading systemd daemon"
    sudo systemctl daemon-reload
    echo "[INFO] Systemd unit patched."
fi


# Step 29: Clean up .bak files in HEIMDALL_HOME
STEP=29
print_step $STEP "Cleaning up .bak files in parent directory of $HEIMDALL_HOME"
ROLLBACK_ACTIONS["$STEP"]=":"  # No rollback needed for cleanup
# Determine the parent directory of HEIMDALL_HOME
HEIMDALL_PARENT_DIR=$(dirname "$HEIMDALL_HOME")
# Find and delete all .bak files within the parent directory
BAK_FILES=$(find "$HEIMDALL_PARENT_DIR" -type f -name "*.bak")
if [[ -n "$BAK_FILES" ]]; then
    echo "[INFO] Removing the following backup files:"
    echo "$BAK_FILES"
    find "$HEIMDALL_PARENT_DIR" -type f -name "*.bak" -exec rm -f {} \;
    echo "[INFO] Cleanup complete."
else
    echo "[INFO] No .bak files found in $HEIMDALL_PARENT_DIR"
fi

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

echo -e "\n⚠️  \033[1mManual Verification Required\033[0m:"
echo -e "   Please review the updated configuration files under:"
echo -e "     \033[1m$HEIMDALL_HOME/config/\033[0m"
echo -e "   and ensure that they match your expected custom values from:"
echo -e "     \033[1m$BACKUP_DIR/config/\033[0m"
echo -e "   Especially if you had non-standard settings (e.g., ports, metrics, logging, pruning)."
echo -e "   The migration only carried over a minimal and safe subset of parameters:\n"
echo -e "📁 \033[1mconfig.toml\033[0m:"
echo -e "   - moniker"
echo -e "   - external_address"
echo -e "   - seeds"
echo -e "   - persistent_peers"
echo -e "   - max_num_inbound_peers"
echo -e "   - max_num_outbound_peers"
echo -e "   - proxy_app"
echo -e "   - addr_book_strict\n"
echo -e "📁 \033[1mapp.toml\033[0m:"
echo -e "   - eth_rpc_url"
echo -e "   - bor_rpc_url"
echo -e "   - bor_grpc_flag"
echo -e "   - bor_grpc_url"
echo -e "   - amqp_url\n"
echo -e "📁 \033[1mclient.toml\033[0m:"
echo -e "   - chain-id = \"$CHAIN_ID\"\n"

echo -e "\n✅ [SUCCESS] Heimdall v2 migration completed successfully! ✅"
echo -e "🕓 Migration completed in ${MINUTES}m ${SECONDS}s."
echo -e "When notified to start heimdall-v2, please run: "
echo -e "sudo systemctl daemon-reload && sudo systemctl start heimdalld"
echo -e "if you are running telemetry, also restart that service with: "
echo -e "sudo systemctl restart telemetry"
echo -e "Then - once heimdall is running, to verify everything is correct - check the logs using:"
echo -e "📌 journalctl -fu heimdalld"

# Don't remove next line!
# End of script
