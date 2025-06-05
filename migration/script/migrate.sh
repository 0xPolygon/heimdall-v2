#!/bin/bash

umask 0022

# -------------------- Env variables, to be adjusted before rolling out --------------------
V1_VERSION="1.2.3-35-gab6cbfb0"
V1_GENESIS_CHECKSUM="25ff6918888b080d6ed4f87320338bd9cfd102b5dd476998e44674bd43a66c8d2d42bc4aa3d370f963cfbc9641e904d79256eb51c145f0daac7cfaf817b66c87"
V2_GENESIS_CHECKSUM="1e3e64360efe2282c065d3b2e8aa7574568bec0ee139561ad3d968939168de22a166b3a6f1a87ab0afe6bec3716ddf71b26217bc17815336ef1e97390396def2"
V2_VERSION="0.1.27"
V1_CHAIN_ID="devnet"
V2_CHAIN_ID="devnet"
V2_GENESIS_TIME="2025-06-05T13:30:00Z"
V1_HALT_HEIGHT=22238836
V2_BRANCH_NAME="migration-mumbai"

# -------------------- const env variables --------------------
V2_INITIAL_HEIGHT=$(( V1_HALT_HEIGHT + 1 ))
VERIFY_EXPORTED_DATA=true
DUMP_V1_GENESIS_FILE_NAME="dump-genesis.json"
DRY_RUN=false
# TODO GitHub has a limit on file size for raw content, so we need to compress the file and adapt the STEP=5 of the script to handle it, or use a different source (not GitHub)
TRUSTED_GENESIS_URL="https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/${V2_BRANCH_NAME}/migration/networks/${V1_CHAIN_ID}/dump-genesis.json"

# -------------------- Script start --------------------
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
NETWORK=""
NODETYPE=""
HEIMDALL_SERVICE_USER=""
GENERATE_GENESIS=""

show_help() {
  echo "Usage: sudo bash migrate.sh \\
            --heimdall-v1-home=<PATH_TO_HEIMDALL_HOME> \\
            --heimdallcli-path=<PATH_TO_HEIMDALLCLI_BINARY> \\
            --heimdalld-path=<PATH_TO_HEIMDALLD_BINARY> \\
            --network=mainnet|amoy \\
            --node-type=sentry|validator \\
            --service-user=<HEIMDALL_SERVICE_USER> \\
            --generate-genesis=true|false"
  echo "Required arguments:"
  echo "  --heimdall-v1-home=PATH          Absolute path to Heimdall home directory (must contain 'config' and 'data')"
  echo "  --heimdallcli-path=PATH               Path to the heimdallcli binary (must be >= v1.0.10)"
  echo "  --heimdalld-path=PATH                 Path to the heimdalld binary (must be apocalypse tag: 1.2.0-41-*)"
  echo "  --network=mainnet|amoy       Network this node is part of (use 'mainnet' or 'amoy')"
  echo "  --node-type=sentry|validator  Whether this node is a sentry or validator"
  echo "  --service-user=USER          System user that runs the Heimdall service"
  echo "                                (typically 'heimdall'; check systemd with 'sudo systemctl status heimdalld')"
  echo "  --generate-genesis=true|false Whether to export genesis from heimdalld (recommended: true)"
  echo "Optional arguments:"
  echo "Example:"
  echo "  sudo bash migrate.sh \\
    --heimdall-v1-home=/var/lib/heimdall \\
    --heimdallcli-path=/usr/bin/heimdallcli \\
    --heimdalld-path=/usr/bin/heimdalld \\
    --network=mainnet \\
    --node-type=validator \\
    --service-user=heimdall \\
    --generate-genesis=true"
  exit 0
}

# Parse args
for arg in "$@"; do
  case $arg in
    --heimdall-v1-home=*) HEIMDALL_HOME="${arg#*=}" ;;
    --heimdallcli-path=*) HEIMDALL_CLI_PATH="${arg#*=}" ;;
    --heimdalld-path=*) HEIMDALLD_PATH="${arg#*=}" ;;
    --network=*) NETWORK="${arg#*=}" ;;
    --node-type=*) NODETYPE="${arg#*=}" ;;
    --service-user=*) HEIMDALL_SERVICE_USER="${arg#*=}" ;;
    --generate-genesis=*) GENERATE_GENESIS="${arg#*=}" ;;
    --help|-h) show_help ;;
    *) echo "[ERROR] Unknown argument: $arg"; exit 1 ;;
  esac
  shift || true
  done

