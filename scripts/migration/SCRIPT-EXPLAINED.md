# Migration Script 101

## Prerequisites

## 🔐 Secure Permissions with `umask`

```bash
umask 0022

```

This sets the default permissions for any newly created files and directories during the script run. It ensures:

- Files are created with permission `644` (read/write for owner, read-only for others).
- Directories are created with permission `755` (read/write/execute for owner, read/execute for others).

This is a **safe default** for most system files.

## ✅ Script Integrity Check

```bash
SCRIPT_PATH=$(realpath "$0")

if ! tail -n 10 "$SCRIPT_PATH" | grep -q "# End of script"; then
    echo "[ERROR] Script appears to be incomplete or partially downloaded."
    exit 1
fi

```

This block ensures you're running the **entire script** and not just part of it. If the script was copied or downloaded incompletely, it could cause a broken migration — this avoids that risk by looking for a known end marker comment.

## 🔐 Root Access Enforcement

```bash
if [[ "$(id -u)" -ne 0 ]]; then
    echo "[ERROR] This script must be run as root. Use sudo."
    exit 1
fi

```

Many operations — like changing file ownership, restarting services, or writing to system directories — **require root privileges**. This early check ensures you're using `sudo`, avoiding permission errors later.

## ⚙️ Default Configuration Parameters

The script sets several default values for directories, binaries, service names, and expected software versions:

```bash
DEFAULT_HEIMDALL_HOME="/var/lib/heimdall"
DEFAULT_NETWORK="amoy"
DEFAULT_NODETYPE="sentry"
DEFAULT_MONIKER_NODE_NAME="polygon-pos-node"
DEFAULT_BOR_PATH="/usr/bin/bor"
DEFAULT_HEIMDALLD_PATH="/usr/bin/heimdalld"
DEFAULT_HEIMDALLCLI_PATH="/usr/bin/heimdallcli"
DEFAULT_HEIMDALL_SERVICE_USER="heimdall"

```

## 📌 Expected Versions and Constants

To avoid misconfigurations or incompatibilities, the script declares:

```bash
# ...
APOCALYPSE_TAG="1.2.0-41-g5d576637"
REQUIRED_BOR_VERSION="2.1.0"
REQUIRED_HEIMDALLCLI_VERSION="v1.0.10"
HEIMDALL_V2_VERSION="0.1.9"
CHECKSUM="b898f317ffd9f78002e8660e7890e13a6d3ad21c325c4fa8fc246de6e4d745a55c465633a075d66e6a1aa7813fc7431638654370626be123bd2d1767cc165486"
# ...

```

These represent:

- **Required versions** for Bor and Heimdall CLI.
- **Known good versions** for the migration.
- A **checksum** to verify the exported genesis file integrity .

## 🔧 Migration-Specific Values

```bash
CHAIN_ID="heimdall-v2-chain-id"
GENESIS_TIME="2024-03-18T00:00:00Z"
INITIAL_HEIGHT="21"
VERIFY_HASH=false
DUMP_V1_GENESIS_FILE_NAME="dump-genesis.json"
DRY_RUN="true"

```

These are constants used internally for the migration, like:

- New `chain-id`
- Hardcoded `genesis time` and `initial block height` (agreed during the governance proposal)
- Whether to **verify the hash** of the v1 genesis dump
- Dry-run toggle for avoid some checks (only used for testing and should be set to false when in production)

## 🔙 Rollback Support

```bash
ROLLBACK_ACTIONS=()
LAST_STEP_EXECUTED=0

```

These variables are used to:

- **Track which steps** have been executed
- **Store rollback commands** to revert changes step-by-step if something goes wrong

## 🧹 Temp File Cleanup

```bash
TEMP_FILES=()
trap cleanup_temp_files EXIT

```

If the script creates temporary files (e.g., downloaded binaries, temp configs), they’re tracked and **automatically deleted** at the end or on error.

## 🧪 Utility Functions

### `print_step`

Prints each step clearly with a timestamp and stores the step number for potential rollback.

### `handle_error`

Logs an error, triggers rollback, and exits safely.

### `rollback`

Reverses actions from the last executed step back to the first, based on registered rollback commands.

### `validate_absolute_path`

Ensures that any user-provided path is an **absolute path** (starts with `/`). This prevents logic errors or accidental overwrites.

### `version_ge`

A helper to compare versions using natural sort order.

### 🔍 Step 1: Dependency Check

```bash
DEPENDENCIES=("curl" "tar" "jq" "sha512sum")
```

The script checks for the presence of four essential command-line tools:

| Dependency | Why It's Needed |
| --- | --- |
| `curl` | To download binaries and files |
| `tar` | To extract compressed files (e.g., Heimdall v2 release tarballs) |
| `jq` | To parse and manipulate JSON data (used for handling genesis files, configs) |
| `sha512sum` | To verify binary integrity using cryptographic checksums |

If any of these are missing, the script **stops immediately** with an error, like:

