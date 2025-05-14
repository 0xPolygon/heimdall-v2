# Heimdall v1 -> v2 README

## ⚠️ Important notice on the migration process
The script will be executed the very first time on a node managed by Polygon team.  
Once the migration on that node is successful:
- The v1 genesis will be exported and made available for the community on heimdall-v2 repo together with a checksum
- The v2 genesis will be created and made available for the community on heimdall-v2 repo together with a checksum
- The script will be distributed with the checksum, to prevent any tampering, and made available for the community on heimdall-v2 repo
- Node operators can perform the migration on their own nodes using the script (or a modified version of it if the architecture is not supported)

## Migration script

- Validates dependencies and prerequisites
- Halts your current Heimdall v1
- Makes sure the latest committed height is reached
  - If yes, it exports the genesis from v1
  - If no, it downloads the v1 genesis from a trusted source
- Generates the genesis checksum and validates it
- Backs up Heimdall v1
- Installs Heimdall v2 and verifies binaries
- Migrates the genesis format to v2
- Generates the migrated genesis checksum and validates it
- Initializes Heimdall v2
- Restores and updates keys, configuration, and validator state
- Assigns ownership permissions
- Updates the systemd unit file
- Cleans up backups (heimdall v1 data will be kept in the backup directory and can manually deleted later on)
---

## 🛡️ Built-In Safety

- ✅ **Script integrity check**: Prevents partial executions
- 🛡️ **Script checksum**: The script will be delivered with checksum , to prevent any tampering
- 🔐 **`sudo` enforcement**: Ensures system permissions
- 🧱 **Checksum validation**: Protects against tampered genesis
- 🧠 **Rollback logic**: Allows partial undo of dangerous steps
- 👤 **Systemd user detection**: Avoids ownership mismatches
- 🪵 **Logs & prompts**: Transparent and user-guided

---

## 🏗 Structure

| Step  | Description                                                                               |
|-------|-------------------------------------------------------------------------------------------|
| 1     | Dependency validation and prerequisites checks                                            |
| 2     | Prompt for paths, versions, and user inputs                                               |
| 3-5   | Stops Heimdall v1, check for latest committed height and export (or download) its genesis |
| 6-7   | Generate and verify exported genesis checksum                                             |
| 8     | Backup Heimdall v1                                                                        |
| 9-13  | Install Heimdall v2 and verify binaries                                                   |
| 14    | Migrate genesis format to v2                                                              |
| 15-16 | Generate and verify migrated genesis checksum                                             |
| 17-19 | Initialize heimdall v2, and make sure the required configs exist                          |
| 20-25 | Restore keys, configuration and validator state                                           |
| 26    | Configs update                                                                            |
| 27    | Assign ownership permissions                                                              |
| 28    | Update systemd unit file                                                                  |
| 29    | Clean backups                                                                             |

---

## ⚙️ Requirements

- Ubuntu 20.04+ or similar Linux distro
- `heimdalld` and`heimdallcli` in PATH (optional control over `bor` if on the same machine)
- Migration prerequisites (`halt_height`, correct config backups…)
- Network: `devnet` (for testing), `amoy` or `mainnet`
- Supported nodes: `sentry` and `validator`

Before running the migration script, make sure the following tools are installed on your system.  
The migration script will anyway fail early if such tools are not installed.  

| Tool        | Purpose               | Install Command (Ubuntu/Debian) |
|-------------|-----------------------|---------------------------------|
| `curl`      | Downloading binaries  | `sudo apt install curl`         |
| `tar`       | Extracting archives   | `sudo apt install tar`          |
| `jq`        | JSON manipulation     | `sudo apt install jq`           |
| `sha512sum` | File integrity checks | `sudo apt install coreutils`    |

---

## 💬 Example Usage

```bash
sudo bash migrate.sh \
  --heimdall-home=/var/lib/heimdall \
  --cli-path=/usr/bin/heimdallcli \
  --d-path=/usr/bin/heimdalld \
  --network=mainnet \
  --nodetype=validator \
  --backup-dir=/var/lib/heimdall.backup \
  --moniker=my-node \
  --service-user=heimdall \
  --generate-genesis=true \
  --bor-path=/usr/bin/bor
```

For a possible output, see [output.log](./output.log)

### 🧩 Required Arguments

| Flag                 | Description                                                                                             |
|----------------------|---------------------------------------------------------------------------------------------------------|
| `--heimdall-home`    | Path to Heimdall v1 home (must contain `config` and `data`)                                             |
| `--cli-path`         | Path to `heimdallcli` (must be latest stable version)                                                   |
| `--d-path`           | Path to `heimdalld` (must be latest stable version)                                                     |
| `--network`          | `mainnet` or `amoy`                                                                                     |
| `--nodetype`         | `sentry` or `validator`                                                                                 |
| `--backup-dir`       | Directory where a backup of Heimdall v1 will be stored                                                  |
| `--moniker`          | Node moniker (must match the value in v1 `config.toml`)                                                 |
| `--service-user`     | System user running Heimdall (e.g., `heimdall`).                                                        |
|                      | 👉 Check with: `systemctl status heimdalld` and inspect the `User=` field.                              |
|                      | Confirm it's correct by checking the user currently running the process.                                |
|                      | This is critical to avoid permission issues in v2!                                                      |
| `--generate-genesis` | Whether to generate genesis using `heimdalld` (recommended: `true`).                                    |
|                      | Note that this value will be anyway overwritten by the script.                                          |
|                      | This happens when the node was not able to commit to the latest block's heigh needed for the migration, |
|                      | hence generate-genesis will be set to false and the genesis.json file downloaded from trusted source.   |


### ⚙️ Optional Arguments

| Flag         | Description                                                                    |
|--------------|--------------------------------------------------------------------------------|
| `--bor-path` | Path to `bor` binary (only needed if Bor runs on the same machine as heimdall) |

## ✅ Supported Platforms

| OS     | Arch    | Package Manager | Supported | Notes                   |
|--------|---------|-----------------|-----------|-------------------------|
| Linux  | x86_64  | `dpkg` (Debian) | ✅         | Uses `.deb` package     |
| Linux  | x86_64  | `rpm` (RedHat)  | ✅         | Uses `.rpm` package     |
| Linux  | aarch64 | `dpkg`          | ✅         | Uses ARM `.deb` package |
| Linux  | aarch64 | `rpm`           | ✅         | Uses ARM `.rpm` package |
| macOS  | Any     | —               | ❌         | Not supported           |
| Alpine | Any     | `apk`           | ❌         | Not supported           |

The script determines the correct Heimdall v2 package to install based on your system architecture and package manager.
If your machine doesn't match any supported platform (or if you are using docker), you would need to modify the script accordingly.  

### Optional: use WebSocket for Bor–Heimdall communication
After the migration, to optimize communication between Heimdall and Bor, you can optionally enable WebSocket support in your bor `config.toml` file.  
By default, Heimdall polls Bor using frequent HTTP requests, which can be inefficient. Enabling WebSocket support reduces overhead and improves sync responsiveness.  
Edit your bor `config.toml` file and add the following under the [heimdall] section:

```toml
[heimdall]
ws-address = "ws://localhost:26657/websocket"
```
This assumes Heimdall is running with its WebSocket endpoint enabled on port 26657. Adjust the port or host if your setup differs.  

After updating, restart your Bor node to apply the new configuration.
