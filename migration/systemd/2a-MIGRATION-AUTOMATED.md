# Automated Migration

---

## Internal Pilot Migration (Polygon Team Only)

> **DO NOT run this externally.** This is intended for internal use by the Polygon team on a fully synced Heimdall v1 node to bootstrap the v2 migration for all network participants.

---

### 1. Pre-Migration Requirements

- Ensure all [PRE-MIGRATION] tasks in the `heimdall-v2` JIRA epic are resolved.

---

### 2. Prepare Local Migration Script

Clone the Heimdall v2 repository and create a working branch:

```bash
git clone https://github.com/0xPolygon/heimdall-v2.git
cd heimdall-v2
git checkout develop
git pull
git checkout -b <BRANCH_NAME>
````

Edit the environment variables in `script/migrate.sh`:

```bash
V1_VERSION="1.6.0"
V2_VERSION="0.2.7"
V1_CHAIN_ID="heimdall-137"
V2_CHAIN_ID="heimdallv2-137"
V2_GENESIS_TIME="2025-07-10T20:00:00Z"
V1_HALT_HEIGHT=24404500
VERIFY_EXPORTED_DATA=true
```

Explanation:

* `V1_VERSION`: installed v1 version on this node
* `V2_VERSION`: upcoming v2 version
* `V2_GENESIS_TIME`: should be ~2 hours in the future from when pilot migration is triggered
* `VERIFY_EXPORTED_DATA`: must be `true` to validate the exported genesis from this node

---

### 3. Access the Pilot Node

```bash
ssh <USER>@<NODE_IP>
```

Ensure the following tools are installed:

* `curl`, `tar`, `jq`, `sha512sum`, `file`, `awk`, `sed`
* `systemctl`, `grep`, `id`

Also verify all config files under `HEIMDALL_HOME` are present and valid.

---

### 4. Setup and Execute the Migration Script

Create and edit the script locally:

```bash
nano migrate.sh
```

Paste the edited contents of the [script](./script/migrate.sh)`.

Collect required CLI flags:

| Flag                 | Description                                                                    |
|----------------------|--------------------------------------------------------------------------------|
| `--heimdall-v1-home` | Path to v1 home (`config/` and `data/` must exist)                             |
| `--heimdallcli-path` | Path to `heimdallcli`, e.g., `/usr/bin/heimdallcli`                            |
| `--heimdalld-path`   | Path to `heimdalld`, e.g., `/usr/bin/heimdalld`                                |
| `--network`          | `mainnet` or `amoy`                                                            |
| `--node-type`        | `validator` or `sentry`                                                        |
| `--service-user`     | System user (from `systemctl status heimdalld` and `ps -o user= -C heimdalld`) |
| `--generate-genesis` | `true` (will be overridden if export fails)                                    |

Execute the script:

```bash
sudo bash migrate.sh \
  --heimdall-v1-home=/var/lib/heimdall \
  --heimdallcli-path=/usr/bin/heimdallcli \
  --heimdalld-path=/usr/bin/heimdalld \
  --network=mainnet \
  --node-type=validator \
  --service-user=heimdall \
  --generate-genesis=true \
  2>&1 | tee migrate.log
```

---

### 5. Post-Migration Steps

Enable self-heal by editing `HEIMDALL_HOME/config/app.toml`:

```toml
sub_graph_url = "<SUBGRAPH_URL>"
enable_self_heal = "true"
sh_state_synced_interval = "1h0m0s"
sh_stake_update_interval = "1h0m0s"
sh_max_depth_duration = "24h0m0s"
```

Then reload the daemon and start heimdall. Eventually (if required) restart telemetry and print the logs:

```bash
sudo systemctl daemon-reload
sudo systemctl start heimdalld
sudo systemctl restart telemetry
journalctl -fu heimdalld
```

If the genesis time is in the future, you'll see:

```
Genesis time is in the future. Sleeping until then...
```

---

### 6. Extract and Share Genesis Artifacts

From your local machine, copy the genesis files and the checksums:

```bash
scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json ./
scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json.sha512 ./
scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/migrated_dump-genesis.json ./
scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/migrated_dump-genesis.json.sha512 ./
```

Upload to GCP:

```bash
gsutil cp dump-genesis.json gs://<BUCKET_NAME>/
```

Update `script/migrate.sh` locally:

```bash
V1_GENESIS_CHECKSUM="<dump-genesis.json.sha512>"
V2_GENESIS_CHECKSUM="<migrated_dump-genesis.json.sha512>"
TRUSTED_GENESIS_URL="https://storage.googleapis.com/mainnet-heimdallv2-genesis/dump-genesis.json"
VERIFY_EXPORTED_DATA=false
V2_VERSION="0.2.8"
```

Generate script checksum:

```bash
cd heimdall-v2/migration/script
sha512sum migrate.sh > migrate.sh.sha512
```

Push changes to the repo, create a PR, and tag a new release to make the Docker image available.

---

## Migration Script Execution (All Node Operators)

---

### 1. Preparation

Confirm you have verified the requirements in the [Migration Checklist](../systemd/1-MIGRATION-CHECKLIST.md).

---

### 2. Download and Verify the Script

```bash
curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/script/migrate.sh
curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/script/migrate.sh.sha512
sha512sum -c migrate.sh.sha512
```

Expected output:

```
migrate.sh: OK
```

**Do not proceed if the checksum verification fails.**

---

### 3. Execute the Script

Prepare the command with the appropriate parameters:

```bash
sudo bash migrate.sh \
  --heimdall-v1-home=/var/lib/heimdall \
  --heimdallcli-path=/usr/bin/heimdallcli \
  --heimdalld-path=/usr/bin/heimdalld \
  --network=mainnet \
  --node-type=validator \
  --service-user=heimdall \
  --generate-genesis=false \
  2>&1 | tee migrate.log
```

This will initialize Heimdall v2 in `/var/lib/heimdall`.

---

### 4. Start Heimdall v2

```bash
sudo systemctl daemon-reload
sudo systemctl start heimdalld
```

Restart telemetry (if applicable):

```bash
sudo systemctl restart telemetry
```

Check logs:

```bash
journalctl -fu heimdalld
```

---

### 5. Sync from Genesis Time

If the genesis time is in the future, you will see:

```
Genesis time is in the future. Sleeping until then...
```

The node will begin syncing once the specified time is reached.
