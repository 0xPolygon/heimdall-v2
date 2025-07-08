# Manual Migration

This guide is for performing a manual migration from Heimdall v1 to v2 using `systemd`.

---

## 1. Confirm Prerequisites

Ensure all required steps in the [Migration Checklist](../systemd/1-MIGRATION-CHECKLIST.md) are completed.

---

## 2. Stop Heimdall v1

Once instructed by the Polygon team, to start the migration, stop the Heimdall v1 service.  
```bash
sudo systemctl stop heimdalld
````

If using a custom service name, replace `heimdalld` accordingly.

---

## 3. Backup Heimdall v1 Data

Move the existing `HEIMDALL_HOME` (typically `/var/lib/heimdall`)
to a backup directory (e.g., `/var/lib/heimdall.backup`), or any other location (or external storage).  

```bash
sudo mv /var/lib/heimdall /var/lib/heimdall.backup
```

This preserves `config` and `data` folders, And `bridge` folder as well, if used.

You can start with the backup as soon as the halt height `24404500` has been reached.   

---

## 4. Backup the Systemd Service File

```bash
sudo mv /lib/systemd/system/heimdalld.service /lib/systemd/system/heimdalld.service.backup
```

---

## 5. Install Heimdall v2

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

## 6. Verify Installation

```bash
heimdalld version
```

Output should match the `v0.2.7` installed.

---

## 7. Manually Migrate Configuration

After initialization, customize the following files under `HEIMDALL_HOME/config`:
The templates for mainnet are available in the [Heimdall v2 GitHub repository](https://github.com/0xPolygon/heimdall-v2/tree/develop/packaging/templates/config/mainnet).

* `app.toml`
* `config.toml`
* `client.toml`

Please migrate your old v1 configurations by applying only the safe subset of configurations needed for v2
(remaining settings can be tuned later).

### `config.toml` (v1 → v2):

Port the following from v1:

```toml
moniker = "..." 
proxy_app = "..."
[p2p]
external_address = "..."
seeds = "..."
persistent_peers = "..."
max_num_inbound_peers = "..."
max_num_outbound_peers = "..."
```

Also set in v2

```toml
log_level = "info"
log_format = "plain"
```

### `heimdall-config.toml` (v1) → `app.toml` (v2):

Port the following:

```toml
[custom]
eth_rpc_url = "..." 
bor_rpc_url = "..."
bor_grpc_url = "..."
amqp_url = "..."
```

Also set:

```toml
[custom]
bor_grpc_flag = false
bor_rpc_timeout = "1s"
```

And make sure 
```toml
[custom]
chain = "mainnet"
```

### `client.toml` (v2 only):

Set directly:

```toml
chain-id = "heimdallv2-137"
```

---

## 8. Restore the `bridge` Folder (If Used)

Move it from your backup into the new `HEIMDALL_HOME`.

---

## 9. Download Genesis File

```bash
wget -O <HEIMDALL_HOME>/config/genesis.json https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json
```

---

## 10. Download Checksum

```bash
wget -O <HEIMDALL_HOME>/config/genesis.json.sha512 https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json.sha512
```

---

## 11. Verify genesis checksum

Move into the folder where you have downloaded the genesis file.
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

## 12. Migrate `priv_validator_key.json`

Extract from the v1 file (under v1's `HEIMDALL_HOME/config`, previously backed-up) the following fields,    
and inject into the corresponding fields of v2’s `priv_validator_key.json`  
*Do not change key types, as in v1 they have a tendermint namespace and in v they have a comet namespace.*  

Values to extract from v1's `priv_validator_key.json`:
* `address`
* `pub_key.value`
* `priv_key.value`

Example of v1's `priv_validator_key.json`:

```json
{
  "address": "...",
  "pub_key": {
    "type": "tendermint/PubKeySecp256k1",
    "value": "..."
  },
  "priv_key": {
    "type": "tendermint/PrivKeySecp256k1",
    "value": "..."
  }
}
```

Example of v2's `priv_validator_key.json`:

```json
{
  "address": "...",
  "pub_key": {
    "type": "cometbft/PubKeySecp256k1eth",
    "value": "..."
  },
  "priv_key": {
    "type": "cometbft/PrivKeySecp256k1eth",
    "value": "..."
  }
}
```

---

## 13. Migrate `node_key.json`

Extract `priv_key.value` from v1 and overwrite the same field in v2.  
This preserves the node’s identity (`node_id`).

Example of v1's `node_key.json`:

```json
{
  "priv_key":{
    "type":"tendermint/PrivKeyEd25519",
    "value":"..."
  }
}
```

Example of v2's `node_key.json`:

```json
{
  "priv_key":{
    "type":"tendermint/PrivKeyEd25519",
    "value":"..."
  }
}
```

---

## 14. Normalize `priv_validator_state.json`

In the v2 `HEIMDALL_HOME/data/priv_validator_state.json`, ensure that the `round` field is an integer (not a string).  
Also, set the `height` field to `24404501`, and make sure the only fields present are `height`, `round`, and `step`:


✅ This is valid:  
```json
{
  "height": "24404501",
  "round": 0,
  "step": 0
}
```

❌ This is invalid:  
```json
{
  "height": "24404501",
  "round": "0",
  "step": 0
}
```
---

## 15. Set File Ownership and Permissions

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

## 16. Update Systemd Service User/Group

Check with:

```bash
systemctl status heimdalld
```

Verify the `User=` and `Group=` match the v1 configuration.  
If needed, edit `/lib/systemd/system/heimdalld.service` accordingly.  
Use your previously backed-up service file as reference.

---

## 17. Reload and Start Heimdall v2

```bash
sudo systemctl daemon-reload
sudo systemctl start heimdalld
```

---

## 18. Upgrade And Restart Telemetry (If Needed)

If you're using telemetry, you need to upgrade the service to be compatible with v2.  
Here the [instructions](https://github.com/vitwit/matic-telemetry/tree/heimdall-v2?tab=readme-ov-file#upgrading-for-heimdall-v2-version-api-change).


```bash
sudo systemctl restart telemetry
```

---

## 19. Configure WebSocket for Bor ↔ Heimdall Communication

Edit Bor's `config.toml` to include:

```toml
[heimdall]
ws-address = "ws://localhost:26657/websocket"
```

---

## 20. Restart Bor (If Step 19 Was Applied)

```bash
sudo systemctl restart bor
```

---

## 21. Check Heimdall Logs

```bash
journalctl -fu heimdalld
```

If the genesis time is in the future, you will see:  

```
Genesis time is in the future. Sleeping until then...
```

Otherwise, the node will begin syncing immediately.  

### Keyring Info
*Not required for the migration*  
The encoding for v2 keys is different from v1, because it uses `base64` and no longer `hex`.  
If you plan to use the `heimdalld` to submit txs, you will need to import your keys into the keyring again.  
Instructions on how to do that are available in the [README](../../README.md) file.
