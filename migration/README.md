
# Heimdall v1 to v2 Migration

Documentation and tools to manage the migration from Heimdall v1 to Heimdall v2.

Please refer to the [bash migration script README](./script/README.md) for more information
on how to use the migration script.  
The [commands](./script/COMMANDS.md) provide a detailed list of commands to execute the v1→v2 migration.  
The [runbook](./script/RUNBOOK.md) provides a step-by-step guide to execute the migration manually (in case the script doesn't work with your architecture).

[network](./networks) contains the network genesis files and checksums for Heimdall v2 on all supported networks.
Such files will be uploaded to a GCP bucket.  
[configs](./configs) contains an example of configuration files for bor,
Heimdall v1 and Heimdall v2, with some instructions on how to migrate them.