# Check required
missing_args=()
[[ -z "$HEIMDALL_HOME" ]] && missing_args+=("--heimdall-v1-home")
[[ -z "$HEIMDALL_CLI_PATH" ]] && missing_args+=("--heimdallcli-path")
[[ -z "$HEIMDALLD_PATH" ]] && missing_args+=("--heimdalld-path")
[[ -z "$NETWORK" ]] && missing_args+=("--network")
[[ -z "$NODETYPE" ]] && missing_args+=("--node-type")
[[ -z "$HEIMDALL_SERVICE_USER" ]] && missing_args+=("--service-user")
[[ -z "$GENERATE_GENESIS" ]] && missing_args+=("--generate-genesis")
if (( ${#missing_args[@]} > 0 )); then
  echo "[ERROR] Missing required arguments: ${missing_args[*]}"
  show_help
fi

# Track temp files to clean up on exit
TEMP_FILES=()

# Function to print step information
print_step() {
    echo ""
    local step_number=$1
    local message=$2
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "\n[$timestamp] [STEP $step_number] $message"
}

# Function to handle errors
handle_error() {
    local step_number=$1
    local message=$2
    echo -e "\n[ERROR] Step $step_number failed: $message"
    exit 1
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
# Define base and new dependencies
DEPENDENCIES=("curl" "tar" "jq" "sha512sum" "file" "awk" "sed" "systemctl" "grep" "id")
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
  NORMALIZED_EXPECTED_VERSION=$(normalize_version "$V1_VERSION")

  if [[ "$NORMALIZED_HEIMDALLCLI_VERSION" != "$NORMALIZED_EXPECTED_VERSION" ]]; then
    handle_error $STEP "heimdallcli version mismatch! Expected: $V1_VERSION, Found: $HEIMDALLCLI_VERSION"
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
    handle_error $STEP "heimdalld version mismatch! Expected: $V1_VERSION, Found: $HEIMDALLD_VERSION"
  fi
fi
# NETWORK
if [[ "$NETWORK" != "amoy" && "$NETWORK" != "mainnet" ]]; then
    handle_error $STEP "Invalid network! Must be 'amoy' or 'mainnet'."
fi
# V1_CHAIN_ID validation (only determine EXPECTED_V1_CHAIN_ID if V1_CHAIN_ID is not "devnet")
if [[ "$V1_CHAIN_ID" != "devnet" ]]; then
    case "$NETWORK" in
        mainnet)
            EXPECTED_V1_CHAIN_ID="heimdall-137"
            ;;
        amoy)
            EXPECTED_V1_CHAIN_ID="heimdall-80002"
            ;;
        *)
            # For any other network, fallback to devnet
            EXPECTED_V1_CHAIN_ID="devnet"
            ;;
    esac
fi
if [[ -n "$EXPECTED_V1_CHAIN_ID" ]]; then
    if [[ "$V1_CHAIN_ID" != "$EXPECTED_V1_CHAIN_ID" ]]; then
        echo "❌ Chain ID mismatch: expected '$EXPECTED_V1_CHAIN_ID', got '$V1_CHAIN_ID'"
        exit 1
    fi
fi
# NODETYPE
if [[ "$NODETYPE" != "sentry" && "$NODETYPE" != "validator" ]]; then
    handle_error $STEP "Invalid node type! Must be 'sentry' or 'validator'."
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
echo "       NETWORK:               $NETWORK"
echo "       NODETYPE:              $NODETYPE"
echo "       HEIMDALL_SERVICE_USER: $HEIMDALL_SERVICE_USER"
echo "       GENERATE_GENESIS:      $GENERATE_GENESIS"
echo "       V1_CHAIN_ID:           $V1_CHAIN_ID"
echo "       V2_CHAIN_ID:           $V2_CHAIN_ID"
echo "       V2_GENESIS_TIME:       $V2_GENESIS_TIME"
echo ""


# Step 3: stop heimdall-v1. The apocalypse tag embeds the halt_height so heimdalld should be down already, running it for consistency/completeness
STEP=3
print_step $STEP "Backing up service file and stopping heimdall-v1"
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
HEIMDALLD_UNIT_FILE="/lib/systemd/system/heimdalld.service"
HEIMDALLD_UNIT_BACKUP="/lib/systemd/system/heimdalld.service.backup"
if [ -f "$HEIMDALLD_UNIT_FILE" ]; then
    if [ -f "$HEIMDALLD_UNIT_BACKUP" ]; then
        echo "[INFO] Backup already exists at $HEIMDALLD_UNIT_BACKUP"
    else
        cp "$HEIMDALLD_UNIT_FILE" "$HEIMDALLD_UNIT_BACKUP"
        echo "[INFO] Backed up $HEIMDALLD_UNIT_FILE to $HEIMDALLD_UNIT_BACKUP"
    fi
else
    echo "[WARNING] $HEIMDALLD_UNIT_FILE not found. Nothing to back up."
fi

# Step 4: Ensure node has committed up to latest height
STEP=4
print_step $STEP "Checking that Heimdall v1 has committed the latest height"

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
    if [[ "$COMMITTED_HEIGHT" -lt "$V1_HALT_HEIGHT" ]]; then
        echo "[WARN] Node has not yet committed the apocalypse height."
        echo "       Expected: $V1_HALT_HEIGHT"
        echo "       Found:    $COMMITTED_HEIGHT"
        echo "       This node will NOT generate its own genesis file."
        GENERATE_GENESIS=false
    else
        echo "[INFO] Node has committed the apocalypse height."
        echo "       Expected: $V1_HALT_HEIGHT"
        echo "       Found:    $COMMITTED_HEIGHT"
        echo "       This node will generate its own genesis file."
        GENERATE_GENESIS=true
    fi
fi

# Step 5: Generate or download Heimdall v1 genesis JSON
STEP=5
print_step $STEP "Obtaining Heimdall v1 genesis JSON file"
GENESIS_FILE="$HEIMDALL_HOME/$DUMP_V1_GENESIS_FILE_NAME"
if $GENERATE_GENESIS; then
    echo "[INFO] Generating genesis file using heimdalld export..."
    if ! $HEIMDALL_CLI_PATH export-heimdall --home "$HEIMDALL_HOME" --chain-id "$V1_CHAIN_ID"; then
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
V1_GENESIS_CHECKSUM_FILE="$HEIMDALL_HOME/$DUMP_V1_GENESIS_FILE_NAME.sha512"
# Ensure the genesis file exists before computing checksum
if [[ ! -f "$GENESIS_FILE" ]]; then
    handle_error $STEP "Genesis file $GENESIS_FILE not found. Cannot generate checksum."
fi
# execute command
sha512sum "$GENESIS_FILE" | awk '{print $1}' > "$V1_GENESIS_CHECKSUM_FILE"
# Verify checksum file exists and is not empty
if [[ ! -s "$V1_GENESIS_CHECKSUM_FILE" ]]; then
    handle_error $STEP "Checksum file was not created or is empty."
fi
GENERATED_V1_GENESIS_CHECKSUM=$(awk '{print $1}' "$V1_GENESIS_CHECKSUM_FILE")
# Print checksum
echo "[INFO] Generated checksum: $GENERATED_V1_GENESIS_CHECKSUM"


# Step 7: verify checksum
STEP=7
print_step $STEP "Verifying checksum"
if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY-RUN] Skipping checksum verification"
else
    V1_GENESIS_CHECKSUM_FILE="$HEIMDALL_HOME/$DUMP_V1_GENESIS_FILE_NAME.sha512"
    # Ensure checksum file exists before reading it
    if [[ ! -f "$V1_GENESIS_CHECKSUM_FILE" ]]; then
        handle_error $STEP "Checksum file $V1_GENESIS_CHECKSUM_FILE not found! Cannot verify checksum."
    fi
    # Read expected checksum from the file
    V1_GENESIS_CHECKSUM=$(awk '{print $1}' "$V1_GENESIS_CHECKSUM_FILE")
    # Verify checksum matches the generated one
    if [[ "$GENERATED_V1_GENESIS_CHECKSUM" != "$V1_GENESIS_CHECKSUM" ]]; then
        handle_error $STEP "Checksum mismatch! Expected: $V1_GENESIS_CHECKSUM, Found: $GENERATED_V1_GENESIS_CHECKSUM"
    fi
    echo "[INFO] Checksum verification passed."
