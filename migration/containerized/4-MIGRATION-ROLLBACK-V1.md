# Rollback procedure

If the migration fails and the Polygon team instruct node operators to roll back permanently to heimdall v1, follow this procedure.
Do not use this file until you are instructed to do so by the Polygon team.
1. Stop heimdall container (it could be v1 or v2 based on the current state of the migration)
2. Restore the backup of the v1 home directory
3. Install the previous version of heimdall-v1 image `v1.5.0-beta`.
4. Start heimdall v1
5. Check the logs
6. Heimdall v1 should be up and running again, with no halt height hardcoded, hence the v1 chain will resume from the last committed block.
