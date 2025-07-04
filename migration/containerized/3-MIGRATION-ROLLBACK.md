# Rollback Procedure

If you executed the migration and heimdall-v2 fails to start or throws some errors,  
you can debug and fix the issues without rolling back to v1.  
Polygon team can guide you through the process if you reach out.  

However, if the migration itself fails due to an error,
and you wish to roll back to the previous stable state to retry the migration, follow the steps below carefully.

## Steps

1. **Stop the Heimdall v2 container** (if running)  
   Depending on the state of your migration, you may be running either Heimdall v1 or v2.

2. **Restore the v1 `HEIMDALL_HOME` backup**

   Replace the existing `HEIMDALL_HOME` directory (containing `/config`, `/data` and potentially `/bridge`)
   with your previously saved v1 backup

3. **Reinstall the previous Heimdall v1 image**

   Pull the correct version of the Heimdall v1 Docker image.

   ```bash
   docker pull 0xpolygon/heimdall:1.6.0
   ```

4. **Start the Heimdall v1 container**

   Launch it using the original configuration and data directory.

5. **Check the logs**

   Monitor startup logs to ensure Heimdall v1 is running as expected:

6. **Retry migration when ready**

   After resolving the issues that caused the failure, you may rerun the migration process from the beginning.

**Note:** Ensure that any state or chain data was not corrupted during the failed migration before attempting to restore v1.