fi

# Step 8: move heimdall-v1 to backup location
STEP=8
BACKUP_DIR="${HEIMDALL_HOME}.backup"
print_step $STEP "Moving $HEIMDALL_HOME to $BACKUP_DIR"
# Create parent directory in case it doesn't exist
sudo mkdir -p "$(dirname "$BACKUP_DIR")" || handle_error $STEP "Failed to create parent directory for $BACKUP_DIR"
# Move Heimdall home to backup location
sudo mv "$HEIMDALL_HOME" "$BACKUP_DIR" || handle_error $STEP "Failed to move $HEIMDALL_HOME to $BACKUP_DIR"
echo "[INFO] Backup (move) completed successfully."

# Step 9 : select the proper heimdall-v2 binary package
STEP=9
print_step $STEP "Create temp directory for heimdall-v2 and target the right package based on current system"
tmpDir="/tmp/tmp-heimdall-v2"
sudo mkdir -p $tmpDir || handle_error $STEP "Cannot create $tmpDir directory for downloading files"
profileInfo=${NETWORK}-${NODETYPE}-config_v${V2_VERSION}
profileInforpm=${NETWORK}-${NODETYPE}-config-v${V2_VERSION}
baseUrl="https://github.com/0xPolygon/heimdall-v2/releases/download/v${V2_VERSION}"
case "$(uname -s).$(uname -m)" in
    Linux.x86_64)
        if command -v dpkg &> /dev/null; then
            type="deb"
            binary="heimdall-v${V2_VERSION}-amd64.deb"
            profile="heimdall-${profileInfo}-all.deb"
        elif command -v rpm &> /dev/null; then
            type="rpm"
            binary="heimdall-v${V2_VERSION}.x86_64.rpm"
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
            binary="heimdall-v${V2_VERSION}-arm64.deb"
            profile="heimdall-${profileInfo}-all.deb"
        elif command -v rpm &> /dev/null; then
            type="rpm"
            binary="heimdall-v${V2_VERSION}.aarch64.rpm"
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
if [ -n "$profile"  ]; then
    profileUrl="${baseUrl}/${profile}"
    profilePackage=$tmpDir/$profile
    curl -L "$profileUrl" -o "$profilePackage" || handle_error $STEP "Failed to download profile from \"$profileUrl\""
