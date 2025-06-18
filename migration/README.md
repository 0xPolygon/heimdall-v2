
# Heimdall v1 to v2 Migration

This directory contains the documentation and tools to manage the migration from Heimdall v1 to Heimdall v2.
If you are willing to run Heimdall v2, you will need to migrate your existing Heimdall v1 node to the new version.  
This migration process is mostly automated, but still requires some manual steps to execute a coordinated upgrade.  

## ⚠️ Important notice on the migration process
The migration will be executed for the very first time on a node managed by the Polygon team (pilot node).  
Once the migration on that node is successful:
- The v1 genesis will be exported and made available for the community in a GCP bucket together with a checksum
- The v2 genesis will be created and made available for the community in a GCP bucket together with a checksum
- The script will be distributed with the checksum to prevent any tampering and made available for the community on heimdall-v2 repo
- Once the pilot node has been migrated, the genesis and the checksum files are available,
  and the script has been distributed, node operators can perform the migration on their own nodes using the [script](script/migrate.sh)
  (please check the [COMMANDS.md](./COMMANDS.md) in case).
  Operators can also execute the migration via [docker](script/DOCKER-README.md) or [manually](script/RUNBOOK.md).

### Containerized Migration
If you are using a containerized version of Heimdall (e.g. `docker` or inside a `kubernetes` cluster),
an image will be available to pull once the pilot node migration is successful.
You'd need to backup everything related to heimdall-v1, then pull the proper image and run heimdall-v2.
Note that the image will contain the `genesis.json` file, so you won't need to download it separately.
However, the file is going to be pretty large, especially for mainnet,
so make sure you have enough disk space available and you have a fast internet connection.
If you prefer to use the migration script, you'd need to make it compatible with your containerized environment.
Otherwise, you can use the [RUNBOOK](script/RUNBOOK.md) to run the migration process manually.
However, we strongly recommend using the containerized approach or the script to avoid mistakes and
ensure a smooth migration process.


### Non-containerized Migration
If you are not using a containerized version of Heimdall (e.g., `Linux/Debian`),
please refer to [these instructions](./script/README.md) for more information
on how to use the migration bash script.
Moreover, [this doc](./script/COMMANDS.md) provides a detailed list of commands to execute the v1→v2 migration.

### Manual Migration
If you prefer not to use any script, but rather execute the migration manually, you can follow the instructions in
the [runbook](./script/RUNBOOK.md).
However, we strongly recommend using the script or the containerized approach to avoid mistakes,  
execute fast, and ensure a smooth migration process.  

### Useful resources
[network](./networks) contains an example of genesis files and checksums for a devnet migration.
[configs](./configs) contains an example of configuration files for bor,
Heimdall v1 and Heimdall v2, with some instructions on how to migrate them.  
The migration script will manage the migration of these files automatically,
only considering the minimum required changes.
