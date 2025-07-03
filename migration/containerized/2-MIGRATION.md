# Containerized Migration

Ensure you have completed all prerequisites
outlined in the [Migration Checklist](../containerized/1-MIGRATION-CHECKLIST.md).

If you are using Heimdall in a containerized setup (e.g., Docker or a Kubernetes cluster),
a v2 container image will be made available after the pilot node migration is confirmed successful.

## 1. Prepare Backup
- Back up the `HEIMDALL_HOME` (default `/var/lib/heimdall`), containing `config/` and `data/` folders outside the container.
- Example (Docker):
  ```bash
  docker cp <container_id>:/var/lib/heimdall /path/to/backup
  ```

## 2. Stop Existing Heimdall v1 Containers
- Gracefully shut down Heimdall v1, e.g., for docker:
  ```bash
  docker stop <container_id>
  ```
  and e.g., for Kubernetes:
  ```bash
  kubectl scale deployment heimdall --replicas=0
  ```

## 3. Pull the Heimdall v2 Image

Download the v2 Docker image from Docker Hub:

```bash
docker pull 0xpolygon/heimdall-v2:<VERSION>
````

Where `<VERSION>` is going to be communicated after the pilot node migration (e.g., `0.2.7`).

## 4. Initialize Default Configuration

Run the `init` command in the container to generate default config files, e.g.,

```bash
docker run --rm -v "<HEIMDALL_HOME>:/var/lib/heimdall" 0xpolygon/heimdall-v2:0.2.7 init <MONIKER> --chain-id <CHAIN_ID>
```

* `<MONIKER>` is your node name (any string), and it should match the moniker from your v1 configs (under `config.toml`).
* `<CHAIN_ID>` is `heimdallv2-137` for Mainnet

## 5. Customization of configs

After initialization, customize the following files under `HEIMDALL_HOME/config`:

* `app.toml`
* `config.toml`
* `client.toml`

The templates for mainnet are available in the [Heimdall v2 GitHub repository](https://github.com/0xPolygon/heimdall-v2/tree/develop/packaging/templates/config/mainnet).
Please migrate your old v1 configs, by applying only the safe subset of configurations needed for v2
(remaining settings can be tuned later).

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

Also set in v2:

* `log_level = "info"`
* `log_format = "plain"`

And ensure the `seeds` and `persistent_peers` match the [default values](https://github.com/0xPolygon/heimdall-v2/blob/develop/packaging/templates/config/mainnet/config.toml#L216). 

#### `heimdall-config.toml` (v1) → `app.toml` (v2):

Port the following from v1:

* `eth_rpc_url`
* `bor_rpc_url`
* `bor_grpc_flag`
* `bor_grpc_url`
* `amqp_url`

Also set in v2:

* `bor_grpc_flag = false`
* `bor_rpc_timeout = "1s"`

And make sure `chain = "mainnet"`

#### `client.toml` (v2 only):

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

### 9. Migrate `priv_validator_key.json`

Extract from the v1 file (under v1's `HEIMDALL_HOME/config`, previously backed-up):

* `address`
* `pub_key.value`
* `priv_key.value`

Inject into the corresponding fields of v2’s `priv_validator_key.json`
**Do not change key types.**

### 10. Migrate `node_key.json`

Extract `priv_key.value` from v1 and overwrite the same field in v2.

This preserves the node’s identity (`node_id`).

### 11. Normalize `priv_validator_state.json`

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

## 12. Start the Heimdall v2 Container

Now that you have the right configuration and genesis file,  
you can run the container with the appropriate configuration.
Example (please adjust the `-v` and `-p` options based on your deployment needs):

```bash
docker run -d --name heimdall-v2 \
  -v "$HEIMDALL_HOME:/var/lib/heimdall" \
  -p 26656:26656 -p 26657:26657 -p 1317:1317 \
  0xpolygon/heimdall-v2:<VERSION> \
  start
```

## Final Notes

* Verify that all ports are correctly mapped and not in use.
* Ensure sufficient system memory and disk availability before running the container.
* Monitor logs to confirm successful startup.