fi

# Step 11: install heimdall-v2 binary
STEP=11
print_step $STEP "Install heimdall-v2 binary"
if [ "$type" = "deb" ]; then
    echo "[INFO] Uninstalling any existing heimdalld..."
    sudo dpkg -r heimdalld heimdall || echo "[WARN] Nothing to uninstall"
    echo "[INFO] Installing $package..."
    sudo dpkg -i "$package" || handle_error $STEP "Failed to install $package"
    if [ -n "$profilePackage" ] && [ ! -d /var/lib/heimdall/config ]; then
        echo "[INFO] Installing profile package..."
        sudo dpkg -i "$profilePackage" || handle_error $STEP "Failed to install profile package"
    fi
elif [ "$type" = "rpm" ]; then
    echo "[INFO] Uninstalling any existing heimdalld..."
    sudo rpm -e heimdall || echo "[WARN] Nothing to uninstall"
    echo "[INFO] Installing $package..."
    sudo rpm -i --force "$package" || handle_error $STEP "Failed to install $package"
    if [ -n "$profilePackage" ] && [ ! -d /var/lib/heimdall/config ]; then
        echo "[INFO] Installing profile package..."
        sudo rpm -i --force "$profilePackage" || handle_error $STEP "Failed to install profile package"
    fi
elif [ "$type" = "apk" ]; then
    echo "[INFO] Installing $package..."
    sudo apk add --allow-untrusted "$package" || handle_error $STEP "Failed to install $package"
elif [ "$type" = "tar.gz" ]; then
    unpack="$tmpDir/unpack"
    mkdir -p "$unpack"
    tar -xzf "$package" -C "$unpack" || handle_error $STEP "Failed to unpack"
    echo "[INFO] Copying binary to /usr/local/bin/heimdalld"
    sudo cp "$unpack/heimdalld" /usr/local/bin/heimdalld || handle_error $STEP "Failed to copy binary to /usr/local/bin/heimdalld"
    sudo chmod +x /usr/local/bin/heimdalld
    echo "[INFO] Copying binary to /usr/bin/heimdalld"
    sudo cp "$unpack/heimdalld" /usr/bin/heimdalld || handle_error $STEP "Failed to copy binary to /usr/bin/heimdalld"
    sudo chmod +x /usr/bin/heimdalld
else
    handle_error $STEP "Unknown package type: $type"
fi
echo "[INFO] Heimdall-v2 binary installation completed"

# Step 12: copy binary to user-specified path
STEP=12
print_step $STEP "Copying heimdalld binary to $HEIMDALLD_PATH"
# Determine source binary path - prefer /usr/bin/heimdalld
if [ -x "/usr/bin/heimdalld" ]; then
    SOURCE_BINARY="/usr/bin/heimdalld"
elif [ -x "/usr/local/bin/heimdalld" ]; then
    SOURCE_BINARY="/usr/local/bin/heimdalld"
else
    # Fallback to command resolution
    SOURCE_BINARY=$(command -v heimdalld)
    if [[ -z "$SOURCE_BINARY" ]] || [[ ! -x "$SOURCE_BINARY" ]]; then
        handle_error $STEP "Could not find heimdalld binary after installation"
    fi
fi
echo "[INFO] Using source binary at: $SOURCE_BINARY"
# Verify binary is valid
file_type=$(file "$SOURCE_BINARY")
echo "[DEBUG] Source binary type: $file_type"
if [[ "$file_type" != *"ELF 64-bit"* ]]; then
    handle_error $STEP "Source heimdalld binary is not valid: $file_type"
fi
# Check target directory exists
dir_path=$(dirname "$HEIMDALLD_PATH")
if [ ! -d "$dir_path" ]; then
    handle_error $STEP "Target directory $dir_path does not exist"
fi
# Skip copy if source and destination are the same
if [[ "$SOURCE_BINARY" = "$HEIMDALLD_PATH" ]]; then
    echo "[INFO] Binary is already at target location: $HEIMDALLD_PATH"
else
    if [ -f "$HEIMDALLD_PATH" ]; then
        echo "[INFO] Backing up existing heimdalld binary"
        sudo mv "$HEIMDALLD_PATH" "${HEIMDALLD_PATH}.bak" || handle_error $STEP "Backup failed"
    fi
    echo "[INFO] Copying $SOURCE_BINARY → $HEIMDALLD_PATH"
    sudo cp "$SOURCE_BINARY" "$HEIMDALLD_PATH" || handle_error $STEP "Failed to copy binary"
    sudo chmod +x "$HEIMDALLD_PATH" || handle_error $STEP "Failed to chmod binary"
