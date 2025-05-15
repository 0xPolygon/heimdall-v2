# First execution (internal)

This is run by the Polygon team on a synced `heimdall` node with `bor` running on the same machine  

1. Adjust the env vars of the [script](migration.sh) to something like:
    ```bash
    APOCALYPSE_TAG="1.2.3-27-g74c8af58"
    REQUIRED_BOR_VERSION="2.0.0"
    HEIMDALL_V2_VERSION="0.1.15"
    CHAIN_ID="devnet"
    GENESIS_TIME="2025-05-15T14:15:00Z"
    APOCALYPSE_HEIGHT=200
    INITIAL_HEIGHT=$(( APOCALYPSE_HEIGHT + 1 ))
    VERIFY_DATA=true
    DUMP_V1_GENESIS_FILE_NAME="dump-genesis.json"
    DRY_RUN=false
    ```
2. ssh into the machine (as the user running `heimdalld`)
3. create the script as `sudo`
    ```bash
    sudo nano migrate.sh
    ```
4. paste the content of the [script](migration.sh) into the created file
5. run the script with a command like (after modifying the parameters):
    ```bash
      sudo bash migrate.sh \
    --heimdall-home=/var/lib/heimdall \
    --cli-path=/home/ubuntu/go/bin/heimdallcli \
    --d-path=/home/ubuntu/go/bin/heimdalld \
    --network=amoy \
    --nodetype=sentry \
    --backup-dir=/var/lib/heimdall.backup \
    --moniker=heimdall0 \
    --service-user=ubuntu \
    --generate-genesis=true \
    --bor-path=/home/ubuntu/go/bin/bor
    ```
6. copy the following files to the local machine 
   - `dump-genesis.json`
   - `dump-genesis.json.sha512`
   - `migrated_dump-genesis.json`
   - `migrated_dump-genesis.json.sha512`
7. copy/move such files under the respective files in the appropriate [network folder](../networks/)
8. update the following configs in the script:
    ```bash
    CHECKSUM="bf981f39f84eeedeaa08cd18c00069d1761cf85b70b6b8546329dbeb6f2cea90529faf90f9f3e55ad037677ffb745b5eca66e794f4458c09924cbedac30b44e7"
    MIGRATED_CHECKSUM="a128f317ffd9f78002e8660e7890e13a6d3ad21c325c4fa8fc246de6e4d745a55c465633a075d66e6a1aa7813fc7431638654370626be123bd2d1767cc165321"
    TRUSTED_GENESIS_URL="https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/e2e-test/migration/networks/devnet/dump-genesis.json"
    ```
    where `CHECKSUM` is the `dump-genesis.json.sha512` and `MIGRATED_CHECKSUM` is the `migrated_dump-genesis.json.sha512`
9. generate the checksum of the [script](migration.sh) by running
    ```bash
    sha512sum migration.sh > migration.sh.sha512
    ```
10. push the changes (they need to be available on the branch mentioned in `TRUSTED_GENESIS_URL`)   


# Other execution (internal and external)

This can be run by any node operator.  

1. download the script
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/e2e-test/migration/script/migrate.sh
   ```
2. verify the script checksum 
   ```bash
   sha512sum -c migrate.sh.sha512
   ```
3. launch the migration script (after adjusting the parameters)
   ```bash
     sudo bash migrate.sh \
       --heimdall-home=/var/lib/heimdall \
       --cli-path=/home/ubuntu/go/bin/heimdallcli \
       --d-path=/home/ubuntu/go/bin/heimdalld \
       --network=amoy \
       --nodetype=sentry \
       --backup-dir=/var/lib/heimdall.backup \
       --moniker=heimdall1 \
       --service-user=ubuntu \
       --generate-genesis=true \
       --bor-path=/home/ubuntu/go/bin/bor
   ```
4. When the script finishes, run the following commands to reload the daemon, and start `heimdall`
   ```bash
   sudo systemctl daemon-reload 
   sudo systemctl start heimdalld
   sudo systemctl restart telemetry
   ```
