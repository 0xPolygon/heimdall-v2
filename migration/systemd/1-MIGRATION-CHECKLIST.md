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

Make sure you're running the latest versions of `bor`/`erigon`, `heimdall` and `heimdallcli`:  
  * `bor v2.2.8` / `erigon v3.0.14`
  * `heimdall v1.6.0`
  * `heimdallcli v1.6.0`

---

## 2. Required Tools

The following tools must be installed. The script will fail early if any are missing.
Please ensure they are present on your system.

| Tool        | Purpose              | Install Command (Ubuntu/Debian) |
|-------------|----------------------|---------------------------------|
| `curl`      | Download binaries    | `sudo apt install curl`         |
| `tar`       | Extract archives     | `sudo apt install tar`          |
| `jq`        | JSON manipulation    | `sudo apt install jq`           |
| `sha512sum` | Integrity checks     | `sudo apt install coreutils`    |
| `file`      | File type detection  | `sudo apt install file`         |
| `awk`       | Text processing      | `sudo apt install gawk`         |
| `sed`       | Stream editing       | `sudo apt install sed`          |
| `tee`       | Output file and logs | `sudo apt install coreutils`    |
| `systemctl` | Service management   | Pre-installed on most systems   |
| `grep`      | Pattern matching     | Pre-installed on most systems   |
| `id`        | User info lookup     | Pre-installed on most systems   |

> Heimdall v2 will use `go 1.25.x`, so ensure your environment supports it.

---

## 3. Shell Requirements

Ensure `bash` is installed and being used while running the script.  
The script relies on `bash` features and will not work with `sh`.

---

## 4. Validate Heimdall v1 Config Files

Verify that the files in `HEIMDALL_HOME/config` are present and correctly formatted.

---

## 5. Memory Requirements

Ensure the system has **at least 20 GB of available RAM**.

---

## 6. Backup And Disk Space Requirements

If you plan to back up the v1 `HEIMDALL_HOME`,  
ensure the system has **at least 2× the size of `HEIMDALL_HOME`** in available disk space.  
If you use the script, the backup will be created under `HEIMDALL_HOME.backup`.  
If you are using the script and want to use a custom location or an external storage for the backup,  
you can set the `--backup` flag to `false` and then manually create the backup in your desired location.  

> This space is needed for backup and temporary files. The backup can be safely deleted a few weeks after a successful migration.

---

## 7. Internet Connectivity

Ensure a **stable and fast internet connection**,
as the script will download a large `genesis.json` file (4 GB on mainnet).

---

## 8. Port Availability

Make sure you have the following ports free on the host machine, so that heimdall-v2 can use them.

| Port          | Use                       | Protocol | Direction        | Notes                                                                 |
|---------------|---------------------------|----------|------------------|-----------------------------------------------------------------------|
| 26656         | **P2P**                   | **TCP**  | Inbound/Outbound | CometBFT peer-to-peer gossiping of blocks and transactions.           |
| 26657         | **RPC (CometBFT)**        | **TCP**  | Inbound          | Public CometBFT RPC API (e.g. `/status`, `/broadcast_tx_async`, etc). |
| 26660 or 6060 | **pprof (Profiling)**     | **TCP**  | Inbound          | Go’s `net/http/pprof` for debugging/profiling.                        |
| 1317          | **REST (Cosmos-SDK API)** | **TCP**  | Inbound          | REST endpoint via gRPC-Gateway from Cosmos SDK.                       |
| 9090          | **gRPC (Cosmos)**         | **TCP**  | Inbound          | Protobuf-based gRPC queries against app state.                        |
| 9091          | **gRPC-Web**              | **TCP**  | Inbound          | gRPC-Web server for browser clients.                                  |


Port `26656` (P2P) must be open on the validator to the sentry IPs.  
This is how sentries connect to the validator for block propagation and voting.  
Port `26657` (RPC) is generally not exposed on the validator.  
However, if your architecture requires sentries to relay transactions via RPC to the validator,  
you can open it to trusted IPs.  


For example, you can check that with:
```bash
sudo lsof -i -P -n | grep LISTEN
```
Or
```bash
sudo ss -tulpn
```

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

| Flag                 | Description                                                                                                                                                 |
|----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--heimdall-v1-home` | Path to Heimdall v1 home (must contain `config/` and `data/`)                                                                                               |
| `--heimdallcli-path` | Path to `heimdallcli` binary (latest stable). Use `which heimdallcli`.                                                                                      |
| `--heimdalld-path`   | Path to `heimdalld` binary (latest stable). Use `which heimdalld`.                                                                                          |
| `--network`          | Target network: `mainnet`                                                                                                                                   |
| `--node-type`        | Node type: `sentry` or `validator`                                                                                                                          |
| `--service-user`     | System user running Heimdall (as confirmed above)                                                                                                           |
| `--generate-genesis` | Whether to export the genesis from local data. Set to `false` (recommended). Will be overridden if needed.                                                  |
| `--backup`           | Whether to create the v1 backup under `HEIMDALL_HOME`.backup (recommended: true). If set to false, make sure to do the backup manually before the migration |

> If the node cannot export the latest required block height, `--generate-genesis` will be automatically set to `false` and the script will download the genesis file from a trusted source.
> If you set --backup to `false`, the script will NOT create a backup of the v1 home directory under `HEIMDALL_HOME.backup`. So make sure you crated your own backup in a custom location before proceeding with the migration.

---

## 11. Prepare the Script Command

Once you have all values, prepare the command but don't run it.
Make sure to have it ready with `sudo` and `bash`:

```bash
sudo bash migrate.sh \
  --heimdall-v1-home=/var/lib/heimdall \
  --heimdallcli-path=/usr/bin/heimdallcli \
  --heimdalld-path=/usr/bin/heimdalld \
  --network=mainnet \
  --node-type=validator \
  --service-user=heimdall \
  --generate-genesis=false \
  --backup=true 
```

Double-check every flag.  
The script will validate all inputs before proceeding with migration.  
It will also save a log file next to the script, named `migrate.log`, which you can review after the migration.    

> The mainnet migration will occur on July 10th, between 2 and 5 PM UTC.  
> As soon as the halt height `24404500` is reached, the Polygon team will create the v2 genesis and distribute to the community.  
> The process should take around 30-45 min.  
> Once the genesis is available, you can start the migration.  
> Expect heimdall downtime of around 2h (bor will continue working).  

**Reminder:** Please migrate the validators first, to ensure the stake is moved over to v2 as soon as possible, and avoid any potential issue with the new network.