fi
echo "[INFO] heimdalld binary installed at $HEIMDALLD_PATH"


# Step 13: verify heimdall-v2 version
STEP=13
print_step $STEP "Verifying Heimdall v2 version"
# Check if heimdalld is installed and executable
if [[ ! -x "$HEIMDALLD_PATH" ]]; then
    handle_error $STEP "Heimdalld binary is missing or not executable: $HEIMDALLD_PATH"
fi
# Check heimdalld version
echo "[INFO] Checking heimdalld version..."
HEIMDALLD_V2_VERSION_RAW=$("$HEIMDALLD_PATH" version 2>/dev/null | awk 'NF' | tail -n 1)
if [[ -z "$HEIMDALLD_V2_VERSION_RAW" ]]; then
    echo "[ERROR] Failed to retrieve Heimdall v2 version"
    echo "[DEBUG] Testing binary execution:"
    "$HEIMDALLD_PATH" version 2>&1 || echo "Binary execution failed"
    handle_error $STEP "Failed to retrieve Heimdall v2 version. Installation may have failed."
fi
# Normalize actual and expected versions
NORMALIZED_HEIMDALLD_V2_VERSION=$(normalize_version "$HEIMDALLD_V2_VERSION_RAW")
NORMALIZED_EXPECTED_V2_VERSION=$(normalize_version "$V2_VERSION")
echo "[DEBUG] Expected version (normalized): $NORMALIZED_EXPECTED_V2_VERSION"
echo "[DEBUG] Found version (normalized): $NORMALIZED_HEIMDALLD_V2_VERSION"
if [[ "$NORMALIZED_HEIMDALLD_V2_VERSION" != "$NORMALIZED_EXPECTED_V2_VERSION" ]]; then
    handle_error $STEP "Heimdall v2 version mismatch! Expected: $V2_VERSION, Found: $HEIMDALLD_V2_VERSION_RAW"
fi
# Ensure HEIMDALL_HOME exists
if [[ ! -d "/var/lib/heimdall" ]]; then
    handle_error $STEP "/var/lib/heimdall does not exist after installation."
fi
echo "[INFO] heimdall-v2 is using the correct version $V2_VERSION"

# Step 14: migrate genesis file
STEP=14
print_step $STEP "Migrating Heimdall v1 genesis file $GENESIS_FILE to v2 format, the result file will be saved in $BACKUP_DIR/migrated_$DUMP_V1_GENESIS_FILE_NAME"
# Define the target output file
MIGRATED_GENESIS_FILE="$BACKUP_DIR/migrated_$DUMP_V1_GENESIS_FILE_NAME"
# Ensure the v1 genesis file exists before proceeding
if [[ ! -f "$BACKUP_DIR/$DUMP_V1_GENESIS_FILE_NAME" ]]; then
    handle_error $STEP "Genesis file $BACKUP_DIR/$DUMP_V1_GENESIS_FILE_NAME not found! Cannot proceed with migration."
fi
# Sanity check: warn if V2_GENESIS_TIME is in the future
GENESIS_TIMESTAMP=$(date -d "$V2_GENESIS_TIME" +%s)
NOW_TIMESTAMP=$(date +%s)
if (( GENESIS_TIMESTAMP > NOW_TIMESTAMP )); then
    echo "[WARNING] V2_GENESIS_TIME is in the future: $V2_GENESIS_TIME"
    echo "          This may cause Heimdall to sleep until that time on startup."
fi
# Run the migration command
if ! heimdalld migrate "$BACKUP_DIR/$DUMP_V1_GENESIS_FILE_NAME" --chain-id="$V2_CHAIN_ID" --genesis-time="$V2_GENESIS_TIME" --initial-height="$V2_INITIAL_HEIGHT" --verify-data="$VERIFY_EXPORTED_DATA"; then
    handle_error $STEP "Migration command failed."
fi
echo "[INFO] Genesis file migrated successfully from v1 to v2"
# ensure migrated genesis file exists
if [[ ! -f "$MIGRATED_GENESIS_FILE" ]]; then
    handle_error $STEP "Expected migrated genesis file not found at $MIGRATED_GENESIS_FILE"
fi
# Confirm initial_height in migrated genesis matches configured V2_INITIAL_HEIGHT
echo "[INFO] Verifying initial_height in migrated genesis file..."
ACTUAL_V2_INITIAL_HEIGHT=$(jq -r '.initial_height' "$MIGRATED_GENESIS_FILE")
[[ "$ACTUAL_V2_INITIAL_HEIGHT" =~ ^[0-9]+$ ]] || handle_error $STEP "Failed to parse initial_height from migrated genesis."
if [[ "$ACTUAL_V2_INITIAL_HEIGHT" != "$V2_INITIAL_HEIGHT" ]]; then
    echo "[WARNING] Mismatch detected!"
    echo "          Configured V2_INITIAL_HEIGHT: $V2_INITIAL_HEIGHT"
    echo "          Genesis file contains:     $ACTUAL_V2_INITIAL_HEIGHT"
    handle_error $STEP "V2_INITIAL_HEIGHT mismatch detected"
