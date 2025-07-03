# Rollback Procedure

If you executed the migration and heimdall-v2 fails to start or throws some errors,  
you can debug and fix the issues without rolling back to v1.

However, if the migration itself fails due to an error,
and you wish to roll back to the previous stable state to retry the migration, follow the steps below carefully.

---

### 1. Stop the Heimdall v2 Service

```bash
sudo systemctl stop heimdalld
````

> ℹ️ If the service does not exist, the command will fail harmlessly. You can safely ignore any related error.

---

### 2. Restore the v1 `HEIMDALL_HOME` Directory

⚠️ **Important:** Make sure the backup directory exists before deleting anything.

Run this to check:

```bash
ls -ld /var/lib/heimdall.backup
```

If the above command confirms the backup exists, proceed with:

```bash
sudo rm -rf /var/lib/heimdall # or any other v1 home you used as flag in the script (`--heimdall-v1-home`)
sudo mv /var/lib/heimdall.backup /var/lib/heimdall
```

> ❌ **Do not delete `/var/lib/heimdall` unless you have confirmed the backup exists!**
> Otherwise, you risk losing critical data.
> If the backup folder doesn't exist, skip this step and do not remove the current v1 home

---

### 3. Delete Genesis Dump Files

```bash
sudo rm -f /var/lib/heimdall/dump_genesis.json
sudo rm -f /var/lib/heimdall/dump_genesis.json.sha512
sudo rm -f /var/lib/heimdall/migrated_dump_genesis.json
sudo rm -f /var/lib/heimdall/migrated_dump_genesis.json.sha512
```

> Optional but recommended to ensure a clean state before retrying migration.
> If these files do not exist, it's not a problem.
---

### 4. Restore the v1 Systemd Service File

First, verify that the backup exists:

```bash
ls -l /lib/systemd/system/heimdalld.service.backup
```

If the file exists, you can restore it with:

```bash
sudo mv -f /lib/systemd/system/heimdalld.service.backup /lib/systemd/system/heimdalld.service
```

> ℹ️ If the backup does **not** exist, skip this step.
> You likely didn’t modify or back up the service file during migration.

### 5. Install Heimdall v1

If you don’t have the v1 binary already backed up, you can reinstall it using:

```bash
curl -L https://raw.githubusercontent.com/maticnetwork/install/main/heimdall.sh | bash -s -- <VERSION> <NETWORK> <NODE_TYPE>
```

Replace:

* `VERSION` with the target v1 version (TODO update it)
* `NETWORK` with `mainnet`
* `NODE_TYPE` with `sentry` or `validator`

---

### 6. Check the Installed Version

```bash
/usr/bin/heimdalld version
```

Expected output:

```
<VERSION>
```

If the output shows a v2 version, replace the binary manually with the correct v1 build.

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

> Optional, only needed if your telemetry service was configured separately.

---

### 9. Check the Logs

```bash
journalctl -fu heimdalld
```

Use this to monitor the node and confirm it is running correctly on v1.

---

### 10. Retry Migration When Ready

Once the underlying issues are resolved, you can rerun the migration script or proceed with a manual migration workflow.
