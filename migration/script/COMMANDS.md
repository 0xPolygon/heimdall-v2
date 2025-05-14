# Download migration script

```bash
curl -O https://raw.githubusercontent.com/<user>/<repo>/<branch>/migration.sh
```

# Verify checksum 
```bash
sha512sum -c migration.sh.sha512
```

# Launch migration script
```bash
  sudo bash migrate.sh \
    --heimdall-home=/var/lib/heimdall \
    --cli-path=/home/ubuntu/go/bin/heimdallcli \
    --d-path=/home/ubuntu/go/bin/heimdalld \
    --network=amoy \
    --nodetype=sentry \
    --backup-dir=/var/lib/heimdall.backup \
    --moniker=heimdall3 \
    --service-user=ubuntu \
    --generate-genesis=true \
    --bor-path=/home/ubuntu/go/bin/bor
```

# Reload daemon 
```bash
sudo systemctl daemon-reload 
```

# Start heimdall 
```bash
sudo systemctl start heimdalld
```

# Restart telemetry
```bash
sudo systemctl restart telemetry
```