else
    echo "[INFO] initial_height in genesis matches expected value: $V2_INITIAL_HEIGHT"
fi


# Step 15: Generate checksum of the migrated genesis
STEP=15
print_step $STEP "Generating checksum for Heimdall v2 genesis file, it will be saved in $MIGRATED_GENESIS_FILE.sha512"
V2_CHAIN_IDGENESIS_CHECKSUM_FILE="$MIGRATED_GENESIS_FILE.sha512"
# Ensure the genesis file exists before computing checksum
if [[ ! -f "$MIGRATED_GENESIS_FILE" ]]; then
    handle_error $STEP "Migrated genesis file $MIGRATED_GENESIS_FILE not found. Cannot generate checksum."
fi
# execute command
sha512sum "$MIGRATED_GENESIS_FILE" | awk '{print $1}' > "$V2_CHAIN_IDGENESIS_CHECKSUM_FILE"
# Verify checksum file exists and is not empty
if [[ ! -s "$V2_CHAIN_IDGENESIS_CHECKSUM_FILE" ]]; then
    handle_error $STEP "Checksum file was not created or is empty."
fi
GENERATED_V2_CHAIN_IDGENESIS_CHECKSUM=$(awk '{print $1}' "$V2_CHAIN_IDGENESIS_CHECKSUM_FILE")
# Print checksum
echo "[INFO] Generated checksum: $GENERATED_V2_CHAIN_IDGENESIS_CHECKSUM"


# Step 16: verify checksum of the migrated genesis
STEP=16
print_step $STEP "Verifying checksum of the migrated genesis file"
if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY-RUN] Skipping checksum verification"
else
    V2_CHAIN_IDGENESIS_CHECKSUM_FILE="$MIGRATED_GENESIS_FILE.sha512"
    # Ensure checksum file exists before reading it
    if [[ ! -f "$V2_CHAIN_IDGENESIS_CHECKSUM_FILE" ]]; then
        handle_error $STEP "Checksum file $V2_CHAIN_IDGENESIS_CHECKSUM_FILE not found! Cannot verify checksum."
    fi
    # Read expected checksum from the file
    V2_GENESIS_CHECKSUM=$(awk '{print $1}' "$V2_CHAIN_IDGENESIS_CHECKSUM_FILE")
    # Verify checksum matches the generated one
    if [[ "$GENERATED_V2_CHAIN_IDGENESIS_CHECKSUM" != "$V2_GENESIS_CHECKSUM" ]]; then
        handle_error $STEP "Checksum mismatch! Expected: $V2_GENESIS_CHECKSUM, Found: $GENERATED_V2_CHAIN_IDGENESIS_CHECKSUM"
    fi
    echo "[INFO] Checksum verification passed."
fi


# Step 17: create temp heimdall-v2 home dir
STEP=17
print_step $STEP "Creating temp directory for heimdall-v2 in and applying proper permissions"
V2_HOME="/var/lib/heimdall"
sudo mkdir -p "/var/lib/heimdall" || handle_error $STEP "Failed to create $V2_HOME directory"
# apply proper permissions for the current user
sudo chmod -R 755 "$V2_HOME" || handle_error $STEP "Failed to set permissions"
sudo chown -R "$HEIMDALL_SERVICE_USER" "$V2_HOME" || handle_error $STEP "Failed to change ownership"
echo "[INFO] $V2_HOME created successfully"
if [ -f "$V2_HOME/config/genesis.json" ]; then
    sudo rm "$V2_HOME/config/genesis.json" || handle_error $STEP "Failed to remove existing json in $V2_HOME/config"
fi

# Step 18: init heimdall-v2
STEP=18
print_step $STEP "Initializing heimdalld for version v2"
# Ensure Heimdall home exists before proceeding
if [[ ! -d "/var/lib/heimdall" ]]; then
    handle_error $STEP "/var/lib/heimdall does not exist. Cannot proceed with initialization."
fi
# Init Heimdall v2
if ! heimdalld init "temp_moniker" --chain-id="$V2_CHAIN_ID" --home="$V2_HOME"; then
    handle_error $STEP "Failed to initialize heimdalld."
fi
echo "[INFO] heimdalld initialized successfully."


# Step 19: verify required directories exist
STEP=19
print_step $STEP "Verifying required directories and configuration files in /var/lib/heimdall"
# Check if required directories exist
REQUIRED_DIRS=("data" "config")
for dir in "${REQUIRED_DIRS[@]}"; do
    if [[ ! -d "/var/lib/heimdall/$dir" ]]; then
        handle_error $STEP "Required directory is missing: /var/lib/heimdall/$dir"
    fi
