# Pilot — First execution (internal)

This is run by the Polygon team on a synced `heimdall` node with `bor` running on the same machine  

1. Checkout heimdall-v2 repo locally
   ```bash
   git clone https://github.com/0xPolygon/heimdall-v2.git
   ```
2. Make sure the required software is installed on the machine, otherwise install them:
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

3. Adjust the following environment vars of the [script](migrate.sh):
    ```bash 
    V1_VERSION="1.2.3-35-gab6cbfb0"
    V2_VERSION="0.1.27"
    V1_CHAIN_ID="devnet"
    V2_CHAIN_ID="devnet"
    V2_GENESIS_TIME="2025-06-05T13:45:00Z"
    V1_HALT_HEIGHT=22238836
    V2_BRANCH_NAME="migration-mumbai"
    ```
   where 
   - `V1_VERSION` is the latest version of heimdall-v1 (currently, for testing is the version from `mardizzone/apocalypse` branch)
   - `V2_VERSION` is the latest version of heimdall-v2
   - `V1_CHAIN_ID` is the chain id of the heimdall-v1 network (`heimdall-137` for mainnet, or `heimdall-80002` for amoy, and `devnet` for testing)
   - `V2_CHAIN_ID` is the chain id of the heimdall-v2 network (pre-agreed during the gov proposal)
   - `V2_GENESIS_TIME` is the genesis time of the v2 network (pre-agreed during the gov proposal, it should be set in the future, e.g., 1h after the pilot migration is initiated)
   - `V1_HALT_HEIGHT` is the height of the heimdall-v1's last block the (pre-agreed during the gov proposal, it should match the height defined in `APOCALYPSE_TAG`)  
   - `V2_BRANCH_NAME` is the branch of the heimdall-v2 repo where the script will be pushed after the migration of the pilot node is completed.  
     
4. ssh into the node machine by using a valid user:
   ```bash
    ssh <USER>@<NODE_IP>
   ```
5. create the script with a command line editor (e.g., `nano`, `vim`, etc.):
    ```bash
    nano migrate.sh
    ```
6. Paste the content of the [script](migrate.sh) into the newly created file
7. Make sure that all the config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted
8. Retrieve the parameters needed by the script

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

9. Run the script with the following command (after modifying the parameters based on the previous step):
    ```bash
      sudo bash migrate.sh \
    --heimdall-v1-home=/mumbai/heimdall \
    --heimdallcli-path=/usr/local/bin/heimdallcli \
    --heimdalld-path=/usr/local/bin/heimdalld \
    --network=amoy \
    --node-type=validator \
    --service-user=heimdall \
    --generate-genesis=true
    ```
   This will create `heimdall-v2` into `/var/lib/heimdall`
10. cd into `heimdall-v2/migration/networks/<NETWORK>` where `<NETWORK>` is the `V1_CHAIN_ID` from step 2
    ```bash
    cd heimdall-v2/migration/networks/<NETWORK>
    ```
11. Copy the following files from the remote machine to the local one (they are located under `backup-dir`, which is `HEIMDALL_HOME.backup/`, typically `/var/lib/heimdall.backup/`):
    - `dump-genesis.json`
    - `dump-genesis.json.sha512`
    - `migrated_dump-genesis.json`
    - `migrated_dump-genesis.json.sha512`
    You can use the following commands from your local machine
    ```bash
     scp <USER>@<NODE_IP>:/mumbai/heimdall.backup/dump-genesis.json ./
     scp <USER>@<NODE_IP>:/mumbai/heimdall.backup/dump-genesis.json.sha512 ./
     scp <USER>@<NODE_IP>:/mumbai/heimdall.backup/migrated_dump-genesis.json ./
     scp <USER>@<NODE_IP>:/mumbai/heimdall.backup/migrated_dump-genesis.json.sha512 ./
    ```
