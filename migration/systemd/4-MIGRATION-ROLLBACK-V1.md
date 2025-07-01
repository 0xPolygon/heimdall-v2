# Rollback Procedure (Permanent)

If the migration fails and the Polygon team instructs node operators to **permanently** revert to Heimdall v1,
follow the steps below.

> **Do not use this procedure unless explicitly instructed by the Polygon team.**

---

### 1. Stop the `heimdalld` Service

This may be a v1 or v2 service, depending on the current state of migration:

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

### 3. Restore the v1 Systemd Service File

```bash
sudo mv -f /lib/systemd/system/heimdalld.service.backup /lib/systemd/system/heimdalld.service
```

---

### 4. Reinstall Heimdall v1

```bash
curl -L https://raw.githubusercontent.com/maticnetwork/install/main/heimdall.sh | bash -s -- <VERSION> <NETWORK> <NODE_TYPE>
```

Replace
- `VERSION` with the target version (TODO update it)
- `NETWORK` with `mainnet`
- `NODE_TYPE` with `sentry` or `validator`

---
---

### 5. Verify Installed Version

```bash
/usr/bin/heimdalld version
```

Expected output:

```
<VERSION>
```

> If the output shows a v2 version, manually move the v1 binary into the correct location.

---

### 6. Reload the Daemon and Start Heimdall

```bash
sudo systemctl daemon-reload && sudo systemctl start heimdalld
```

---

### 7. Restart Telemetry (If Needed)

```bash
sudo systemctl restart telemetry
```

---

### 8. Check Logs

```bash
journalctl -fu heimdalld
```

---

### 9. Chain Behavior After Rollback

Heimdall v1 should now be up and running.
No halt height is hardcoded, so the chain will automatically resume from the last committed block.