```bash
[ERROR] Step 1 failed: Missing dependencies: jq curl. Please install them and rerun the script.

```

This avoids continuing the migration in an incomplete or broken environment.

If everything is present, the script confirms:

```bash
[INFO] All required dependencies are installed.

```

> ✅ Tip for non-Linux operators: If you're running this on macOS or a different UNIX system, ensure these tools are available in your PATH (they can usually be installed via brew or pkg). If you're running inside Docker, this script is not supported and you'll need to handle each step manually.
>

## ⚙️ Step 2: Validating the provided arguments

In this step, the script validates all the required inputs for a successful migration. These values are used throughout the rest of the process to tailor operations to your node's environment.

### 🗂️ HEIMDALL_HOME

You are asked to provide the absolute path to your Heimdall v1 data directory. This directory **must contain both `config/` and `data/` subdirectories**. If these are missing, the script exits early.

### 🧰 heimdallcli Binary

It validates:

- The path is absolute.
- The binary is executable.
- The version is **at least `v1.0.10`**.

If any check fails, the script aborts.

### 🧰 heimdalld Binary

Same logic as `heimdallcli`. The script:

- Verifies it is executable.
- Validates that the version matches exactly `APOCALYPSE_TAG`, unless in dry-run mode.

### 💻 BOR Binary

The script asks whether `bor` is running locally via an optional arg.

- If yes:
    - Verifies it’s executable and version equals `REQUIRED_BOR_VERSION`.

### 🌐 Network Selection

`amoy` or `mainnet`. The input is validated strictly.

### 🧱 Node Type

You must choose between `sentry` or `validator`, which determines how some configuration and identity files are handled later.

### 🗃️ Backup Directory

You are prompted for a directory to store a full backup of the v1 Heimdall state. The script ensures this directory:

- Is an absolute path.
- Is **not equal** to `HEIMDALL_HOME` (to avoid overwriting).

### 🏷️ Node Moniker

You provide a node moniker that identifies your Heimdall v2 node. It cannot be empty, and it's must match v1 value which can be found in `HEIMDALL_HOME/config/config.toml`.

### 👤 Heimdall System User

The script requires the **system user that runs the Heimdall service** (e.g., `heimdall`). This is used later to fix permissions on all migrated files. The user must exist on the system.

## 🛑 Step 3: Stopping Heimdall v1

In this step, the script ensures that your Heimdall v1 node is fully stopped before the migration proceeds. This is required to guarantee the integrity of the exported state and to avoid file corruption during backup and data manipulation.

Even though the tag for Heimdall v1 is configured with a `halt_height` (which should cause the node to shut down automatically), this step **explicitly stops** the `heimdalld` service for completeness and safety.

### 🧩 What it does

The script checks for the presence and status of the `heimdalld` service:

1. If the node is managed via `systemd`:
    - It checks if `heimdalld.service` exists and is running.
    - If active, it stops the service using `systemctl stop heimdalld`.
2. If `systemd` is not used but `service` is available:
    - It runs `service heimdalld stop` if the service is running.
3. If neither applies or the service is already stopped:
    - It prints an informational message and proceeds.

## 🧾 Step 4: Obtain Heimdall v1 Genesis File

This step generates (or downloads) a **snapshot of the current Heimdall v1 state** in the form of a `genesis.json` file.

### 📍 What it does

- Uses the `heimdallcli export-heimdall --home=HEIMDALL_HOME` command to export the full Heimdall v1 genesis state.  
- Or download it from a trusted source (github URL)  
- The output is saved to:

    ```bash
    $HEIMDALL_HOME/dump-genesis.json
    
    ```

  (The actual filename is configurable using the `$DUMP_V1_GENESIS_FILE_NAME` variable.)


### 🧯 Rollback

If something goes wrong later in the script, this step registers a rollback action:

```bash
rm -f $GENESIS_FILE

```

This ensures that any partial or temporary `genesis.json` file created by this step will be removed if needed.

### ❗ Important

If this step fails, it likely means one of the following:

- The `heimdallcli` binary is incorrect or broken.
- The `HEIMDALL_HOME` directory is not valid or contains corrupted data.
- The node wasn’t halted cleanly.

## 🔐 Step 5: Generate Checksum of the v1 Genesis File

After exporting the Heimdall v1 genesis in the previous step, this step computes a **SHA-512 checksum** of that file. This is critical for verifying the integrity of the export and ensuring it wasn't corrupted or modified. This checksum can be verified against a string made available by the community.

### 📍 What it does

- Reads the `dump-genesis.json` file (or whatever name is set in `$DUMP_V1_GENESIS_FILE_NAME`).
- Runs:

    ```bash
    sha512sum <genesis file> | awk '{print $1}' > <genesis file>.sha512
    
    ```

- The output is saved in:

    ```bash
    $HEIMDALL_HOME/dump-genesis.json.sha512
    
    ```


### 🧪 Sanity Checks

