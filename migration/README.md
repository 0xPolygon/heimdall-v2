# Heimdall v1 -> v2 migration guide

This folder contains the migration guide for Heimdall v1 to v2, in two different formats:

- [containerized](./containerized) contains instructions for containerized migration (e.g., docker or kubernetes setups).
  This folder contains:
  -`1-MIGRATION-CHECKLIST.md`: file with the checklist to run ahead of migration.
  -`2-MIGRATION.md`: file with the migration's instructions.
  -`3-MIGRATION-ROLLBACK.md`: file with the instructions to temporarily roll back to v1 in case the migration fails.
  -`4-MIGRATION-ROLLBACK-V1.md`: file with the instructions to roll back to v1 in case the migration fails.
    Do not use this file until you are instructed to do so by the Polygon team.
- [systemd](./systemd) contains instructions for the migration for users running Heimdall v1 on systemd.
  Inside this folder you will find instructions for both automated and manual migration.
  Automated migration is preferred, as it is less error-prone and easier to execute.
  Manual migration is provided as a fallback option in case the automated script does not work for your setup.
  This folder contains:
  -`1-MIGRATION-CHECKLIST.md`: file with the checklist to run ahead of migration.
  -`2a-MIGRATION-AUTOMATED.md`: file with the migration's instructions for automated script executions.
  -`2b-MIGRATION-MANUAL.md`: file with the migration's instructions for manual executions.
  -`3-MIGRATION-ROLLBACK.md`: file with the instructions to temporarily roll back to v1 in case the migration fails.
  -`4-MIGRATION-ROLLBACK-V1.md` file with the instructions to roll back to v1 in case the migration fails.
  Do not use this file until you are instructed to do so by the Polygon team.
  Also, this folder contains the [script](./systemd/script), which is used to automate the migration process 
  in case `2a-MIGRATION-AUTOMATED.md` is used.

Please follow the instructions based on your setup/environment and preferred option.
