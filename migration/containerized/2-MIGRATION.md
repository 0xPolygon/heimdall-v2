# Containerized Migration

Ensure you have completed all prerequisites
outlined in the [Migration Checklist](../containerized/1-MIGRATION-CHECKLIST.md).

Please migrate the validators first, to ensure the stake is moved over to v2 as soon as possible,
and avoid any potential issue with the new network.

Once instructed by the Polygon team, please start the migration.  

## 1. Stop Existing Heimdall v1 Containers
- Gracefully shut down Heimdall v1, e.g., for docker:
  ```bash
  docker stop <container_id>
  ```

## 2. Prepare Backup
- Back up the `HEIMDALL_HOME` (default `/var/lib/heimdall`), containing `config/`, `data/` and `bridge/` folders outside the container.
- Example (for Docker):
  ```bash
  docker cp <container_id>:/var/lib/heimdall /path/to/backup
  ```
  
You can start with the backup as soon as the halt height `24404500` has been reached.

## 3. Pull the Heimdall v2 Image

Download the v2 Docker image from Docker Hub:

```bash
docker pull 0xpolygon/heimdall-v2:0.2.7
````

## 4. Initialize Default Configuration

Run the `init` command in the container to generate default config files, e.g.,

```bash
docker run --rm -v "<HEIMDALL_HOME>:/var/lib/heimdall" 0xpolygon/heimdall-v2:0.2.7 init <MONIKER> --chain-id heimdallv2-137
```
* `MONIKER` is your node name (any string), and it should match the moniker from your v1 configs (under `config.toml`)

This can't maybe be done in Kubernetes or other environments.  
In that case, place the default config files under `HEIMDALL_HOME/config` manually.  

## 5. Customization of configs

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

Also set in v2:

```toml
log_level = "info"
log_format = "plain"
```

And ensure the `seeds` and `persistent_peers` match the [default values](https://github.com/0xPolygon/heimdall-v2/blob/develop/packaging/templates/config/mainnet/config.toml#L216). 

### `heimdall-config.toml` (v1) → `app.toml` (v2):

Port the following from v1:

```toml
[custom]
eth_rpc_url = "..."
bor_rpc_url = "..."
bor_grpc_url = "..."
amqp_url = "..."
```

Also set in v2:

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

## 6. Download the Genesis File

Download the appropriate `genesis.json` from the following GCP bucket:

* [Mainnet genesis](https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json) *(available after pilot migration)*

Save the file with name `genesis.json`.

For example, you can use this command:
```bash
wget -O genesis.json https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json
```

**Note:** The genesis file is large (expected to be around 4 GB on Mainnet).   
Ensure you have sufficient disk space and a reliable, fast internet connection.

## 7. Verify genesis checksum

Move into the folder where you have downloaded the genesis file.

Download the checksum file available [here](https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json.sha512).
And name it `genesis.json.sha512`.  

For example, you can use this command:
```bash
wget -O genesis.json.sha512 https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json.sha512
```

And make sure this file is placed in the same folder as the `genesis.json`.  
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

## 8. Place the `genesis.json` under v2 home

Place the previously downloaded genesis.json in your `HEIMDALL_HOME/config` directory.

## 9. Migrate `priv_validator_key.json`

Extract from the v1 file (under v1's `HEIMDALL_HOME/config`, previously backed-up) the following fields,  
and inject into the corresponding fields of v2’s `priv_validator_key.json`.  
**Do not change key types, as in v1 they have a tendermint namespace and in v2 they have a comet namespace.**

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

## 10. Migrate `node_key.json`

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

## 11. Normalize `priv_validator_state.json`

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


## 12. Start the Heimdall v2 Container

Now that you have the right configuration and genesis file,  
you can run the container with the appropriate configuration.  
For example (please adjust the `-v` and `-p` options based on your deployment needs):

for non-validators:
```bash
docker run -d --name heimdall-v2 \
  -v "$HEIMDALL_HOME:/var/lib/heimdall" \
  -p 26656:26656 -p 26657:26657 -p 1317:1317 \
  0xpolygon/heimdall-v2:0.2.7 \
  start
```
and for validators:
```bash
docker run -d --name heimdall-v2 \
  -v "$HEIMDALL_HOME:/var/lib/heimdall" \
  -p 26656:26656 -p 26657:26657 -p 1317:1317 \
  0xpolygon/heimdall-v2:0.2.7 \
  start --bridge --all --chain=amoy --rest-server
```

If the genesis time is in the future, you will see logs like these:  

```
Genesis time is in the future. Sleeping until then...
```

Otherwise, the node will begin syncing immediately.  

## 13. Configure WebSocket for Bor ↔ Heimdall Communication

Edit Bor's `config.toml` to include:

```toml
[heimdall]
ws-address = "ws://localhost:26657/websocket"
```

---

## 14. Restart Bor (If Step 13 Was Applied)

Use your container orchestration tool to restart the Bor container.  

## 15. Upgrade telemetry (if used)
If you're using telemetry, you need to upgrade the service to be compatible with v2.  
Here the [instructions](https://github.com/vitwit/matic-telemetry/tree/heimdall-v2?tab=readme-ov-file#upgrading-for-heimdall-v2-version-api-change).   

### Keyring Info
*Not required for the migration*  
The encoding for v2 keys is different from v1, because it uses `base64` and no longer `hex`.  
If you plan to use the `heimdalld` to submit txs, you will need to import your keys into the keyring again.  
Instructions on how to do that are available in the [README](../../README.md) file.
