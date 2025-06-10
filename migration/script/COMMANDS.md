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
    V1_VERSION="1.2.3-35-gdafcbb67"
    V2_VERSION="0.1.31"
    V1_CHAIN_ID="devnet"
    V2_CHAIN_ID="devnet"
    V2_GENESIS_TIME="2025-06-10T00:00:00Z"
    V1_HALT_HEIGHT=900
    VERIFY_EXPORTED_DATA=true
    ```
    where
   - `V1_VERSION` is the latest version of heimdall-v1 (currently, for testing is the version from `mardizzone/apocalypse` branch)
   - `V2_VERSION` is the latest version of heimdall-v2
   - `V1_CHAIN_ID` is the chain id of the heimdall-v1 network (`heimdall-137` for mainnet, or `heimdall-80002` for amoy, and `devnet` for testing)
   - `V2_CHAIN_ID` is the chain id of the heimdall-v2 network (pre-agreed)
   - `V2_GENESIS_TIME` is the genesis time of the v2 network (pre-agreed, it should be set in the future, e.g., 30mins after the pilot migration is initiated)
   - `V1_HALT_HEIGHT` is the height of the heimdall-v1's last block the (pre-agreed, it should match the height defined in `APOCALYPSE_TAG`)
   - `VERIFY_EXPORTED_DATA` is set to `true` because the genesis data will be verified on the pilot node.  
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
    --heimdall-v1-home=/var/lib/heimdall \
    --heimdallcli-path=/usr/bin/heimdallcli \
    --heimdalld-path=/usr/bin/heimdalld \
    --network=amoy \
    --node-type=validator \
    --service-user=ubuntu \
    --generate-genesis=true
    ```
   This will create `heimdall-v2` into `/var/lib/heimdall`
10. Copy the following files from the remote machine to the local one (they are located under `backup-dir`, which is `HEIMDALL_HOME.backup/`, typically `/var/lib/heimdall.backup/`):
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
11. Upload such files to the GCP bucket so that they can be accessed by other node operators.
    - For example, you can upload them to the GCP bucket `heimdall-genesis` with the following command:
      ```bash
      gsutil cp dump-genesis.json gs://heimdall-genesis/
      gsutil cp dump-genesis.json.sha512 gs://heimdall-genesis/
      gsutil cp migrated_dump-genesis.json gs://heimdall-genesis/
      gsutil cp migrated_dump-genesis.json.sha512 gs://heimdall-genesis/
      ```
