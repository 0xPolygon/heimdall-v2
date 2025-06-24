# (Internal) Pilot — First execution

This is run by the Polygon team on a synced `heimdall` node.
To let all node operators run the migration on their nodes, the pilot node must be migrated first.  

1. Make sure staking UI is stopped and all the [PRE-MIGRATION] tasks in JIRA under heimdall-v2 epic are resolved
2. Checkout `heimdall-v2` repo on your local machine and create a branch from `develop`. This will be needed to adjust the script parameters before distributing it to other node operators.  
   ```bash
    git clone https://github.com/0xPolygon/heimdall-v2.git
    cd heimdall-v2
    git checkout develop
    git pull 
    git checkout -b <BRANCH_NAME>
   ```
3. Adjust the following environment vars of the [script](migrate.sh):
    ```bash 
    V1_VERSION="1.6.0-beta"
    V2_VERSION="0.2.1"
    V1_CHAIN_ID="heimdall-80002"
    V2_CHAIN_ID="heimdallv2-80002"
    V2_GENESIS_TIME="2025-06-24T20:00:00Z"
    V1_HALT_HEIGHT=8788500
    VERIFY_EXPORTED_DATA=true
    ```
    where
   - `V1_VERSION` is the latest version of heimdall-v1 (currently, `v1.6.0-beta`)
   - `V2_VERSION` is the latest version of heimdall-v2 (currently, `v0.2.1`)
   - `V1_CHAIN_ID` is the chain id of the heimdall-v1 network (`heimdall-137` for mainnet, or `heimdall-80002` for amoy, and `devnet` for testing)
   - `V2_CHAIN_ID` is the chain id of the heimdall-v2 network (e.g., `heimdallv2-80002` for amoy and `heimdallv2-137` for mainnet)
   - `V2_GENESIS_TIME` is the genesis time of the v2 network (it should be set in the future, e.g., 30 mins after the pilot migration is initiated)
   - `V1_HALT_HEIGHT` is the height of the heimdall-v1's last block (`8788500` for amoy)
   - `VERIFY_EXPORTED_DATA` is set to `true` because the genesis data will be verified on the pilot node.  
4. `ssh` into the node machine by using a valid user:
   ```bash
    ssh <USER>@<NODE_IP>
   ```
5. Make sure the required software is installed, otherwise install them:
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
6. Make sure that all the config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted
7. Create the script with a command line editor (e.g., `nano`, `vim`, etc.):
    ```bash
    nano migrate.sh
    ```
8. Paste the content of the (modified) [script](migrate.sh) into the newly created file
9. Retrieve the parameters needed by the script

   | Flag                 | Description                                                                                                    |
   |----------------------|----------------------------------------------------------------------------------------------------------------|
   | `--heimdall-v1-home` | Path to Heimdall v1 home (must contain `config` and `data`)                                                    |
   | `--heimdallcli-path` | Path to `heimdallcli` (must be latest stable version). It can be retrieved with `which heimdallcli`            |
   | `--heimdalld-path`   | Path to `heimdalld` (must be latest stable version). It can be retrieved with `which heimdalld`                |
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

10. Run the script with the following command (after modifying the parameters based on the previous step). Note that the script uses `bash` features, hence `sh` won't work.  
     ```bash
       sudo bash migrate.sh \
     --heimdall-v1-home=/var/lib/heimdall \
     --heimdallcli-path=/usr/bin/heimdallcli \
     --heimdalld-path=/usr/bin/heimdalld \
     --network=amoy \
     --node-type=validator \
     --service-user=heimdall \
     --generate-genesis=true
     ```
    This will migrate heimdall and create its home under `/var/lib/heimdall`
11. When the script finishes, enable self-heal on v2 by setting the following in `app.toml` (for amoy):
    ```toml
    sub_graph_url = "https://api.studio.thegraph.com/query/113009/amoy-subgraph-polygon/version/latest"
    enable_self_heal = "true"
    sh_state_synced_interval = "1h0m0s"
    sh_stake_update_interval = "1h0m0s"
    sh_max_depth_duration = "24h0m0s"
    ```
12. Run the following commands to reload the daemon, and start `heimdall`
    ```bash
    sudo systemctl daemon-reload && sudo systemctl start heimdalld
    ```
