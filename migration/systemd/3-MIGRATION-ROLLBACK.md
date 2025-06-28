# Rollback Procedure

If the migration fails due to an error,
and you wish to roll back to the previous state to retry the migration later, follow the steps below.

---

### 1. Stop the Heimdall v2 Service

```bash
sudo systemctl stop heimdalld
````

---

### 2. Restore the v1 `HEIMDALL_HOME` Directory

Example:
```bash
sudo rm -rf /var/lib/heimdall && \
sudo mv /var/lib/heimdall.backup /var/lib/heimdall
```

---

### 3. Delete Genesis Dump Files

```bash
sudo rm -f /var/lib/heimdall/dump_genesis.json
sudo rm -f /var/lib/heimdall/dump_genesis.json.sha512
sudo rm -f /var/lib/heimdall/migrated_dump_genesis.json
sudo rm -f /var/lib/heimdall/migrated_dump_genesis.json.sha512
```

---

### 4. Restore the v1 Systemd Service File

```bash
sudo mv -f /lib/systemd/system/heimdalld.service.backup /lib/systemd/system/heimdalld.service
```

---

### 5. Install Heimdall v1

If you don’t have the binary backed up, reinstall it using:

```bash
curl -L https://raw.githubusercontent.com/maticnetwork/install/main/heimdall.sh | bash -s -- <VERSION> <NETWORK> <NODE_TYPE>
```

Replace 
- `VERSION` with the target version (TODO update it)
- `NETWORK` with `mainnet`
- `NODE_TYPE` with `sentry` or `validator`

---

### 6. Check the Installed Version

```bash
/usr/bin/heimdalld version
```

Expected output:

```
<VERSION>
```

If the output shows the v2 version, manually replace the binary with the v1 version.

---

### 7. Reload the Daemon and Start Heimdall

```bash
sudo systemctl daemon-reload && sudo systemctl start heimdalld
```

---

### 8. Restart Telemetry (If Needed)

```bash
sudo systemctl restart telemetry
```

---

### 9. Check the Logs

```bash
journalctl -fu heimdalld
```

---

### 10. Retry Migration When Ready

Once the underlying issues are resolved, you can rerun the migration script or proceed with manual migration.

```
