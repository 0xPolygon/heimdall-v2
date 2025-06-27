# Manual Migration

1. Make sure you have all the prerequisites in place, as described in [Migration Checklist](../systemd/1-MIGRATION-CHECKLIST.md).
2. Stop Heimdall v1 by running:
   ```bash
   sudo systemctl stop heimdalld
   ```
   or, if you are using a different service name, replace `heimdalld` with the correct service name.
3. Backup Heimdall v1 by moving `HEIMDALL_HOME` directory (containing `config`, `data` and potentially `bridge`) into a different folder, e.g.:
   ```bash 
   sudo mv /var/lib/heimdall /var/lib/heimdall.backup
   ```
4. Backup the service file by running:
   ```bash
    sudo mv /lib/systemd/system/heimdalld.service /lib/systemd/system/heimdalld.service.backup
    ```
5. Install Heimdall-v2
   You can use the install-script as follows
   ```bash
   curl -L https://raw.githubusercontent.com/maticnetwork/install/heimdall-v2/heimdall-v2.sh | sudo bash -s -- <VERSION> <NETWORK> <NODE_TYPE>
   ```
   If this fails, you can always build the binary source:
   ```bash
    git clone https://github.com/0xPolygon/heimdall-v2.git
    cd heimdall-v2
    git checkout <VERSION>
    make build
    sudo cp build/heimdalld /usr/bin/heimdalld
    ```
6. Verify Heimdall-v2 installation by running the `version` command:
    ```bash
    heimdalld version
    ```
    It should print the `<VERSION>` of heimdall-v2 previously installed.
7. Apply the minimal configuration migration from the backed-up v1 configs to the ne2 v2 configs. 
   This is the same safe subset automatically handled by the migration script.
   You can tune other parameters later as needed. Please only apply the following for the sake of the migration to v2.
   You can apply the changes manually by editing the files under your v2 `HEIMDALL_HOME/config/` directory.
   From `config.toml` (v1) → `config.toml` (v2), port these values from your v1 node:
     - `moniker`
     - `external_address`
     - `seeds`
     - `persistent_peers`
     - `max_num_inbound_peers`
     - `max_num_outbound_peers`
     - `proxy_app`
     - `addr_book_strict`
   And set these additional static values:
     - `log_level = "info"`
     - `log_format = "plain"`
   From `heimdall-config.toml` (v1) → `app.toml` (v2), port these values from you v1 node:
     - `eth_rpc_url`
     - `bor_rpc_url`
     - `bor_grpc_flag`
     - `bor_grpc_url`
     - `amqp_url`
   And set these additional static values:
     - `bor_grpc_flag = false`
     - `bor_rpc_timeout = "1s"`
   In the new `client.toml` (v2), not present in v1, set the following value directly:
     - `chain-id = "heimdallv2-137"`
8. Move the bridge folder from the backup to the new `HEIMDALL_HOME` directory, if it was used in v1.
9. Download the v2 genesis file from the GCP bucket:  
   ```bash
   wget -O <HEIMDALL_HOME>/config/genesis.json https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json
   ```
10. Download the v2 genesis checksum file from the GCP bucket:  
    ```bash
    wget -O <HEIMDALL_HOME>/config/genesis.json.sha512 https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json.sha512
    ```
11. Verify the checksum of the downloaded genesis file:
   ```bash
   sha512sum -c migrated_dump-genesis.json.sha512
   ```
12. Ensure the `priv_validator_key.json` is ported over and formatted correctly.
    Retrieve the `address`, `pub_key.value`, and `priv_key.value` from `priv_validator_key.json` file of the v1 (backed-up) configs.
    Inject those values into the v2 `priv_validator_key.json` file under v2 `HEIMDALL_HOME/config`.
    Do not change the types in the new file, just port address, and the keys' values.
    This ensures that the validator retains the same private key and can continue signing blocks and participating in consensus.
13. Ensure the `node_key.json` is ported over and formatted correctly.
    Retrieve the `priv_key.value` field from `node_key.json` file of the v1 (backed-up) configs.
    Replace the v2 `HEIMDALL_HOME/config/node_key.json` file’s corresponding field with the value from v1.
    Doing this allows the node to preserve its original `node_id` and rejoin the network with its prior identity.
14. Normalize `priv_validator_state.json`
    This file tracks the validator’s latest consensus state locally.
    Make sure that v2 `round` field is an integer and not a string (no double quotes present) for v2's `HEIMDALL_HOME/data/priv_validator_state.json`.
15. Make sure the `HEIMDALL_HOME` directory has the right permissions and ownership, so that heimdall service can access it.
    We recommend setting `640` for all files and `755` for all directories under `HEIMDALL_HOME`.
    For sensitive files (`HEIMDALL_HOME/config/priv_validator_key.json`, `HEIMDALL_HOME/config/node_key.json`, `HEIMDALL_HOME/data/priv_validator_state.json`)
    we recommend setting `600` permissions, e.g.: 
    ```bash
    sudo chown -R "HEIMDALL_SERVICE_USER" "HEIMDALL_HOME"
    find "HEIMDALL_HOME" -type f -exec chmod 640 {} \;
    find "$HEIMDALL_HOME" -type d -exec chmod 755 {} \;    
    chmod 600 "$HEIMDALL_HOME/config/priv_validator_key.json"
    chmod 600 "$HEIMDALL_HOME/config/node_key.json"
    chmod 600 "$HEIMDALL_HOME/data/priv_validator_state.json"
    ```
16. Make sure the new heimdall v2 system file has the same user and group as the v1 service file.
    You can check that with `systemctl status heimdalld` and inspect the `User=` and `Group=` fields.
    If they are not set correctly, edit the new service file `/lib/systemd/system/heimdalld.service` and set the correct user and group.
    You should have backed up the v1 service file as reference.
17. Reload daemon and start heimdall with `sudo systemctl daemon-reload && sudo systemctl start heimdalld`
18. Restart telemetry (if needed) with `sudo systemctl restart telemetry`
19. WebSocket for Bor–Heimdall comm - Edit bor `config.toml` file by adding the following under the [heimdall] section:
    ```toml
    [heimdall]
    ws-address = "ws://localhost:26657/websocket"
    ```
20. Restart bor only in case the step above was done.
21. Check the logs
    ```bash
    journalctl -fu heimdalld
    ```
