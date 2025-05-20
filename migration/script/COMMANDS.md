# Pilot — First execution (internal)

This is run by the Polygon team on a synced `heimdall` node with `bor` running on the same machine  

1. Check that all the config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted  

2. Adjust the env vars of the [script](migrate.sh) to something like:
    ```bash
    APOCALYPSE_TAG="1.2.3-27-g74c8af58"
    REQUIRED_BOR_VERSION="2.0.0"
    HEIMDALL_V2_VERSION="0.1.18"
    CHAIN_ID="devnet"
    GENESIS_TIME="2025-05-15T14:15:00Z"
    APOCALYPSE_HEIGHT=200
    INITIAL_HEIGHT=$(( APOCALYPSE_HEIGHT + 1 ))
    VERIFY_DATA=true
    DUMP_V1_GENESIS_FILE_NAME="dump-genesis.json"
    DRY_RUN=false
    ```
3. ssh into the node machine (as the user running the `heimdalld` service)
   ```bash
    ssh <USER>@<NODE_IP>
   ```
4. create the script with `sudo`
    ```bash
    sudo nano migrate.sh
    ```
5. paste the content of the [script](migrate.sh) into the created file
6. retrieve the parameters needed by the script

   | Flag                 | Description                                                                                                    |
   |----------------------|----------------------------------------------------------------------------------------------------------------|
   | `--heimdall-home`    | Path to Heimdall v1 home (must contain `config` and `data`)                                                    |
   | `--cli-path`         | Path to `heimdallcli` (must be latest stable version). It can be retrieved with `which heimdallcli`            |
   | `--d-path`           | Path to `heimdalld` (must be latest stable version). It can be retrieved with `which heimdalld`                |
   | `--network`          | `mainnet` or `amoy`                                                                                            |
   | `--nodetype`         | `sentry` or `validator`                                                                                        |
   | `--backup-dir`       | Directory where a backup of Heimdall v1 will be stored. Recommended to use `<HEIMDALL_HOME>.backup`            |
   | `--moniker`          | Node moniker (must match the value in v1 `<HEIMDALL_V1_HOME>/config/config.toml`)                              |
   | `--service-user`     | System user running Heimdall (e.g., `heimdall`).                                                               |
   |                      | Check with: `systemctl status heimdalld` and inspect the `User=` field.                                        |
   |                      | Confirm it's correct by checking the user currently running the process (e.g., with `ps -o user= -C heimdalld` |
   |                      | This is critical to avoid permission issues in v2!                                                             |
   | `--generate-genesis` | Whether to generate genesis using `heimdalld` (recommended: `true`).                                           |
   |                      | Note that this value will be anyway overwritten by the script.                                                 |
   |                      | This happens when the node was not able to commit to the latest block's heigh needed for the migration,        |
   |                      | hence generate-genesis will be set to false and the genesis.json file downloaded from trusted source.          |

7. run the script with a command like the following (after modifying the parameters based on the previous step):
    ```bash
      sudo bash migrate.sh \
    --heimdall-home=/var/lib/heimdall \
    --cli-path=/home/ubuntu/go/bin/heimdallcli \
    --d-path=/home/ubuntu/go/bin/heimdalld \
    --network=amoy \
    --nodetype=validator \
    --backup-dir=/var/lib/heimdall.backup \
    --moniker=heimdall0 \
    --service-user=ubuntu \
    --generate-genesis=true \
    --bor-path=/home/ubuntu/go/bin/bor
    ```
8. cd into `heimdall-v2/migration/networks/<NETWORK>` where `<NETWORK>` is the `CHAIN_ID` from step 2
   ```bash
   cd heimdall-v2/migration/networks/<NETWORK>
   ```
9. copy the following files to the local machine (they are located under `backup-dir`, recommended to be `/var/lib/heimdall.backup/`):
   - `dump-genesis.json`
   - `dump-genesis.json.sha512`
   - `migrated_dump-genesis.json`
   - `migrated_dump-genesis.json.sha512`
   ```bash
    scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json ./
    scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json.sha512 ./
    scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/migrated_dump-genesis.json ./
    scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/migrated_dump-genesis.json.sha512 ./
   ```
10. update the following configs in the script:
     ```bash
     CHECKSUM="bf981f39f84eeedeaa08cd18c00069d1761cf85b70b6b8546329dbeb6f2cea90529faf90f9f3e55ad037677ffb745b5eca66e794f4458c09924cbedac30b44e7"
     MIGRATED_CHECKSUM="a128f317ffd9f78002e8660e7890e13a6d3ad21c325c4fa8fc246de6e4d745a55c465633a075d66e6a1aa7813fc7431638654370626be123bd2d1767cc165321"
     TRUSTED_GENESIS_URL="https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/e2e-test/migration/networks/devnet/dump-genesis.json"
     ```
     where `CHECKSUM` is the content of `dump-genesis.json.sha512`, and `MIGRATED_CHECKSUM` is the content of `migrated_dump-genesis.json.sha512`  
     and `TRUSTED_GENESIS_URL` is the URL of the genesis file (branch you are currently using).  
