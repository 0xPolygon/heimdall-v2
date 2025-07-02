# Manual Migration

This guide is for performing a manual migration from Heimdall v1 to v2 using `systemd`.

---

### 1. Confirm Prerequisites

Ensure all required steps in the [Migration Checklist](../systemd/1-MIGRATION-CHECKLIST.md) are completed.

---

### 2. Stop Heimdall v1

```bash
sudo systemctl stop heimdalld
````

If using a custom service name, replace `heimdalld` accordingly.

---

### 3. Backup Heimdall v1 Data

Move the existing `HEIMDALL_HOME` (typically `/var/lib/heimdall`)
to a backup directory (e.g., `/var/lib/heimdall.backup`), or any other.

```bash
sudo mv /var/lib/heimdall /var/lib/heimdall.backup
```

This preserves `config` and `data` folders, And `bridge` folder as well, if used.

---

### 4. Backup the Systemd Service File

```bash
sudo mv /lib/systemd/system/heimdalld.service /lib/systemd/system/heimdalld.service.backup
```

---

### 5. Install Heimdall v2

You can use the installation script:

```bash
curl -L https://raw.githubusercontent.com/maticnetwork/install/heimdall-v2/heimdall-v2.sh | sudo bash -s -- v0.2.7 mainnet <NODE_TYPE>
```
where: 
- `NODE_TYPE` is `sentry` or `validator`

If the script fails, build from source:
```bash
git clone https://github.com/0xPolygon/heimdall-v2.git
cd heimdall-v2
git checkout v0.2.7
make build
sudo cp build/heimdalld /usr/bin/heimdalld
```

---

### 6. Verify Installation

```bash
heimdalld version
```

Output should match the `v0.2.7` installed.

---

### 7. Manually Migrate Configuration

Apply only the safe subset of configurations needed for v2 (remaining settings can be tuned later).

#### `config.toml` (v1 → v2):

Port the following from v1:

* `moniker`
* `external_address`
* `seeds`
* `persistent_peers`
* `max_num_inbound_peers`
* `max_num_outbound_peers`
* `proxy_app`
* `addr_book_strict`

Also set:

* `log_level = "info"`
* `log_format = "plain"`

#### `heimdall-config.toml` (v1) → `app.toml` (v2):

Port the following:

* `eth_rpc_url`
* `bor_rpc_url`
* `bor_grpc_flag`
* `bor_grpc_url`
* `amqp_url`

Also set:

* `bor_grpc_flag = false`
* `bor_rpc_timeout = "1s"`

#### `client.toml` (v2 only):

Set directly:

```toml
chain-id = "heimdallv2-137"
```

---

### 8. Restore the `bridge` Folder (If Used)

Move it from your backup into the new `HEIMDALL_HOME`.

---

### 9. Download Genesis File

```bash
wget -O <HEIMDALL_HOME>/config/genesis.json https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json
```

---

### 10. Download Checksum

```bash
wget -O <HEIMDALL_HOME>/config/genesis.json.sha512 https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json.sha512
```

---

### 11. Verify Genesis File

Move into the folder where you have downloaded the genesis file and its checksum.  
Generate the checksum of the `genesis.json` file by running

```
sha512sum genesis.json
```

The output will be something like
```bash
<CHECKSUM> genesis.json
```

Verify that the `CHECKSUM` string matches the one present in `genesis.json.sha512`

**Do not proceed if the checksum verification fails (string mismatch).**

---

### 12. Migrate `priv_validator_key.json`

Extract from the v1 file:

* `address`
* `pub_key.value`
* `priv_key.value`

Inject into the corresponding fields of v2’s `priv_validator_key.json`
**Do not change key types.**

---

### 13. Migrate `node_key.json`

Extract `priv_key.value` from v1 and overwrite the same field in v2.

This preserves the node’s identity (`node_id`).

---

### 14. Normalize `priv_validator_state.json`

In the v2 `HEIMDALL_HOME/data/priv_validator_state.json`, ensure that the `round` field is an integer (not a string).

Example:

```json
"round": 0  // ✅ valid
```

```json
"round": "0"  // ❌ invalid
```

Also, set the `height` field to `24404501`, e.g.,

```json
{
  "height": "24404501",
  "round": 0,
  "step": 0
}
```

---

### 15. Set File Ownership and Permissions

Ensure `heimdall` can access all necessary files:

```bash
sudo chown -R HEIMDALL_SERVICE_USER HEIMDALL_HOME
find HEIMDALL_HOME -type f -exec chmod 640 {} \;
find HEIMDALL_HOME -type d -exec chmod 755 {} \;

chmod 600 HEIMDALL_HOME/config/priv_validator_key.json
chmod 600 HEIMDALL_HOME/config/node_key.json
chmod 600 HEIMDALL_HOME/data/priv_validator_state.json
```

---

### 16. Update Systemd Service User/Group

Check with:

```bash
systemctl status heimdalld
```

Verify the `User=` and `Group=` match the v1 configuration.
If needed, edit `/lib/systemd/system/heimdalld.service` accordingly.
Use your previously backed-up service file as reference.

---

### 17. Reload and Start Heimdall v2

```bash
sudo systemctl daemon-reload
sudo systemctl start heimdalld
```

---

### 18. Restart Telemetry (If Needed)

```bash
sudo systemctl restart telemetry
```

---

### 19. Configure WebSocket for Bor ↔ Heimdall Communication

Edit Bor's `config.toml` to include:

```toml
[heimdall]
ws-address = "ws://localhost:26657/websocket"
```

---

### 20. Restart Bor (If Step 19 Was Applied)

```bash
sudo systemctl restart bor
```

---

### 21. Check Heimdall Logs

```bash
journalctl -fu heimdalld
```