done
# Ensure config directory contains the necessary files
REQUIRED_CONFIG_FILES=("app.toml" "client.toml" "config.toml" "genesis.json" "node_key.json" "priv_validator_key.json")
for file in "${REQUIRED_CONFIG_FILES[@]}"; do
    if [[ ! -f "/var/lib/heimdall/config/$file" ]]; then
        handle_error $STEP "Missing required configuration file: $file"
    fi
done
# Ensure data directory contains the necessary files
REQUIRED_DATA_FILES=("priv_validator_state.json")
for file in "${REQUIRED_DATA_FILES[@]}"; do
    if [[ ! -f "/var/lib/heimdall//data/$file" ]]; then
        handle_error $STEP "Missing required data file: $file"
    fi
done
echo "[INFO] All required directories are present in var/lib/heimdall"


# Step 20: Restore bridge directory from backup
STEP=20
print_step $STEP "Restoring bridge directory from backup if present"
BRIDGE_SRC="$BACKUP_DIR/bridge"
BRIDGE_DEST="/var/lib/heimdall/bridge"

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
print_step $STEP "Moving genesis file to /var/lib/heimdall"
TARGET_GENESIS_FILE="/var/lib/heimdall/config/genesis.json"
# Backup existing genesis file before replacing it
if [ -f "$TARGET_GENESIS_FILE" ]; then
    echo "[INFO] Backing up existing genesis file..."
    mv "$TARGET_GENESIS_FILE" "${TARGET_GENESIS_FILE}.bak"
    echo "[INFO] Backup saved at: $TARGET_GENESIS_FILE.bak"
fi
# Replace with the migrated genesis
cp -p "$MIGRATED_GENESIS_FILE" "$TARGET_GENESIS_FILE" || handle_error $STEP "Failed to replace genesis file with migrated version."


# Step 22: edit priv_validator_key.json file according to v2 setup
STEP=22
print_step $STEP "Updating priv_validator_key.json file"
PRIV_VALIDATOR_FILE="/var/lib/heimdall/config/priv_validator_key.json"
TEMP_PRIV_FILE="temp_priv_validator_key.json"
TEMP_FILES+=("$TEMP_PRIV_FILE")

if [ -f "$PRIV_VALIDATOR_FILE" ]; then
    echo "[INFO] Creating backup of priv_validator_key.json..."
    sudo cp "$PRIV_VALIDATOR_FILE" "$PRIV_VALIDATOR_FILE.bak" || handle_error $STEP "Failed to backup priv_validator_key.json"
    echo "[INFO] Backup saved at: $PRIV_VALIDATOR_FILE.bak"
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
NODE_KEY_FILE="/var/lib/heimdall/config/node_key.json"
TEMP_NODE_KEY_FILE="temp_node_key.json"
TEMP_FILES+=("$TEMP_NODE_KEY_FILE")
if [ -f "$NODE_KEY_FILE" ]; then
    echo "[INFO] Creating backup of node_key.json..."
    cp "$NODE_KEY_FILE" "$NODE_KEY_FILE.bak" || handle_error $STEP "Failed to backup node_key.json"
    echo "[INFO] Backup saved at: $NODE_KEY_FILE.bak"
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
PRIV_VALIDATOR_STATE="/var/lib/heimdall/data/priv_validator_state.json"
TEMP_STATE_FILE="temp_priv_validator_state.json"
TEMP_FILES+=("$TEMP_STATE_FILE")
if [ ! -f "$PRIV_VALIDATOR_STATE" ]; then
    handle_error $STEP "priv_validator_state.json not found in /var/lib/heimdall/data/"
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
    mv "$TEMP_STATE_FILE" "$PRIV_VALIDATOR_STATE" || handle_error $STEP "Failed to move updated priv_validator_state.json into place"
else
    handle_error $STEP "Failed to update priv_validator_state.json"
fi
echo "[INFO] Successfully updated priv_validator_state.json"


# Step 25: Restore addrbook.json from backup if it exists
STEP=25
print_step $STEP "Restoring addrbook.json from backup (if present)"
ADDRBOOK_FILE="$BACKUP_DIR/config/addrbook.json"
TARGET_ADDRBOOK_FILE="/var/lib/heimdall/config/addrbook.json"
if [ -f "$ADDRBOOK_FILE" ]; then
    # Backup current one (if any)
    if [ -f "$TARGET_ADDRBOOK_FILE" ]; then
        cp "$TARGET_ADDRBOOK_FILE" "$TARGET_ADDRBOOK_FILE.bak" || handle_error $STEP "Failed to backup existing addrbook.json"
        echo "[INFO] Backup saved at: $TARGET_ADDRBOOK_FILE.bak"
    fi
    cp "$ADDRBOOK_FILE" "$TARGET_ADDRBOOK_FILE" || handle_error $STEP "Failed to restore addrbook.json from backup"
    echo "[INFO] addrbook.json restored successfully."