12. Update the following configs in the script:
     ```bash
     V1_GENESIS_CHECKSUM="4eb6ddd5d84f08f6db4c1cc0f1793cc8b68ac112684eae4ce2b26042a7a9b3645ac6657fda212d77e5881c54cbc829384e1fc31eb9ced167c6d10ac8afbadd7e"
     V2_GENESIS_CHECKSUM="02c4d40eada58ee8835bfdbe633bda07f2989bc0d65c18114df2cbfe4b07d8fdbbce3a72a1c3bfeef2b7fc9c295bbf5b4d5ede70c3fb480546625075459675e2"
     TRUSTED_GENESIS_URL="https://ff4e1ab493fa24466b3a3009a12c1d75fb5c73be934b516ae874408-apidata.googleusercontent.com/download/storage/v1/b/devnet-genesis-bucket/o/dump-genesis.json?jk=AXbWWml5MvxXEdQk3zUyyhgkH-5Ot7rZk2MedTvPZHkddiQSShcH7x4mxX1ouiPvLYFEFR7ghAD3y8CQ-rqh3dV7j_K3gEtgdE0jjlcdp8D-9uljy0emzfOMpmvD5Gb1nz1eUhH-Apn4ELaVAuy8VvFAprUkRG-UlFtpcgoHjYm1exhNCMTImWKBDJrn_-SDA6loZNIMHUCpWlBPYCptzcSerGTehLOQdyrAnY4-vVPpgzMSaSP6BvsOgD-0a05xeilUvYQDCbSla21LBavH5F9SjgT8hTZgY9rz8Bt75ZxmsX16OkWem1T2Quoqg6dd5TvSqG7Au994eYve8gzMBucauWvTHgOkyV0mLytPK5CSfcGemy4L84lc-hTxCYO047untGYrDthYLJdi-jV4c-u82nNrjY8ZvVI8UIyhapFPFTcks9EJYVGdEDFFaGNZwcMo4CBfLKgwPgNdxREhDkZzT7iZrTwp1_dttf1Tob4FLjacJKMz1W6uJjTZ8ifJsJlyqiXDgL0E_NeuZNpH6pyd-L9LfgbAbxA6LjWMwScCUOXhG_F3O9dH8QGu5GAYjhO5BMMxvfnaIB25QBgYzzFsw0Es67kT6TDLKmjUlGJD27xsZhzMRegKJ4PMXLT5A8EHIQMlsv23lYwrmVx-ndc6kcBAPM02CLqiah5rivgFV2rDM-82NZviiH0BeHUgVtu8MEGKdm4mjVolTjUXwvR8AKSQHIkZ1pXZGC2AmLkZdjEP5pNBOwtL0PL9xSBGtbJqCcQ319RREqKKxvnCZ-8IK038FYC800xZawI3lVfDr0uilpVVkXzyRaO_Ruh8gPiqxJs74XOuKX5ezxlyjx81lfLQhssV4g7DnjP8-1nVXXrQGU0PnHveX0cGBQS2MpqD64LG5EyB-rjLcxLQqjC2Oy6xAMCBoUBD1c3LOeqJeK6SD8_CEqThWvZSfqNc5zSom401pW6jkQ05AW9z9insKauEo34djbXeVPy3Oq3ne7zKDLYApr0Z-Sp-M5GZ3hSi_DXaOIIy3lwg-hU9RdBn_F3EsMYEWQjS_p8dCQD_fZGIffbSIglW4SfzfsVOMy83p8OMujHnMeuvL9IL6O1O8hjZKFm8d5mKF4f2Ig_XaVqrEDp0aOgGGqfUk609JaI_HeVzm9iPammQ3_3k9cxzkRRDPaTlRsb9gZMHcItf8RNlnfNbdXHHLceJ_5QMtEfqunxB9UkpWF_zyrZnH9_FjVR_EtQFd-XH&isca=1"
     VERIFY_EXPORTED_DATA=false
     ```
    where
    - `V1_GENESIS_CHECKSUM` is the content of `dump-genesis.json.sha512`
    - `V2_GENESIS_CHECKSUM` is the content of `migrated_dump-genesis.json.sha512`
    - `TRUSTED_GENESIS_URL` is the URL of the genesis file (previously updated on a GCP bucket).
    - `VERIFY_EXPORTED_DATA` is set to `false` because the genesis data has been already verified on the pilot node, and this will save some time and computational resources on other nodes.  
13. cd into the migration script folder
    ```bash
    cd heimdall-v2/migration/script
    ```
14. generate the checksum of the [script](migrate.sh) by running
     ```bash
     sha512sum migrate.sh > migrate.sh.sha512
     ```
15. When the script finishes, run the following commands to reload the daemon, and start `heimdall`
    ```bash
    sudo systemctl daemon-reload && sudo systemctl start heimdalld
    ```
16. Restart telemetry (if needed)
    ```bash
    sudo systemctl restart telemetry
    ```
17. check the logs by running
   ```bash
      journalctl -fu heimdalld
   ```
18. If the genesis time is set in the future, `heimdalld` will print something like:
    ```bash
    heimdalld[147853]: 10:57AM INF Genesis time is in the future. Sleeping until then... genTime=2025-05-15T14:15:00Z module=server
    ```
    Otherwise, it will start syncing immediately
    (trying to connect to peers and throw errors if they are not yet available).
19. Wait until the genesis time is reached, and the node will start syncing.
20. Now other node operators can run the migration.


# Other executions (internal and external)

This can be run by any node operator.

1. check that all the config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted
2. download the script
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/migrate-tests/migration/script/migrate.sh
   ```
3. download the checksum
   ```bash
   curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/mardizzone/migrate-tests/migration/script/migrate.sh.sha512
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
       --heimdallcli-path=/home/ubuntu/go/bin/heimdallcli \
       --heimdalld-path=/home/ubuntu/go/bin/heimdalld \
       --network=amoy \
       --node-type=validator \
       --service-user=ubuntu \
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
