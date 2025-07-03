# Automated Migration

## Migration Script Execution (All Node Operators)

---

### 1. Preparation

Confirm you have verified the requirements in the [Migration Checklist](../systemd/1-MIGRATION-CHECKLIST.md).

---

### 2. Download and Verify the Script

```bash
curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/script/migrate.sh
curl -O https://raw.githubusercontent.com/0xPolygon/heimdall-v2/refs/heads/develop/migration/script/migrate.sh.sha512
sha512sum -c migrate.sh.sha512
```

Expected output:

```
migrate.sh: OK
```

**Do not proceed if the checksum verification fails.**

---

### 3. Execute the Script

Prepare the command with the appropriate parameters:

```bash
sudo bash migrate.sh \
  --heimdall-v1-home=/var/lib/heimdall \
  --heimdallcli-path=/usr/bin/heimdallcli \
  --heimdalld-path=/usr/bin/heimdalld \
  --network=mainnet \
  --node-type=validator \
  --service-user=heimdall \
  --generate-genesis=false \
  2>&1 | tee migrate.log
```

This will initialize Heimdall v2 in `/var/lib/heimdall`.

---

### 4. Start Heimdall v2

```bash
sudo systemctl daemon-reload
sudo systemctl start heimdalld
```

Restart telemetry (if applicable):

```bash
sudo systemctl restart telemetry
```

Check logs:

```bash
journalctl -fu heimdalld
```

---

### 5. Sync from Genesis Time

If the genesis time is in the future, you will see:

```
Genesis time is in the future. Sleeping until then...
```

The node will begin syncing once the specified time is reached.