11. cd into the migration script folder
    ```bash
    cd heimdall-v2/migration/script
    ```
12. generate the checksum of the [script](migrate.sh) by running
     ```bash
     sha512sum migrate.sh > migrate.sh.sha512
     ```
13. cd into the root of the `heimdall-v2` repo
    ```bash
    cd ../..
    ```
14. commit and push the changes on `heimdall-v2` repo (they need to be available on the branch mentioned in `TRUSTED_GENESIS_URL`)   
15. When the script finishes, run the following commands to reload the daemon, and start `heimdall`
    ```bash
    sudo systemctl daemon-reload 
    sudo systemctl start heimdalld
    sudo systemctl restart telemetry
    ```
16. check the logs by running
   ```bash
      journalctl -fu heimdalld
   ```
17. The genesis time is most probably set in the future so `heimdalld` will print something like:
    ```bash
    heimdalld[147853]: 10:57AM INF Genesis time is in the future. Sleeping until then... genTime=2025-05-15T14:15:00Z module=server
    ```
18. Wait until the genesis time is reached, and the node will start syncing.
19. Now other node operators can run the migration.


# Other executions (internal and external)

This can be run by any node operator.  

1. check that all the config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted  
2. download the script
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/migration-tests/migration/script/migrate.sh
   ```
3. download the checksum
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/migration-tests/migration/script/migrate.sh.sha512
   ```
4. verify the script checksum 
   ```bash
   sha512sum -c migrate.sh.sha512
   ```
   This should output something like:
   ```bash
   migrate.sh: OK
   ```
5. retrieve the parameters needed by the script

   | Flag                 | Description                                                                                                    |
      |----------------------|----------------------------------------------------------------------------------------------------------------|
   | `--heimdall-home`    | Path to Heimdall v1 home (must contain `config` and `data`)                                                    |
   | `--cli-path`         | Path to `heimdallcli` (must be latest stable version). It can be retrieved with `which heimdallcli`            |
   | `--d-path`           | Path to `heimdalld` (must be latest stable version). It can be retrieved with `which heimdalld`                |
   | `--network`          | `mainnet` or `amoy`                                                                                            |
   | `--nodetype`         | `sentry` or `validator`                                                                                        |
   | `--backup-dir`       | Directory where a backup of Heimdall v1 will be stored. Recommended to use `<HEIMDALL_HOME>.backup`            |
   | `--moniker`          | Node moniker (must match the value in v1 `<HEIMDALL_V1_HOME>/config/config.toml`)                              |
   | `--service-user`     | System user running Heimdall (e.g., `heimdall`).                                                               |
   |                      | Check with: `systemctl status heimdalld` and inspect the `User=` field.                                        |
   |                      | Confirm it's correct by checking the user currently running the process (e.g., with `ps -o user= -C heimdalld` |
   |                      | This is critical to avoid permission issues in v2!                                                             |
   | `--generate-genesis` | Whether to generate genesis using `heimdalld` (recommended: `true`).                                           |
   |                      | Note that this value will be anyway overwritten by the script.                                                 |
   |                      | This happens when the node was not able to commit to the latest block's heigh needed for the migration,        |
   |                      | hence generate-genesis will be set to false and the genesis.json file downloaded from trusted source.          |

6. if checksum verification is correct, launch the migration script (after adjusting the parameters)
   ```bash
     sudo bash migrate.sh \
       --heimdall-home=/var/lib/heimdall \
       --cli-path=/home/ubuntu/go/bin/heimdallcli \
       --d-path=/home/ubuntu/go/bin/heimdalld \
       --network=amoy \
       --nodetype=sentry \
       --backup-dir=/var/lib/heimdall.backup \
       --moniker=heimdall3 \
       --service-user=ubuntu \
       --generate-genesis=true \
       --bor-path=/home/ubuntu/go/bin/bor
   ```
7. When the script finishes, run the following commands to reload the daemon, and start `heimdall`
   ```bash
   sudo systemctl daemon-reload 
   sudo systemctl start heimdalld
   sudo systemctl restart telemetry
   ```
8. check the logs by running
   ```bash
      journalctl -fu heimdalld
   ```
9. The genesis time is most probably set in the future so `heimdalld` will print something like:
    ```bash
    heimdalld[147853]: 10:57AM INF Genesis time is in the future. Sleeping until then... genTime=2025-05-15T14:15:00Z module=server
    ```
10. Wait until the genesis time is reached, and the node will start syncing.