- Before hashing, the script ensures the genesis file exists.
- After hashing, it checks that the `.sha512` file:
    - Was created successfully.
    - Is **not empty**.
- If any of these fail, the script stops with an error.

### 🧯 Rollback

If later steps fail, the script will delete the generated checksum file automatically using:

```bash
rm -f <checksum file>

```

## ✅ Step 6: Verify Genesis Checksum

This step verifies that the SHA-512 checksum of the exported Heimdall v1 genesis file matches the expected value — ensuring data integrity before proceeding with the migration.

### 📍 What it does

If not running in dry-run mode (`DRY_RUN != true`):

1. Confirms the `.sha512` file exists.
2. Extracts the expected checksum from the file.
3. Compares it against the previously computed `GENERATED_CHECKSUM`.
4. If they don't match, the script exits with an error.
5. If they do, it prints:

```bash
[INFO] Checksum verification passed.

```

## 🗄️ Step 7: Backup Heimdall v1 Data

This step creates a full backup of your existing Heimdall v1 state directory. It's a **critical safety measure** in case anything goes wrong during migration or you need to restore the previous state.

### 📍 What it does

1. Creates the backup directory path (e.g., `/var/lib/heimdall.backup`).
2. Copies **everything** from `$HEIMDALL_HOME` to `$BACKUP_DIR`, preserving:
    - File contents
    - Permissions
    - Ownership
    - Timestamps

The copy is done with:

```bash
cp -a "$HEIMDALL_HOME/." "$BACKUP_DIR"

```

The `-a` flag ensures a complete archival copy.

### 🧯 Rollback

If a later step fails, the script will automatically delete the backup directory using:

```bash
rm -rf "$BACKUP_DIR"

```

This helps clean up temporary or partially copied backups that could clutter the system.

> ⚠️ The rollback only removes the backup if the script fails mid-migration. It won’t touch your original v1 data.
>

### 💡 Tip

You can even manually archive the backup directory later (e.g., `tar czf heimdall-backup.tar.gz`) and store it elsewhere for safekeeping, after the migration is completed.

## 🧹 Step 8: Remove Original Heimdall v1 Directory

After a successful backup in Step 7, this step **removes the original Heimdall v1 data directory** to make room for a clean Heimdall v2 setup.

### ⚠️ Safety Net

Before this step runs:

- The previous step must have **successfully backed up** the original directory.
- If deletion fails, the script exits immediately to avoid inconsistent state.

### 🧯 Rollback

If something goes wrong after this step, the script can restore the deleted directory using:

```bash
mkdir -p "$HEIMDALL_HOME" && cp -a "$BACKUP_DIR/." "$HEIMDALL_HOME"

```

If the restoration fails, it cleans up the partially restored directory with:

```bash
rm -rf "$HEIMDALL_HOME"

```

This ensures your environment is never left in a broken or half-migrated state.

## 📦 Step 9: Select Heimdall v2 Binary Package for Your System

In this step, the script automatically determines the correct Heimdall v2 binary distribution based on your operating system and architecture. It prepares to download both the binary and the configuration profile needed for your node type and network.

### 📍 What it does

1. Creates a temporary working directory:

    ```bash
    /tmp/tmp-heimdall-v2
    
    ```

2. Detects the system OS and architecture using:

    ```bash
    uname -s
    uname -m
    
    ```

3. Based on this, it selects the right combination of:
    - Binary package (either `.deb` or `.rpm`)
    - Configuration profile specific to:
        - Your `NETWORK` (e.g., `mainnet`, `amoy`)
        - Your `NODETYPE` (e.g., `validator`, `sentry`)
        - The specified Heimdall v2 version
4. Sets the following download variables:
    - `url`: the GitHub release URL for the Heimdall v2 binary
    - `package`: the local path in the temp directory where the binary will be downloaded

### 🧯 Rollback

If something fails later, the rollback action for this step will clean up the temporary directory:

```bash
rm -rf /tmp/tmp-heimdall-v2

```

### 🚫 Unsupported Platforms

The script exits early with a clear error message if you're running on:

- macOS (`Darwin`)
- Alpine Linux (`apk`based)
- Any other unrecognized OS/arch combination

These platforms are **not supported** by the current Heimdall v2 binary distribution (and not even v1...).

### ✅ Supported Combinations

| OS | Arch | Package Types |
| --- | --- | --- |
| Linux | x86_64 | `.deb`, `.rpm` |
| Linux | aarch64 | `.deb`, `.rpm` |

The script checks for available tools (`dpkg`, `rpm`) to determine which format to use.

## 🌐 Step 10: Download Heimdall v2 Binary Package

