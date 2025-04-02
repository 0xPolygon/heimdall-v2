# Heimdall v1 -> v2 migration script

The migration script

- Halts and backs up your current Heimdall v1
- Migrates the genesis file and configuration
- Validates binaries and checksum integrity
- Sets up Heimdall v2 with proper permissions
- Starts the new service safely

---

## 🛡️ Built-In Safety

- ✅ **Script integrity check**: Prevents partial executions
- 🛡️ **Script checksum and signature**: The script will be delivered with checksum and signature, to prevent any tampering  
- 🔐 **`sudo` enforcement**: Ensures system permissions
- 🧱 **Checksum validation**: Protects against tampered genesis
- 🧠 **Rollback logic**: Allows partial undo of dangerous steps
- 👤 **Systemd user detection**: Avoids ownership mismatches
- 🪵 **Logs & prompts**: Transparent and user-guided

---

## 🏗 Structure

| Step | Description                                                      |
| --- |------------------------------------------------------------------|
| 1 | Dependency validation and prerequisites checks                   |
| 2 | Prompt for paths, versions, and user inputs                      |
| 3-4 | Stops Heimdall v1 and export (or download) its genesis           |
| 5-6 | Generate and verify exported genesis checksum                    |
| 7-8 | Backup and remove Heimdall v1                                    |
| 9-13 | Install Heimdall v2 and verify binaries                          |
| 14 | Migrate genesis format to v2                                     |
| 15-16 | Generate and verify migrated genesis checksum                    |
| 17-19 | Initialize heimdall v2, and make sure the required configs exist |
| 20-25 | Restore keys, configuration and validator state                  |
| 26 | Configs update                                                   |
| 27 | Assign ownership permissions                                     |
| 28 | Update systemd unit file                                         |
| 29 | Clean backups                                                    |
| 30 | Start heimdall-v2                                                |

---

## ⚙️ Requirements

- Ubuntu 20.04+ or similar Linux distro
- `heimdalld` and`heimdallcli` in PATH (control over `bor` even if on a different machine)
- Migration prerequisites (`halt_height`, correct config backups…)
- Network: `mainnet` or `amoy`
- Supported nodes: `sentry` and `validator`

Before running the migration script, make sure the following tools are installed on your system:

| Tool         | Purpose                           | Install Command (Ubuntu/Debian)        |
|--------------|-----------------------------------|----------------------------------------|
| `curl`       | Downloading binaries              | `sudo apt install curl`                |
| `tar`        | Extracting archives               | `sudo apt install tar`                 |
| `jq`         | JSON manipulation                 | `sudo apt install jq`                  |
| `sha512sum`  | File integrity checks             | `sudo apt install coreutils`           |

---

## 💬 Example Usage

```bash
sudo ./migrate.sh \
  --heimdall-home=/var/lib/heimdall \
  --cli-path=/usr/bin/heimdallcli \
  --d-path=/usr/bin/heimdalld \
  --network=mainnet \
  --nodetype=validator \
  --backup-dir=/var/lib/heimdall.backup \
  --moniker=my-node \
  --service-user=heimdall
```

For a possible output, see [output.log](./output.log)

### 🧩 Required Arguments

| Flag                 | Description                                                                  |
|----------------------|------------------------------------------------------------------------------|
| `--heimdall-home`    | Path to Heimdall v1 home (must contain `config` and `data`)                  |
| `--cli-path`         | Path to `heimdallcli` (must be >= v1.0.10)                                   |
| `--d-path`           | Path to `heimdalld` (must be `1.2.0-41-*`)                                   |
| `--network`          | `mainnet` or `amoy`                                                          |
| `--nodetype`         | `sentry` or `validator`                                                      |
| `--backup-dir`       | Directory where a backup of Heimdall v1 will be stored                       |
| `--moniker`          | Node moniker (must match the value in `config.toml`)                         |
| `--service-user`     | System user running Heimdall (e.g., `heimdall`).                             |
|                      | 👉 Check with: `systemctl status heimdalld` and inspect the `User=` field.   |
|                      | Confirm it's correct with by checking the user currently running the process |
| `--generate-genesis` | Whether to generate genesis using `heimdalld` (recommended: `true`)            |

### ⚙️ Optional Arguments

| Flag                   | Description                                                                 |
|------------------------|-----------------------------------------------------------------------------|
| `--bor-path`           | Path to `bor` binary (only needed if Bor runs on this machine)              |


## ✅ Supported Platforms
| OS | Arch | Package Manager | Supported | Notes |
| --- | --- | --- | --- | --- |
| Linux | x86_64 | `dpkg` (Debian) | ✅ | Uses `.deb` package |
| Linux | x86_64 | `rpm` (RedHat) | ✅ | Uses `.rpm` package |
| Linux | aarch64 | `dpkg` | ✅ | Uses ARM `.deb` package |
| Linux | aarch64 | `rpm` | ✅ | Uses ARM `.rpm` package |
| macOS | Any | — | ❌ | Not supported |
| Alpine | Any | `apk` | ❌ | Not supported |

The script determines the correct Heimdall v2 package to install based on your system architecture and package manager.
