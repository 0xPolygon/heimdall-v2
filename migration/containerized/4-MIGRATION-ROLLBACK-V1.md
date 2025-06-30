# Full Rollback to Heimdall v1 (Permanent Reversion)

This procedure should only be followed
if the Polygon team explicitly instructs node operators to **permanently** revert to Heimdall v1.  
**Do not execute these steps unless directed by the Polygon team.**

## Steps

1. **Stop the Heimdall container**

   Depending on the state of your migration, you may be running either Heimdall v1 or v2.
   Stop the currently running container.

2. **Restore the v1 `HEIMDALL_HOME` directory**

   Use the backup you created before starting the migration to restore the original data and configuration.

3. **Reinstall the Heimdall v1 image**

   Pull the appropriate version of the v1 image, e.g., `1.5.0-beta`. TODO update version

   ```bash
   docker pull 0xpolygon/heimdall:<VERSION>
   ```

4. **Start the Heimdall v1 container**

   Re-run the container using your original mount paths and port mappings:

5. **Check the logs**

   Confirm that Heimdall v1 has started successfully:

6. **Verify normal operation**

   Heimdall v1 will resume from the latest committed block.
   No halt height will be configured, and the v1 chain should continue operating normally.

---

**Reminder:** Permanent rollback means abandoning the coordinated upgrade. Coordinate with the Polygon team before taking this step.
