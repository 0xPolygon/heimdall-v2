# Rollback procedure

If the migration fails with some error, and you want to roll back to the previous state, to retry the migration, follow this procedure:
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
5. Install the previous version of heimdall `v1.5.0-beta`.
   This can be previously backed up or downloaded with the following command, after replacing the version tag, network name (`amoy` or `mainnet`), and node type (`sentry` or `validator`).
    ```bash
    curl -L https://raw.githubusercontent.com/maticnetwork/install/main/heimdall.sh | bash -s -- v1.6.0-beta <network> <node_type>    ```
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
