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
    V1_VERSION="0.0.1-hv2-test"
    V2_VERSION="0.1.32"
    V1_CHAIN_ID="devnet"
    V2_CHAIN_ID="devnet"
    V2_GENESIS_TIME="2025-06-13T16:00:00Z"
    V1_HALT_HEIGHT=21710414
    VERIFY_EXPORTED_DATA=true
    ```
    where
   - `V1_VERSION` is the latest version of heimdall-v1 (currently, for testing is the version from `mardizzone/apocalypse` branch)
   - `V2_VERSION` is the latest version of heimdall-v2
   - `V1_CHAIN_ID` is the chain id of the heimdall-v1 network (`heimdall-137` for mainnet, or `heimdall-80002` for amoy, and `devnet` for testing)
   - `V2_CHAIN_ID` is the chain id of the heimdall-v2 network (pre-agreed)
   - `V2_GENESIS_TIME` is the genesis time of the v2 network (pre-agreed, it should be set in the future, e.g., 30 mins after the pilot migration is initiated)
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

9. Run the script with the following command (after modifying the parameters based on the previous step). Note that the script uses `bash` features, hence `sh` won't work.  
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
10. Copy the following file from the remote machine to the local one (they are located under `backup-dir`, which is `HEIMDALL_HOME.backup/`, typically `/var/lib/heimdall.backup/`):
    - `dump-genesis.json`
    - `dump-genesis.json.sha512`
    - `migrated_dump-genesis.json`
    - `migrated_dump-genesis.json.sha512`
     You can use the following commands from your local machine
    ```bash
     scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json ./
     scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/dump-genesis.json.sha512 ./
     scp <USER>@<NODE_IP>:/var/lib/heimdall.backup/migrated_dump-genesis.json.sha512 ./
      ```
11. Login to GCP by using the following command:
    ```bash
    gcloud auth login
    ```
12. Upload the `dump-genesis.json` to the GCP bucket so that they can be accessed by other node operators.
    For example, for `mumbai` bucket, you can use the following command:
    ```bash
      gsutil cp dump-genesis.json gs://mumbai-genesis-bucket
      ```
    In case the command fails,
    you can always upload the file manually [here](https://console.cloud.google.com/storage/browser/mumbai-genesis-bucket;tab=objects?authuser=1&inv=1&invt=Abz_Zg&project=prj-polygonlabs-pos-v1-dev&prefix=&forceOnObjectsSortingFiltering=false) with drag and drop
13. Update the following configs in the script:
     ```bash
     V1_GENESIS_CHECKSUM="4eb6ddd5d84f08f6db4c1cc0f1793cc8b68ac112684eae4ce2b26042a7a9b3645ac6657fda212d77e5881c54cbc829384e1fc31eb9ced167c6d10ac8afbadd7e"
     V2_GENESIS_CHECKSUM="02c4d40eada58ee8835bfdbe633bda07f2989bc0d65c18114df2cbfe4b07d8fdbbce3a72a1c3bfeef2b7fc9c295bbf5b4d5ede70c3fb480546625075459675e2"
     TRUSTED_GENESIS_URL="https://ffe639e7356c9c75efc1f5e069f06b2f8b5db92829dfb9cb6c837e4-apidata.googleusercontent.com/download/storage/v1/b/mumbai-genesis-bucket/o/dump-genesis.json?jk=AXbWWmkb96ODnd94uFXV5cLH2Jqh693AY2RUj4xy6SW_lQkOccqN8AVbXSUquJgaBSAjzoKXHXjNWed9WliFNc5D5HgU7abgyTpc0Zbnyy48S58_sjzful6n3JP13Ji9SNRGvzPaMM8W4RX5T_sotpXwCfyflqpk41p4_u3GcFfYAjMCpw_rBvjmGbnl23rmd2HIF7d6MifVQEnxkh4WvKK3v-KnOjXXA3exJByFGN1080t5u06N205Tb-8H1OGP2u5YLUQMGux7w_Oyzq6VOR8K-2N-0XtpfdpFC2yWGtpzqfWZIq9Q-wSqM_USy_7ekwZ-Aehr5RsKPr7YXJ5fVfO9bg-0FU3BVLn5wBbr76xg2aVeQu8FgYlF85EG0aJ4NVSzeH2IewwfSvWQySgeQ38x1EYN2k1Cis502bXmFbS2qGYr1bA3Trb93zjjNpTAKmYDdIIdtKI84RHR9cvE0wcalrd_WBMPOASdJMEQp5VlLFe6lNN9triVeQgONYtsO7Sb7uom6K-SxTnF3pDqYQGJ3cosNpd981LpFlOgAUCphF9yMzAL6C__2Fad9whRWjZh4dzAkC90tD0uHQctL_-_NQwaovTOGB9588KDGSHIz92lyHVbaq6hhpHA3VpE0K5Fds60VQI7AYhXy3KVNAkz4Ta-J1diAxPMPWvZARdFpEMwtEoZFopgahMVTlaMeF-FT1d6sz666ZjWQZkKuW7JetG3R0ghIXhWZkbZTg6y4YgWKeeXzmnSpVvVdQ_lgnjqVoBhHs725Lb99--7AHsdcFDAGeDftd_BuLE6Z7pGqCfnvddMxQLx09UJFDxctIF-WuIMTfvlF-sQxR2iNoZuHqs5Dvjkgtd4qbqIsy-WQOufGNvpm4IRna97VSTwKXopPgDVbwLOFGW7stBcEoZZ4Eer7C_CKMVaxGMH1lbzD-xkRlbxlZTDhPV10-IOFsOipKgpHa6t-UIEOyKivg6ldIM8IrifSraSZPzGyIpdksFmMFBF0Dqw42XXVFtN87TXFyLWrqKTr0Mrk-ma9_j4ReCjv-vsAjemCYD0-YX-UPCJOnuPiBLHqcPdEdRVZmUyjX9zwf3mIP3CMwy3n6e_8anNdrzmn8s4p3BSnoPsUXOOeS8mIMH0wae67pV0jYtCIytgCpzXF8JsilporA2vFoTM0L7-IDYorOYY1uAxvyoqm3mpGgZYUgIQmO0X69Qk3Vpol52UvnJAWj2G5lri3m4UNNY75NBt&isca=1"
     VERIFY_EXPORTED_DATA=false
     ```
    where
    - `V1_GENESIS_CHECKSUM` is the content of `dump-genesis.json.sha512`
    - `V2_GENESIS_CHECKSUM` is the content of `migrated_dump-genesis.json.sha512`
    - `TRUSTED_GENESIS_URL` is the URL of the genesis file (previously updated on a GCP bucket). It can be fetched by clicking "Download" on the file in the GCP bucket, and copying the URL from the browser (e.g., [here](https://console.cloud.google.com/storage/browser/_details/mumbai-genesis-bucket/dump-genesis.json;tab=live_object?authuser=1&inv=1&invt=Abz_Zg&project=prj-polygonlabs-pos-v1-dev))
    - `VERIFY_EXPORTED_DATA` is set to `false` because the genesis data has been already verified on the pilot node, and this will save some time and computational resources on other nodes.  
14. cd into the migration script folder
    ```bash
    cd heimdall-v2/migration/script
    ```
15. generate the checksum of the [script](migrate.sh) by running
     ```bash
     sha512sum migrate.sh > migrate.sh.sha512
     ```
16. When the script finishes run the following commands to reload the daemon, and start `heimdall`
    ```bash
    sudo systemctl daemon-reload && sudo systemctl start heimdalld
    ```
17. Restart telemetry (if needed)
    ```bash
    sudo systemctl restart telemetry
    ```
18. check the logs by running
   ```bash
      journalctl -fu heimdalld
   ```
19. If the genesis time is set in the future, `heimdalld` will print something like:
    ```bash
    heimdalld[147853]: 10:57AM INF Genesis time is in the future. Sleeping until then... genTime=2025-05-15T14:15:00Z module=server
    ```
    Otherwise, it will start syncing immediately
    (trying to connect to peers and throw errors if they are not yet available, in that case it will eventually crash, and it'll need to be restarted when other peers complete the migration).
20. Wait until the genesis time is reached, and the node will start syncing.
21. Now other node operators can run the migration.


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
       --heimdall-v1-home=/var/lib/heimdall \
       --heimdallcli-path=/usr/bin/heimdallcli \
       --heimdalld-path=/usr/bin/heimdalld \
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
   Or build it manually (if the `heimdall-tester` repo is being used),
   and replace the binary in `/usr/bin/heimdalld` and `/usr/bin/heimdallcli` with the one
   built from the `avalkov/reset` branch.
   ```bash
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