else
    echo "[INFO] No addrbook.json found in backup. Skipping restore."
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
echo -e "   - chain-id = \"$V2_CHAIN_ID\"\n"
echo -e "💡 You may manually edit other parameters (e.g. ports, metrics, logging) after migration."
echo -e "\n📁 \033[1mBor Configuration Notice\033[0m:"
echo -e "   ⚠️  Please update your Bor's \033[1mbor/config.toml\033[0m manually to reflect v2-compatible settings."
echo -e "   💡 You can optionally add the following entry under the \033[1m[heimdall]\033[0m section to enable WebSocket support:"
echo -e "\n     [heimdall]"
echo -e "     ws-address = \"ws://localhost:26657/websocket\"\n"
echo -e "   ✅ This setting is recommended, as it improves performance by reducing the number of HTTP polling requests from Heimdall to Bor."
echo -e "   🔄 After updating the config, make sure to restart your Bor node for changes to take effect.\n"
# 1. Set chain-id in client.toml
CLIENT_TOML="/var/lib/heimdall/config/client.toml"
echo "[INFO] Setting chain-id in client.toml..."
set_toml_key "$CLIENT_TOML" "chain-id" "$V2_CHAIN_ID"
actual_chain_id=$(grep -E '^chain-id\s*=' "$CLIENT_TOML" | cut -d'=' -f2 | tr -d ' "')
if [[ "$actual_chain_id" != "$V2_CHAIN_ID" ]]; then
    echo "[WARN] Validation failed: expected chain-id = $V2_CHAIN_ID, found $actual_chain_id"
fi
echo "[OK]   client.toml: chain-id = $V2_CHAIN_ID"
# 2. Migrate config.toml keys
OLD_CONFIG_TOML="$BACKUP_DIR/config/config.toml"
NEW_CONFIG_TOML="/var/lib/heimdall/config/config.toml"
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
OLD_HEIMDALL_CONFIG_TOML="$BACKUP_DIR/config/heimdall-config.toml"
NEW_APP_TOML="/var/lib/heimdall/config/app.toml"
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
        echo "[WARN] Validation failed for '$key' in app.toml: expected '$expected', got '$actual'"
    fi
done
echo "[INFO] app.toml values migrated successfully."


# Step 27: Assign correct ownership to Heimdall directories
STEP=27
print_step $STEP "Assigning correct ownership and permissions under /var/lib/heimdall as user: $HEIMDALL_SERVICE_USER"
# Sanity check: avoid chowning critical paths
CRITICAL_PATHS=("/" "/usr" "/usr/bin" "/bin" "/lib" "/lib64" "/etc" "/boot")
for path in "${CRITICAL_PATHS[@]}"; do
    if [[ "/var/lib/heimdall" == "$path" ]]; then
        handle_error $STEP "Refusing to chown critical system path: $path"
    fi
done
echo "[INFO] Recursively setting ownership of all contents in /var/lib/heimdall to $HEIMDALL_SERVICE_USER"
sudo chown -R "$HEIMDALL_SERVICE_USER":"$HEIMDALL_SERVICE_USER" "/var/lib/heimdall" || handle_error $STEP "Failed to chown /var/lib/heimdall"
# Set 600 permissions for all files
echo "[INFO] Setting 600 permissions for all files under /var/lib/heimdall"
find "/var/lib/heimdall" -type f ! -name '.*' -exec chmod 600 {} \; || handle_error $STEP "Failed to chmod files"
# Set 700 permissions for all directories
echo "[INFO] Setting 700 permissions for all directories under /var/lib/heimdall"
find "/var/lib/heimdall" -type d ! -name '.*' -exec chmod 700 {} \; || handle_error $STEP "Failed to chmod directories"
echo "[INFO] Ownership and permissions successfully enforced under /var/lib/heimdall"

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


# Step 29: Clean up .bak files in /var/lib/heimdall
STEP=29
print_step $STEP "Cleaning up .bak files in parent directory of /var/lib/heimdall"
# Determine the parent directory of /var/lib/heimdall
HEIMDALL_PARENT_DIR=$(dirname "/var/lib/heimdall")
# Find and delete all .bak files or directories
BAK_FILES=$(find "$HEIMDALL_PARENT_DIR" -name "*.bak")
if [[ -n "$BAK_FILES" ]]; then
    echo "[INFO] Removing the following backup files or directories:"
    echo "$BAK_FILES"
    find "$HEIMDALL_PARENT_DIR" -name "*.bak" -exec rm -rf {} \;
    echo "[INFO] Cleanup complete."
else
    echo "[INFO] No .bak files or directories found in $HEIMDALL_PARENT_DIR"
fi

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

echo -e "\n⚠️  \033[1mManual Verification Required\033[0m:"
echo -e "   Please review the updated configuration files under:"
echo -e "     \033[1m/var/lib/heimdall/config/\033[0m"
echo -e "   and ensure that they match your expected custom values from:"
echo -e "     \033[1m/var/lib/heimdall/config/\033[0m"
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
echo -e "   - chain-id = \"$V2_CHAIN_ID\"\n"

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
