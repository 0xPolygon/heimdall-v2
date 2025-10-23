# Automated Migration with provided [script](./script/migrate.sh)

---

## 1. Preparation

Confirm you have verified the requirements in the [Migration Checklist](../systemd/1-MIGRATION-CHECKLIST.md).

---

## 2. Download and Verify the Script

```bash
curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/systemd/script/migrate.sh
curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/systemd/script/migrate.sh.sha512
sha512sum -c migrate.sh.sha512
```

Expected output:

```
migrate.sh: OK
```

**Do not proceed if the checksum verification fails.**

---

## 3. Execute the Script

Prepare the command with the appropriate parameters:

```bash
sudo bash migrate.sh \
  --heimdall-v1-home=/var/lib/heimdall \
  --heimdallcli-path=/usr/bin/heimdallcli \
  --heimdalld-path=/usr/bin/heimdalld \
  --network=mainnet \
  --node-type=validator \
  --service-user=ubuntu \  
  --generate-genesis=true \ 
  --backup=true
```

This will initialize Heimdall v2 in `/var/lib/heimdall`.

---

## 4. Start Heimdall v2

```bash
sudo systemctl daemon-reload
sudo systemctl start heimdalld
```

## 5. Upgrade And Restart telemetry (if applicable):

If you're using telemetry, you need to upgrade the service to be compatible with v2.  
Here the [instructions](https://github.com/vitwit/matic-telemetry/tree/heimdall-v2?tab=readme-ov-file#upgrading-for-heimdall-v2-version-api-change).

Then you can restart the telemetry service:  
```bash
sudo systemctl restart telemetry
```

## 6. Check Heimdall Logs
Check logs:

```bash
journalctl -fu heimdalld
```

---

## 7. Sync from Genesis Time

If the genesis time is in the future, you will see:

```
Genesis time is in the future. Sleeping until then...
```

The node will begin syncing once the specified time is reached.

## 8. Configure WebSocket for Bor ↔ Heimdall Communication

Edit Bor's `config.toml` to include:

```toml
[heimdall]
ws-address = "ws://localhost:26657/websocket"
```

---

## 9. Restart Bor (If Step 8 Was Applied)

```bash
sudo systemctl restart bor
```

### Keyring Info
*Not required for the migration*  
The encoding for v2 keys is different from v1, because it uses `base64` and no longer `hex`.  
If you plan to use the `heimdalld` to submit txs, you will need to import your keys into the keyring again.  
Instructions on how to do that are available in the [README](../../README.md) file.