Now that the proper binary and profile packages were selected in Step 9, this step downloads them from the official [Heimdall v2 GitHub release](https://github.com/0xPolygon/heimdall-v2/releases) page.

### 📍 What it does

1. Uses `curl` to download the Heimdall v2 binary to the temp directory:

    ```bash
    curl -L "$url" -o "$package"
    
    ```

2. If a configuration profile (`$profile`) was also determined in the previous step:
    - Downloads the profile using the same method.
    - Saves it in the same temp directory.

### 📦 Files Downloaded

| File Type | Description |
| --- | --- |
| Binary Package | Heimdall v2 binary (`.deb` or `.rpm`) |
| Config Profile | Optional. Network/node-specific config bundle |

Both files are stored temporarily in `/tmp/tmp-heimdall-v2`.

### 🧯 Rollback

If this step or any future step fails, the temp directory and all downloaded files will be removed via:

```bash
rm -rf /tmp/tmp-heimdall-v2

```

## 🛠️ Step 11: Unpack and Install Heimdall v2

This step installs the Heimdall v2 binary on your system, using the correct format based on your platform (`.deb`, `.rpm`, or `tar.gz`). It also installs the associated network/node-specific profile if needed.

### 📍 What it does

The install logic varies depending on the package type:

### 📦 `.tar.gz` (manual unpack)

- Creates a temporary unpack directory.
- Extracts the archive using `tar`.
- If a previous binary exists in `/usr/local/bin/heimdalld`, backs it up to `heimdalld.bak`.
- Copies the new binary into `/usr/local/bin/heimdalld`.

### 📦 `.deb` (Debian/Ubuntu)

- Uninstalls existing `heimdall` or `heimdalld` packages (if any).
- Installs the new `.deb` package using `dpkg -i`.
- If a profile `.deb` exists and no config directory is present, installs the config profile package too.

### 📦 `.rpm` (RHEL/CentOS/Amazon Linux)

- Uninstalls existing `heimdall` packages.
- Installs the new `.rpm` package using `rpm -i --force`.
- Installs the config profile `.rpm` package if available and needed.

### 📦 `apk` (Alpine — unsupported)

While present for completeness, `apk`-based systems are **not supported** and this path should not be reached due to earlier checks.

### 🧯 Rollback

The rollback action varies:

- For manual tar installs, the script:
    - Deletes the unpack directory.
    - Restores the previous binary if backed up.
- For `.deb` and `.rpm` installs, no explicit rollback is defined — the user would need to reinstall the old version manually if needed.

### ✅ Final Output

```bash
[INFO] Heimdall-v2 installation completed.

```

At this point, the Heimdall v2 binary is installed and ready for configuration.

## 🚚 Step 12: Move Heimdall v2 Binary to System Path

This step ensures that the newly installed Heimdall v2 binary is correctly placed in the path defined by `$HEIMDALLD_PATH`, making it the default binary for future system operations.

### 📍 What it does

1. Extracts the directory part of the path in `$HEIMDALLD_PATH` (e.g., `/usr/bin/heimdalld` → `/usr/bin`).
2. Verifies that the target directory exists.
3. If the target binary already exists, it is backed up as:

    ```bash
    <path>.bak
    
    ```

4. Resolves where the new binary was placed:
    - Prefers `/usr/bin/heimdalld` (default install location).
    - Falls back to the package file path if applicable.
5. Copies the new binary to the correct location.
6. Ensures the binary is marked as executable:

    ```bash
    chmod +x $HEIMDALLD_PATH
    
    ```


### 🧯 Rollback

If something fails after this step, the script restores the original binary using:

```bash
mv "${HEIMDALLD_PATH}.bak" "$HEIMDALLD_PATH"

```

Only if the backup exists.

### ✅ Success Output

```bash
[INFO] heimdalld binary copied and set as executable successfully!

```

At this point, the system has the correct `heimdalld` binary in place for the rest of the migration.

## 🧪 Step 13: Verify Heimdall v2 Version

This step validates that the installed `heimdalld` binary matches the expected Heimdall v2 version and that it’s ready to proceed with initialization and migration.

### 📍 What it does

1. Checks if the binary at `$HEIMDALLD_PATH` exists and is executable.
2. Executes:

    ```bash
    heimdalld version
    
    ```

   and captures the version output.

3. Compares the version string to the expected `$HEIMDALL_V2_VERSION`.
4. If the version is correct, the script proceeds. If not, it aborts with an error.
5. It also double-checks that the `$HEIMDALL_HOME` directory was created during the installation or profile unpacking process.

### 🧯 Rollback

If a backup of the previous Heimdall binary exists (`$HEIMDALLD_PATH.bak`), it is restored using:

```bash
sudo mv "${HEIMDALLD_PATH}.bak" "$HEIMDALLD_PATH"

```

Otherwise, the rollback action is a no-op.

### ❗ Troubleshooting

If this step fails:

- Double-check that the installation step (Step 11) completed successfully.
- Make sure the expected version (`$HEIMDALL_V2_VERSION`) is correct.
- Inspect the binary directly: `file $(which heimdalld)` and run it with `-version` manually if needed.

## 🔄 Step 14: Migrate Genesis File to Heimdall v2 Format

This step converts the previously exported Heimdall v1 genesis file into the new format expected by Heimdall v2 using the built-in `heimdalld migrate` command.

### 📍 What it does

1. Defines a target output file:

    ```bash
    $BACKUP_DIR/migrated_dump-genesis.json
    
    ```

2. Verifies that the original v1 genesis file exists in the backup directory.
3. Checks if the configured `GENESIS_TIME` is in the future:
    - If so, prints a warning.
    - Gives the user a chance to pause (e.g. to edit the time manually).
4. Executes the actual migration:

    ```bash
    heimdalld migrate <v1_genesis_file> --chain-id=<id> --genesis-time=<time> --initial-height=<height> --verify-hash=<bool>
    
    ```

5. Verifies the output file was created.
6. Reads the `initial_height` from the migrated genesis and checks it matches the expected value.

### 🧯 Rollback

If the migration was successful and the file exists, the rollback action deletes the migrated genesis:

```bash
rm -f "$MIGRATED_GENESIS_FILE"

```

### ✅ Success Output

```bash
[INFO] Genesis file migrated successfully from v1 to v2
[INFO] initial_height in genesis matches expected value: <value>

```

This confirms the migration was successful and the genesis file is ready for Heimdall v2 initialization.

## 🔐 Step 15: Generate Checksum of the v2 Genesis File

After migrating the Heimdall v1 to the v2 genesis file in the previous step, this step computes a **SHA-512 checksum** of that file. This is critical for verifying the integrity of the migrated genesis and ensuring it wasn't corrupted or modified. This checksum can be verified against a string made available by the community.

### 📍 What it does

- Reads the `$MIGRATED_GENESIS_FILE`.
- Runs:

    ```bash
    sha512sum < migrated genesis file> | awk '{print $1}' > <migrated genesis file>.sha512
    
    ```

- The output is saved in:

    ```bash
    MIGRATED_GENESIS_FILE="$BACKUP_DIR/migrated_$DUMP_V1_GENESIS_FILE_NAME.sha512"
    
    ```


### 🧪 Sanity Checks

- Before hashing, the script ensures the genesis file exists.
- After hashing, it checks that the `.sha512` file:
    - Was created successfully.
    - Is **not empty**.
- If any of these fail, the script stops with an error.

### 🧯 Rollback

If later steps fail, the script will delete the generated checksum file automatically using:

```bash
rm -f <migrated checksum file>

```

## ✅ Step 16: Verify Genesis Checksum

This step verifies that the SHA-512 checksum of the exported Heimdall v1 genesis file matches the expected value — ensuring data integrity before proceeding with the migration.

### 📍 What it does

If not running in dry-run mode (`DRY_RUN != true`):

1. Confirms the `.sha512` file exists.
2. Extracts the expected checksum from the file.
3. Compares it against the previously computed `GENERATED_MIGRATED_CHECKSUM`.
4. If they don't match, the script exits with an error.
5. If they do, it prints:

```bash
[INFO] Checksum verification passed.

```

## 🧰 Step 17: Create Temporary Heimdall v2 Home Directory

This step prepares a temporary home directory for initializing Heimdall v2 using the migrated genesis file, before promoting it to the final path.

### 🧯 Rollback

If something fails later, the rollback will clean up this directory:

```bash
rm -rf /tmp/tmp-heimdall-v2-home

```

## 🧱 Step 18: Initialize Heimdall v2

This step performs the actual initialization of Heimdall v2 using the `heimdalld init` command. It sets up the configuration and directory structure in the proper `$HEIMDALL_HOME` path, using the previously created temporary environment.

### 📍 What it does

1. Verifies that `$HEIMDALL_HOME` exists.
2. If it does, backs it up by renaming the directory to:

    ```bash
    $HEIMDALL_HOME.bak
    
    ```

3. Registers a rollback step to restore the backup in case of failure.
4. Runs the `heimdalld init` command using:

    ```bash
    heimdalld init <MONIKER_NODE_NAME> --home=/tmp/tmp-heimdall-v2-home
    
    ```

   This generates:

    - A fresh `config` directory
    - `data` folder scaffolding
    - Node identity and validator keys (if needed)
5. Clears the content of the now-empty `$HEIMDALL_HOME` directory.
6. Copies the fully-initialized contents from the temp v2 home into `$HEIMDALL_HOME`.
7. Deletes the temporary working directory.

### 🧯 Rollback

If this step fails, the rollback action will restore your original Heimdall home:

```bash
mv "$HEIMDALL_HOME.bak" "$HEIMDALL_HOME"

```

### ✅ Success Output

```bash
[INFO] heimdalld initialized successfully.

```

This means Heimdall v2 is now fully initialized and ready to receive the migrated genesis and configuration.

### 💡 Tip

The initialized directory does **not yet contain** the migrated genesis file — that will be added in a later step.

## 📂 Step 19: Verify Required Directories and Configuration Files

This step ensures that the newly initialized Heimdall v2 directory contains all critical subdirectories and configuration files before continuing the migration.

### 📍 What it does

1. Checks for the presence of the required subdirectories under `$HEIMDALL_HOME`:
    - `data`
    - `config`
2. Ensures that all the following config files exist inside `config/`:
    - `app.toml`
    - `client.toml`
    - `config.toml`
    - `genesis.json`
    - `node_key.json`
    - `priv_validator_key.json`
3. Verifies that the following data file exists in `data/`:
    - `priv_validator_state.json`

If **any file or directory is missing**, the script exits with an error.

### 🧯 Rollback

If a failure occurs, the rollback action restores the previous Heimdall home from backup:

```bash
mv "$HEIMDALL_HOME.bak" "$HEIMDALL_HOME"

```

If no backup is present, rollback is skipped.

## 🌉 Step 20: Restore `bridge/` Directory from Backup (Validators Only)

This step restores the `bridge` directory from your Heimdall v1 backup. This directory is critical for **validators**.

### 📍 What it does

1. Defines the expected bridge paths:
    - Source: `$BACKUP_DIR/bridge`
    - Destination: `$HEIMDALL_HOME/bridge`
2. If the `bridge` directory exists in the backup:
    - It is copied back into the new Heimdall v2 home.
3. If not present (e.g. for `sentry` nodes), a message is printed and the step is skipped.

### 🔍 When This Matters

- **Validator nodes**: Need the `bridge/` directory to continue signing and relaying events.
- **Sentry nodes**: Don’t use `bridge/`, so this step will be skipped silently.

## 📜 Step 21: Move Migrated Genesis File to Heimdall v2 Home

This step replaces the default genesis file generated during `heimdalld init` with the **migrated v1-to-v2 genesis file**, placing it in the correct location inside your Heimdall v2 setup.

### 📍 What it does

1. Defines the target path:

    ```bash
    $HEIMDALL_HOME/config/genesis.json
    
    ```

2. If that file already exists:
    - It is backed up as `genesis.json.bak`.
    - A rollback action is registered to restore the original if needed.
3. The migrated genesis file is then copied over using:

    ```bash
    cp -p "$MIGRATED_GENESIS_FILE" "$TARGET_GENESIS_FILE"
    
    ```

   The `-p` flag preserves file metadata (timestamps, permissions).


### 🧯 Rollback

Rollback logic depends on whether the file previously existed:

- If it did, restore from `.bak`.
- If it didn’t, remove the newly copied genesis file.

### 💡 Tip

This is a critical point in the migration — from now on, Heimdall v2 will start with **your real, production state** from v1. Double-check the contents of the file if you're running into unexpected chain behavior.

## 🔐 Step 22: Update `priv_validator_key.json` for Heimdall v2

This step updates the validator's private key configuration to ensure it matches the values from your original Heimdall v1 setup. This is **critical for validators** to sign blocks and participate in consensus under v2.

### 📍 What it does

1. Locates the `priv_validator_key.json` file in the new v2 config path:

    ```bash
    $HEIMDALL_HOME/config/priv_validator_key.json
    
    ```

2. If the file exists, it is backed up:

    ```bash
    cp priv_validator_key.json priv_validator_key.json.bak
    
    ```

3. Extracts the original `address`, `pub_key.value`, and `priv_key.value` from the backup:

    ```bash
    $BACKUP_DIR/config/priv_validator_key.json
    
    ```

4. Overwrites the target file using `jq` to inject the correct values.
5. Validates that the updated file is not empty or corrupted.

### 🧯 Rollback

If the step fails later, the script restores the backed-up key file from:

```bash
$HEIMDALL_HOME/config/priv_validator_key.json.bak

```

### ✅ Success Output

```bash
[INFO] Updated priv_validator_key.json file saved as /path/to/priv_validator_key.json

```

This confirms that the v2 node will use the **same key identity** as the v1 node.

### 💡 Tip

- This file is critical for **validators only**. Sentries don’t use this for signing, but the file must still exist.
- Ensure that no manual edits or formatting errors break the JSON structure.

## 🔑 Step 23: Update `node_key.json` for Heimdall v2

This step ensures the node identity key is correctly carried over from Heimdall v1. The `node_key.json` defines how your node is identified in the network — this is not for signing blocks, but is essential for peer-to-peer communication.

### 📍 What it does

1. Targets the file:

    ```bash
    $HEIMDALL_HOME/config/node_key.json
    
    ```

2. Backs up the existing v2 version (if present) as `node_key.json.bak`.
3. Extracts the original node's `priv_key.value` from the backup config:

    ```bash
    $BACKUP_DIR/config/node_key.json
    
    ```

4. Replaces the corresponding value in the v2 file using `jq`.
5. Writes to a temp file and performs validation (ensures it's non-empty).
6. Moves the updated file into place.

### 🧯 Rollback

If this step fails, the backup file is restored:

```bash
mv "$NODE_KEY_FILE.bak" "$NODE_KEY_FILE"

```

### ✅ Success Output

```bash
[INFO] Updated node_key.json file saved as /path/to/node_key.json

```

This confirms the node will keep its identity across the upgrade, preserving its P2P reputation and connection consistency.

### 💡 Tip

- All nodes (validators and sentries) use `node_key.json`.
- Do not reuse the same key across multiple nodes unless you're setting up a clone intentionally.

## 🧾 Step 24: Fix `priv_validator_state.json` JSON Format

This step corrects a known formatting issue in the v1 `priv_validator_state.json` file where the `round` field may be stored as a string. Heimdall v2 expects this field to be an integer.

### 📍 What it does

1. Targets the file:

    ```bash
    $HEIMDALL_HOME/data/priv_validator_state.json
    
    ```

2. Verifies that the file exists.
3. Creates a backup:

    ```bash
    cp priv_validator_state.json priv_validator_state.json.bak
    
    ```

4. Ensures the JSON is valid by running:

    ```bash
    jq empty
    
    ```

5. Converts the `round` field from string to integer using:

    ```bash
    jq '.round |= tonumber'
    
    ```

6. Validates that the result is non-empty and then moves the corrected file into place.

### 🧯 Rollback

If something goes wrong, the script restores the backup:

```bash
mv priv_validator_state.json.bak priv_validator_state.json

```

## 📘 Step 25: Restore `addrbook.json` from Backup (if present)

This step restores the peer address book (`addrbook.json`) from your Heimdall v1 backup. It helps the node reconnect to previously known peers more quickly after the migration.

### 📍 What it does

1. Checks for the presence of:

    ```bash
    $BACKUP_DIR/config/addrbook.json
    
    ```

2. If the file exists:
    - Backs up any current `addrbook.json` in the v2 config directory to `addrbook.json.bak`.
    - Copies the file from the backup to:

        ```bash
        $HEIMDALL_HOME/config/addrbook.json
        
        ```

3. If the file does **not** exist in the backup:
    - Logs that it is being skipped.

### 🧯 Rollback

If a backup was created:

- Rollback will restore the `.bak` file.

Otherwise:

- It will remove the restored `addrbook.json`.

If nothing was restored, the rollback is a no-op.

### 💡 Tip

While not critical for chain operation, restoring this file speeds up the process of rejoining the peer-to-peer network — especially useful for validators and sentries with many peer relationships.

## ⚙️ Step 26: Configuration Migration – Minimal Auto-Port from v1 to v2

This step automatically migrates a minimal and safe subset of configuration values from Heimdall v1 to Heimdall v2 using `sed`. It reduces the burden of manual config editing while preserving critical node-specific values.

### 📍 What it does

1. Migrates selected values from your backed-up v1 config files:
   - From `$BACKUP_DIR/config/config.toml` → `$HEIMDALL_HOME/config/config.toml`:
      - `moniker`
      - `external_address`
      - `seeds`
      - `persistent_peers`
      - `max_num_inbound_peers`
      - `max_num_outbound_peers`
   - From `$BACKUP_DIR/config/heimdall-config.toml` → `$HEIMDALL_HOME/config/app.toml`:
      - `eth_rpc_url`
      - `bor_rpc_url`
      - `bor_grpc_flag`
   - Into `$HEIMDALL_HOME/config/client.toml`:
      - `chain-id` (set to `$CHAIN_ID`)
2. Prints all copied values.
3. Validates that each key was correctly written and matches the original value.
4. Pauses and informs the user about additional manual configuration that may be required (e.g., ports, metrics).
5. Warns the user to manually update `bor/config.toml` (TODO: to be defined in future).

### ✅ Success Output

```bash
[INFO] chain-id successfully set to "polygon"
[OK]   config.toml: moniker = my-node
[OK]   config.toml: seeds = ...
[OK]   app.toml: eth_rpc_url = https://...
...
[INFO] config.toml values migrated successfully.
[INFO] app.toml values migrated successfully.
```

This confirms the required minimal configuration has been auto-migrated successfully, and the script will continue.

### 💡 Tip

For non-critical or environment-specific settings (e.g. port overrides, logging, tracing, instrumentation), you can safely edit the configuration files manually after the migration.

### 📝 Bor Config Reminder

At the end of this step, the script will remind you to update your Bor's `config.toml` manually.

> ⚠️ **TODO**: This guide and script will be updated once the final Bor settings required for Heimdall v2 integration are confirmed.

## 👤 Step 27: Assign Correct Ownership and Permissions

This step ensures that all files and directories under your Heimdall v2 home directory are securely owned by the correct system user and have proper access permissions.

### 📍 What it does

1. Prevents accidental changes to critical system directories (e.g. `/`, `/usr/bin`, etc.).
2. Recursively assigns ownership to the `$HEIMDALL_SERVICE_USER`:

    ```bash
    chown -R $HEIMDALL_SERVICE_USER:$HEIMDALL_SERVICE_USER $HEIMDALL_HOME
    
    ```

3. Applies strict permissions:
    - All **files**: `600` (read/write only for owner)
    - All **directories**: `700` (full access only for owner)

### 🔒 Why This Is Important

- Avoids permission errors when running `heimdalld` as a non-root system service.
- Protects sensitive files like keys and state from unauthorized access.

### ⚠️ Safety Check

The script includes a **hard-coded list** of critical system paths and refuses to run `chown` if `$HEIMDALL_HOME` accidentally points to something dangerous:

```bash
("/", "/usr", "/usr/bin", "/bin", "/lib", "/lib64", "/etc", "/boot")

```

This protects against misconfigurations that could damage your system.

## 🛠️ Step 28: Patch `systemd` Unit File to Set Correct Service User

This step ensures that the `heimdalld` systemd service runs under the correct user account (`$HEIMDALL_SERVICE_USER`). This is critical for permission consistency and safe service management. If not consistent, heimdall v2 could throw permission errors when accessing files under `HEIMDALL_HOME` (e.g. config files or bridge db)

### 📍 What it does

1. Attempts to detect the active systemd unit file for `heimdalld` using:

    ```bash
    systemctl status heimdalld | grep 'Loaded:'
    
    ```

2. If the service file is found:
    - Backs it up to `${SERVICE_FILE}.bak`.
    - Updates the `User=` line within the `[Service]` block using `sed`:

        ```bash
        sed -i "/^\\[Service\\]/,/^\\[/{s/^\\(\\s*User=\\).*/\\1$HEIMDALL_SERVICE_USER/}" "$SERVICE_FILE"
        
        ```

    - Reloads the systemd manager with:

        ```bash
        sudo systemctl daemon-reload
        
        ```

3. If the file cannot be found:
    - Prints a warning and asks you to manually edit the service file.
    - Waits for user confirmation before continuing.

### 🧯 Rollback

No rollback action is defined for this step. The backup of the systemd file can be restored manually from:

```bash
$SERVICE_FILE.bak

```

### ✅ Success Output

```bash
[INFO] Systemd unit patched.

```

If the file wasn’t found:

```bash
[WARNING] Could not detect systemd unit file for heimdalld. Please update it manually.

```

### 💡 Tip

To manually update your systemd unit file:

1. Locate: heimdalld.service (e.g. `/etc/systemd/system/heimdalld.service`)
2. Add or modify:

    ```
    [Service]
    User=<YOUR_USER>
    
    ```

3. Then run:

    ```bash
    sudo systemctl daemon-reload
    
    ```


This ensures your node starts with the correct user permissions.

## 🧼 Step 29: Clean Up `.bak` Files in Heimdall Home

This step removes temporary backup (`.bak`) files created during earlier migration steps. These files were used for safety and rollback, but are no longer needed if everything completed successfully.

### 📍 What it does

1. Searches for all files ending in `.bak` under:

    ```bash
    $HEIMDALL_HOME/config/
    $HEIMDALL_HOME/data/
    
    ```

   using:

    ```bash
    find "$HEIMDALL_HOME" -type f -name "*.bak"
    
    ```

2. If any `.bak` files are found:
    - Lists them to the console
    - Deletes them using `rm -f`
3. If no files are found:
    - Prints an informational message and moves on

## 🚀 Step 30: Start Heimdall v2 Service

This is the final step of the migration — it starts the Heimdall v2 systemd service and performs safety checks to confirm that the node started successfully under the expected system user.

### 📍 What it does

1. Reloads the systemd daemon:

    ```bash
    sudo systemctl daemon-reload
    
    ```

2. Starts the `heimdalld` service:

    ```bash
    sudo systemctl start heimdalld
    
    ```

3. If Heimdall was previously backed up:
    - Registers a rollback action to stop the service and restore the backup.
4. Verifies the process started correctly:
    - Waits up to 10 seconds for `heimdalld` to appear via `pgrep`.
    - Retrieves the actual user running the process.
    - Warns if the user is different from `$HEIMDALL_SERVICE_USER`.
5. Optionally restarts `telemetry.service` if it's found on the system.

### 🧯 Rollback

If anything fails after the service is started:

```bash
sudo systemctl stop heimdalld
mv $HEIMDALL_HOME.bak $HEIMDALL_HOME  # if backup exists

```

### ✅ Success Output

```bash
[INFO] Heimdall successfully started as user: <username>
[INFO] All services started successfully.

✅ [SUCCESS] Heimdall v2 migration completed successfully! ✅
🕓 Migration completed in X seconds.

```

It also prompts the user with:

```bash
journalctl -fu heimdalld

```

to monitor logs and confirm that everything is running smoothly.

### 💡 Tip

If the user mismatch warning appears, inspect the unit file with:

```bash
sudo systemctl cat heimdalld

```

Correct the `User=` setting if needed and reload systemd.
