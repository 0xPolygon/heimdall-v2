# Heimdall v1 → v2 Migration Guide

This directory contains migration guides for upgrading Heimdall from v1 to v2.
Two deployment environments are supported:

---

## [containerized](./containerized)

Use this if you run Heimdall using Docker, Kubernetes, or other containerized environments.

Contents:
- `1-MIGRATION-CHECKLIST.md`: Pre-migration checklist. Verify ahead of the migration time
- `2-MIGRATION.md`: Step-by-step migration instructions.
- `3-MIGRATION-ROLLBACK.md`: Temporary rollback procedure (to v1) if the migration fails.

---

## [systemd](./systemd)

Use this if your Heimdall v1 runs as a `systemd` service.

This folder contains instructions for **both automated and manual** migration.
Automated migration is recommended for simplicity and reduced risk of errors.

Contents:
- `1-MIGRATION-CHECKLIST.md`: Pre-migration checklist. Verify ahead of the migration time.
- `2a-MIGRATION-AUTOMATED.md`: Instructions for using the automated migration [script](./systemd/script/migrate.sh).
- `2b-MIGRATION-MANUAL.md`: Instructions for manual migration.
- `3-MIGRATION-ROLLBACK.md`: Temporary rollback procedure (to v1) if the migration fails.
- [`script/`](./systemd/script): Directory containing the automated migration script used in `2a-MIGRATION-AUTOMATED.md`.

---

### Choosing the Right Guide

Please follow the instructions based on your deployment type (`containerized` or `systemd`).    
In case of `systemd`, also sure your preferred migration path (automated or manual).