12. Update the following configs in the script:
     ```bash
     V1_GENESIS_CHECKSUM="bf981f39f84eeedeaa08cd18c00069d1761cf85b70b6b8546329dbeb6f2cea90529faf90f9f3e55ad037677ffb745b5eca66e794f4458c09924cbedac30b44e7"
     V2_GENESIS_CHECKSUM="a128f317ffd9f78002e8660e7890e13a6d3ad21c325c4fa8fc246de6e4d745a55c465633a075d66e6a1aa7813fc7431638654370626be123bd2d1767cc165321"
     TRUSTED_GENESIS_URL="https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/<BRANCH_NAME>/migration/networks/devnet/dump-genesis.json"
     ```
     where 
     - `V1_GENESIS_CHECKSUM` is the content of `dump-genesis.json.sha512`
     - `V2_GENESIS_CHECKSUM` is the content of `migrated_dump-genesis.json.sha512`  
     - `TRUSTED_GENESIS_URL` is the URL of the genesis file (branch you are currently using).  
13. cd into the migration script folder
    ```bash
    cd heimdall-v2/migration/script
    ```
14. generate the checksum of the [script](migrate.sh) by running
     ```bash
     sha512sum migrate.sh > migrate.sh.sha512
     ```
15. cd into the root of the `heimdall-v2` repo
    ```bash
    cd ../..
    ```
16. commit and push the changes on `heimdall-v2` repo (they need to be available on the branch mentioned in `TRUSTED_GENESIS_URL`)   
17. When the script finishes, run the following commands to reload the daemon, and start `heimdall`
    ```bash
    sudo systemctl daemon-reload && sudo systemctl start heimdalld
    ```
18. Restart telemetry (if needed)
    ```bash
    sudo systemctl restart telemetry
    ```
19. check the logs by running
   ```bash
      journalctl -fu heimdalld
   ```
20. The genesis time is most probably set in the future so `heimdalld` will print something like:
    ```bash
    heimdalld[147853]: 10:57AM INF Genesis time is in the future. Sleeping until then... genTime=2025-05-15T14:15:00Z module=server
    ```
21. Wait until the genesis time is reached, and the node will start syncing.
22. Now other node operators can run the migration.


# Other executions (internal and external)

This can be run by any node operator.  

1. check that all the config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted  
2. download the script
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/migration-mumbai/migration/script/migrate.sh
   ```
3. download the checksum
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/migration-mumbai/migration/script/migrate.sh.sha512
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
       --heimdall-v1-home=/mumbai/heimdall \
       --heimdallcli-path=/usr/local/bin/heimdallcli \
       --heimdalld-path=/usr/local/bin/heimdalld \
       --network=amoy \
       --node-type=validator \
       --service-user=heimdall \
       --generate-genesis=true
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
   sudo rm -rf /mumbai/heimdall && \
   sudo mv /mumbai/heimdall.backup /mumbai/heimdall
    ```
3. Delete the genesis dump and its migrated version
   ```bash
   sudo rm -f /mumbai/heimdall/dump_genesis.json
   sudo rm -f /mumba/heimdall/dump_genesis.json.sha512
   sudo rm -f /mumbai/heimdall/migrated_dump_genesis.json
   sudo rm -f /mumbai/heimdall/migrated_dump_genesis.json.sha512
   ```
4. Restore the v1 service file (previously backed up by the script)
    ```bash
    sudo mv -f /lib/systemd/system/heimdalld.service.backup /lib/systemd/system/heimdalld.service
    ```
5. Install the "fallback version" of heimdall (without `halt_height` embedded). 
   This can be previously backed up or downloaded with the following command, after replacing the version tag, network name (`amoy` or `mainnet`), and node type (`sentry` or `validator`).
    ```bash
    curl -L https://raw.githubusercontent.com/maticnetwork/install/main/heimdall.sh | bash -s -- v<version> <network> <node_type>    ```
    ```
6. Check `heimdalld` version
    ```bash
    /usr/bin/heimdalld version
    # It should print
    # <version>
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
