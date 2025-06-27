# Rollback procedure

If the migration fails and the Polygon team instruct node operators to roll back permanently to heimdall v1, follow this procedure.
Do not use this file until you are instructed to do so by the Polygon team.
1. Stop `heimdalld` service (it can be v1 or v2 based on the current state of the migration)
   ```bash
   sudo systemctl stop heimdalld
   ```
2. Restore the backup of the v1 home directory, e.g.:
   ```bash
   sudo rm -rf /var/lib/heimdall && \
   sudo mv /var/lib/heimdall.backup /var/lib/heimdall
    ```
3. Restore the v1 service file
    ```bash
    sudo mv -f /lib/systemd/system/heimdalld.service.backup /lib/systemd/system/heimdalld.service
    ```
4. Install the previous version of heimdall `v1.5.0-beta`.
    ```bash
    curl -L https://raw.githubusercontent.com/maticnetwork/install/main/heimdall.sh | bash -s -- v1.5.0-beta <network> <node_type>
    ```
5. Check `heimdalld` version
    ```bash
    /usr/bin/heimdalld version
    # It should print
    # v1.5.0-beta
    ```
   If it still prints the v2 version, you need to move the v1 binary to the correct location.
6. Reload the daemon and start heimdall
   ```bash
   sudo systemctl daemon-reload && sudo systemctl start heimdalld
    ```
7. Restart telemetry (if needed)
   ```bash
   sudo systemctl restart telemetry
   ```
8. Check the logs
   ```bash
    journalctl -fu heimdalld
    ```
9. Heimdall v1 should be up and running again, with no halt height hardcoded, hence the v1 chain will resume from the last committed block.
