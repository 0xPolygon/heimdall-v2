# Heimdall v1 -> v2 README

## ⚠️ Important notice on the migration process
The script will be executed for the very first time on a node managed by the Polygon team (pilot node).  
Once the migration on that node is successful:
- The v1 genesis will be exported and made available for the community on heimdall-v2 repo together with a checksum
- The v2 genesis will be created and made available for the community on heimdall-v2 repo together with a checksum
- The script will be distributed with the checksum to prevent any tampering and made available for the community on heimdall-v2 repo
- Node operators can perform the migration on their own nodes using the script (or a modified version of it if the architecture is not supported)
For more info about the process, check [COMMANDS.md](./COMMANDS.md) and [script](migrate.sh).

## Migration script

- Validate dependencies and prerequisites
  - the system needs to have the following tools installed:
    - `curl`
    - `tar`
    - `jq`
    - `sha512sum`
    - `file`
    - `awk`
    - `sed`
    - `systemctl`
    - `grep`
    - `id`
- Halt your current Heimdall v1
- Make sure the latest committed height is reached
  - If yes, it exports the genesis from v1
  - If no, it downloads the v1 genesis from a trusted source
- Generate the genesis checksum and validates it
- Back up Heimdall v1
- Install Heimdall v2 and verifies binaries
- Migrate the genesis format to v2
- Generate the migrated genesis checksum and validates it
- Initialize Heimdall v2
- Restore and update keys, configuration, and validator state
- Assign ownership permissions
- Update the systemd unit file
---

## 🛡️ Built-In Safety

- ✅ **Script integrity check**: Prevents partial executions
- 🛡️ **Script checksum**: The script will be delivered with checksum to prevent any tampering
- 🔐 **`sudo` enforcement**: Ensures system permissions
- 🧱 **Checksum validation**: Protects against tampered genesis
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

---

## ⚙️ Requirements

- Ubuntu 20.04+ or similar Linux distro
- `heimdalld` and`heimdallcli` in PATH
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
| `file`      | File type detection   | `sudo apt install file`         |
| `awk`       | Text processing       | `sudo apt install gawk`         |
| `sed`       | Stream editing        | `sudo apt install sed`          |
| `systemctl` | Service management    | Pre-installed on most distros   |
| `grep`      | Text searching        | Pre-installed on most distros   |
| `id`        | User information      | Pre-installed on most distros   |

Also, make sure the node's disk has enough space to store the backup of Heimdall v1 and the new genesis file.  
Furthermore, the user must ensure that heimdall v1 config files are correct and properly formatted.  

---

## 💬 Example Usage

```bash
sudo bash migrate.sh \
  --heimdall-v1-home=/var/lib/heimdall \
  --heimdallcli-path=/usr/bin/heimdallcli \
  --d-path=/usr/bin/heimdalld \
  --network=mainnet \
  --node-type=validator \
  --backup-dir=/var/lib/heimdall.backup \
  --moniker=my-node \
  --service-user=heimdall \
  --generate-genesis=true \
```

For a possible output, see [output.log](./output-example.txt)

### 🧩 Required Arguments

| Flag                 | Description                                                                                                    |
|----------------------|----------------------------------------------------------------------------------------------------------------|
| `--heimdall-v1-home` | Path to Heimdall v1 home (must contain `config` and `data`)                                                    |
| `--heimdallcli-path` | Path to `heimdallcli` (must be latest stable version). It can be retrieved with `which heimdallcli`            |
| `--d-path`           | Path to `heimdalld` (must be latest stable version). It can be retrieved with `which heimdalld`                |
| `--network`          | `mainnet` or `amoy`                                                                                            |
| `--node-type`        | `sentry` or `validator`                                                                                        |
| `--service-user`     | System user running Heimdall (e.g., `heimdall`).                                                               |
|                      | Check with: `systemctl status heimdalld` and inspect the `User=` field.                                        |
|                      | Confirm it's correct by checking the user currently running the process (e.g., with `ps -o user= -C heimdalld` |
|                      | This is critical to avoid permission issues in v2!                                                             |
| `--generate-genesis` | Whether to generate genesis using `heimdalld` (recommended: `true`).                                           |
|                      | Note that this value will be anyway overwritten by the script.                                                 |
|                      | This happens when the node was not able to commit to the latest block's heigh needed for the migration,        |
|                      | hence generate-genesis will be set to false and the genesis.json file downloaded from trusted source.          |

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
By default, heimdall polls bor by using frequent HTTP requests, which can be inefficient. Enabling WebSocket support reduces overhead and improves sync responsiveness.  
Edit your bor `config.toml` file and add the following under the [heimdall] section:

```toml
[heimdall]
ws-address = "ws://localhost:26657/websocket"
```
This assumes Heimdall is running with its WebSocket endpoint enabled on port 26657. Adjust the port or host if your setup differs.  

After updating, restart your Bor node to apply the new configuration.
