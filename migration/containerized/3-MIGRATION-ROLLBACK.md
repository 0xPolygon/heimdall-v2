# Rollback procedure

If the migration fails with some error, and you want to roll back to the previous state, follow this procedure:
1. Stop v2 heimdall container
2. Restore the backup of the v1 home directory
3. Install the previous version of heimdall-v1 image `v1.6.0-beta`.
4. Start heimdall v1
5. Check the logs
6. Potentially rerun the migration process when the issues are fixed.