13. Restart telemetry (if needed)
    ```bash
    sudo systemctl restart telemetry
    ```
14. check the logs by running
   ```bash
      journalctl -fu heimdalld
   ```
15. If the genesis time is set in the future, `heimdalld` will print something like:
    ```bash
    heimdalld[147853]: 10:57AM INF Genesis time is in the future. Sleeping until then... genTime=2025-05-15T14:15:00Z module=server
    ```
    Otherwise, it will start syncing immediately
    (trying to connect to peers and throw errors if they are not yet available,
    eventually leading to crash, so a new restart of heimdall could be needed later).
16. Wait until the genesis time is reached, and the node will start syncing.
17. Copy the following file from the remote machine to the local one (they are located under `backup-dir`, which is `HEIMDALL_HOME.backup/`, typically `/var/lib/heimdall.backup/`):
    - `dump-genesis.json`
    - `dump-genesis.json.sha512`
    - `migrated_dump-genesis.json`
    - `migrated_dump-genesis.json.sha512`
     You can use the following commands from your local machine
    ```bash
     scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json ./
     scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json.sha512 ./
     scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/migrated_dump-genesis.json ./
     scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/migrated_dump-genesis.json.sha512 ./
      ```
18. Upload the `dump-genesis.json` to the GCP bucket so that they can be accessed by other node operators:
    - You can drag and drop it on GCP console, or upload it to the GCP bucket with the following command (which requires `gcloud auth login` first):  
      ```bash
      gsutil cp dump-genesis.json gs://<BUCKET_NAME>/
      ```
19. Update the following configs in the script locally:
     ```bash
     V1_GENESIS_CHECKSUM="4eb6ddd5d84f08f6db4c1cc0f1793cc8b68ac112684eae4ce2b26042a7a9b3645ac6657fda212d77e5881c54cbc829384e1fc31eb9ced167c6d10ac8afbadd7e"
     V2_GENESIS_CHECKSUM="02c4d40eada58ee8835bfdbe633bda07f2989bc0d65c18114df2cbfe4b07d8fdbbce3a72a1c3bfeef2b7fc9c295bbf5b4d5ede70c3fb480546625075459675e2"
     TRUSTED_GENESIS_URL="https://storage.googleapis.com/amoy-heimdallv2-genesis/dump-genesis.json"
     VERIFY_EXPORTED_DATA=false
     V2_VERSION="0.2.2"
     ```
    where
    - `V1_GENESIS_CHECKSUM` is the content of `dump-genesis.json.sha512`
    - `V2_GENESIS_CHECKSUM` is the content of `migrated_dump-genesis.json.sha512`
    - `TRUSTED_GENESIS_URL` is the `Public URL` of the genesis file (previously updated on a GCP bucket).
    - `VERIFY_EXPORTED_DATA` is set to `false` because the genesis data has been already verified on the pilot node, and this will save some time and computational resources on other nodes.  
    - `V2_VERSION` is the next version we are going to release, which is likely to be `0.2.2` for amoy.
20. cd into the migration script folder
    ```bash
    cd heimdall-v2/migration/script
    ```
21. generate the checksum of the [script](migrate.sh) by running
     ```bash
     sha512sum migrate.sh > migrate.sh.sha512
     ```
22. Push the `migrated_dump-genesis.json` into the docker image as `genesis.json`, so that node operators using docker can pull the latest image and run the migration without needing to download the genesis file manually. 
23. Push all the changes (modified version of the script, checksum, etc...) to `heimdall-v2` repo, create a PR and merge it to `develop`.
24. Create a release from GitHub, so that the docker image is available for other node operators to pull.
25. Now other node operators can run the migration.


# Migration script execution - For all node operators

This can be run by any node operator.

1. check that all the config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted
2. download the script
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/script/migrate.sh
   ```
3. download the checksum
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/script/migrate.sh.sha512
   ```
4. verify the script checksum
   ```bash
   sha512sum -c migrate.sh.sha512
   ```
   This should output something like:
   ```bash
   migrate.sh: OK
   ```
   DO NOT run the script if the checksum verification fails!

5. Make sure the required software is installed on the machine, otherwise install them:
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

