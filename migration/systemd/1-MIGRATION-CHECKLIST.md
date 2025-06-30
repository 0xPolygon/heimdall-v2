# Checklist for Heimdall v1 → v2 Migration via Script (Systemd)

This checklist applies to users migrating Heimdall from v1 to v2
using the automated `migrate.sh` script under a `systemd`-based deployment.

---

## 1. Platform Compatibility

Ensure your platform is supported by the migration script:

| OS     | Arch    | Package Manager | Supported | Notes                 |
|--------|---------|-----------------|-----------|-----------------------|
| Linux  | x86_64  | `dpkg` (Debian) | ✅         | Uses `.deb` package   |
| Linux  | aarch64 | `dpkg`          | ✅         | Uses ARM `.deb`       |

---

## 2. Required Tools

The following tools must be installed. The script will fail early if any are missing.

| Tool        | Purpose               | Install Command (Ubuntu/Debian) |
|-------------|-----------------------|---------------------------------|
| `curl`      | Download binaries     | `sudo apt install curl`         |
| `tar`       | Extract archives      | `sudo apt install tar`          |
| `jq`        | JSON manipulation     | `sudo apt install jq`           |
| `sha512sum` | Integrity checks      | `sudo apt install coreutils`    |
| `file`      | File type detection   | `sudo apt install file`         |
| `awk`       | Text processing       | `sudo apt install gawk`         |
| `sed`       | Stream editing        | `sudo apt install sed`          |
| `systemctl` | Service management    | Pre-installed on most systems   |
| `grep`      | Pattern matching      | Pre-installed on most systems   |
| `id`        | User info lookup      | Pre-installed on most systems   |

---

## 3. Shell Requirements

Ensure `bash` is installed and used.  
The script relies on `bash` features and will not work with `sh`.

---

## 4. Validate Heimdall v1 Config Files

Verify that the files in `HEIMDALL_HOME/config` are present and correctly formatted.

---

## 5. Memory Requirements

Ensure the system has **at least 20 GB of available RAM**.

---

## 6. Disk Space Requirements

Ensure the system has **at least 3× the size of `HEIMDALL_HOME`** in available disk space.

> This space is needed for backup and temporary files. The backup can be safely deleted a few weeks after a successful migration.

---

## 7. Internet Connectivity

Ensure a **stable and fast internet connection**,
as the script will download a large `genesis.json` file (4–5 GB on mainnet).

---

## 8. Port Availability

Check that no other process is using the ports required by Heimdall v2.

| Port  | Protocol | Purpose                                                                 |
|-------|----------|-------------------------------------------------------------------------|
| 26656 | TCP      | CometBFT P2P                                                            |
| 26657 | HTTP     | CometBFT RPC (`/status`, `/broadcast_tx_sync`, etc.)                    |
| 26660 | HTTP     | CometBFT pprof (if enabled)                                             |
| 6060  | HTTP     | Alternate pprof port (Go default)                                       |
| 1317  | HTTP     | REST API (Cosmos SDK gRPC-Gateway)                                      |
| 9090  | gRPC     | Cosmos SDK gRPC server                                                  |
| 9091  | gRPC     | gRPC-Web server (optional)                                              |

For example, you can check with:
```bash
sudo lsof -i -P -n | grep LISTEN
# or
sudo netstat -tuln | grep LISTEN
# or
sudo ss -tuln
````

---

## 9. Validate Systemd Service User

Ensure the user running the Heimdall v1 service exists and is correct.

* You can check with:

  ```bash
  systemctl status heimdalld
  ```

  Look for the `User=` field.

* Confirm the actual process user:

  ```bash
  ps -o user= -C heimdalld
  ```

This user must be passed to the script via `--service-user` to ensure proper permissions and file ownership in v2.

---

## 10. Collect Required Parameters

Record the values for these flags before running the script:

| Flag                 | Description                                                                                                |
|----------------------|------------------------------------------------------------------------------------------------------------|
| `--heimdall-v1-home` | Path to Heimdall v1 home (must contain `config/` and `data/`)                                              |
| `--heimdallcli-path` | Path to `heimdallcli` binary (latest stable). Use `which heimdallcli`.                                     |
| `--heimdalld-path`   | Path to `heimdalld` binary (latest stable). Use `which heimdalld`.                                         |
| `--network`          | Target network: `mainnet` or `amoy`                                                                        |
| `--node-type`        | Node type: `sentry` or `validator`                                                                         |
| `--service-user`     | System user running Heimdall (as confirmed above)                                                          |
| `--generate-genesis` | Whether to export the genesis from local data. Set to `false` (recommended). Will be overridden if needed. |

> If the node cannot export the latest required block height, `--generate-genesis` will be automatically set to `false` and the script will download the genesis file from a trusted source.

---

## 11. Prepare the Script Command

Once you have all values, prepare the command. Always run it with `sudo` and `bash`:

```bash
sudo bash migrate.sh \
  --heimdall-v1-home=/var/lib/heimdall \
  --heimdallcli-path=/usr/bin/heimdallcli \
  --heimdalld-path=/usr/bin/heimdalld \
  --network=mainnet \
  --node-type=validator \
  --service-user=heimdall \
  --generate-genesis=false 2>&1 | tee migrate.log
```

Double-check every flag before execution. 
The script will validate all inputs before proceeding with migration.