6. retrieve the parameters needed by the script

   | Flag                 | Description                                                                                                    |
   |----------------------|----------------------------------------------------------------------------------------------------------------|
   | `--heimdall-v1-home` | Path to Heimdall v1 home (must contain `config` and `data`)                                                    |
   | `--heimdallcli-path` | Path to `heimdallcli` (must be latest stable version). It can be retrieved with `which heimdallcli`            |
   | `--heimdalld-path`   | Path to `heimdalld` (must be latest stable version). It can be retrieved with `which heimdalld`                |
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

7. if checksum verification is correct, launch the migration script (after adjusting the parameters)
   ```bash
     sudo bash migrate.sh \
       --heimdall-v1-home=/var/lib/heimdall \
       --heimdallcli-path=/usr/bin/heimdallcli \
       --heimdalld-path=/usr/bin/heimdalld \
       --network=amoy \
       --node-type=validator \
       --service-user=heimdall \
       --generate-genesis=false
   ```
   This will create `heimdall-v2` into `/var/lib/heimdall`
8. When the script finishes, run the following commands to reload the daemon, and start `heimdall`
   ```bash
   sudo systemctl daemon-reload && sudo systemctl start heimdalld
   ```
9. Restart telemetry (if needed)
   ```bash
   sudo systemctl restart telemetry
   ```
10. check the logs by running
    ```bash
       journalctl -fu heimdalld
    ```
11. The genesis time is most probably set in the future so `heimdalld` will print something like:
     ```bash
     heimdalld[147853]: 10:57AM INF Genesis time is in the future. Sleeping until then... genTime=2025-05-15T14:15:00Z module=server
     ```
12. Wait until the genesis time is reached, and the node will start syncing.

# Rollback procedure
If the migration itself doesn't go as planned, you can roll back to the previous state by following this procedure:
1. Stop v2 `heimdalld` service
   ```bash
   sudo systemctl stop heimdalld
   ```
2. Restore the backup of the v1 home directory
   ```bash
   sudo rm -rf /var/lib/heimdall && \
   sudo mv /var/lib/heimdall.backup /var/lib/heimdall
    ```
3. Delete the genesis dump and its migrated version
   ```bash
   sudo rm -f /var/lib/heimdall/dump_genesis.json
   sudo rm -f /var/lib/heimdall/dump_genesis.json.sha512
   sudo rm -f /var/lib/heimdall/migrated_dump_genesis.json
   sudo rm -f /var/lib/heimdall/migrated_dump_genesis.json.sha512
   ```
4. Restore the v1 service file (previously backed up by the script)
    ```bash
    sudo mv -f /lib/systemd/system/heimdalld.service.backup /lib/systemd/system/heimdalld.service
    ```
5. Install the "fallback version" of heimdall (without `halt_height` embedded), `v1.5.0-beta`.
   This can be previously backed up or downloaded with the following command, after replacing the version tag, network name (`amoy` or `mainnet`), and node type (`sentry` or `validator`).
    ```bash
    curl -L https://raw.githubusercontent.com/maticnetwork/install/main/heimdall.sh | bash -s -- v1.5.0-beta <network> <node_type>    ```
    ```
6. Check `heimdalld` version
    ```bash
    /usr/bin/heimdalld version
    # It should print
    # v1.5.0-beta
    ```
   If it still prints the v2 version, you need to move the v1 binary to the correct location.
7. Reload the daemon and start heimdall
   ```bash
   sudo systemctl daemon-reload && sudo systemctl start heimdalld
    ```
8. Restart telemetry (if needed)
   ```bash
   sudo systemctl restart telemetry
   ```
9. Check the logs
   ```bash
    journalctl -fu heimdalld
    ```
10. Potentially rerun the migration process when the issues are fixed.


# (Internal) Verification
1. Once the migration is completed and the v2 network is up and running:
   - Make sure checkpoints are going through via APIs
   - If the next checkpoint is stuck in the buffer, send the ack message for it manually:
      ```bash
      heimdalld tx checkpoint send-ack --home /var/lib/heimdall --auto-configure=true
      ```
2. Make sure state syncs are going through via APIs
3. Update contract `RootChain` on L1 via method `RootChain.setHeimdallId` with the chainId previously agreed
4. Resolve all the [POST-MIGRATION] tasks in JIRA under heimdall-v2 epic
