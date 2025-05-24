# Heimdall v1 -> v2 RUNBOOK

## ⚠️ Important notice on the migration process
The script will be executed the very first time on a node managed by Polygon team.  
Once the migration on that node is successful:
- The v1 genesis will be exported and made available for the community on heimdall-v2 repo together with a checksum
- The v2 genesis will be created and made available for the community on heimdall-v2 repo together with a checksum
- The script will be distributed with the checksum, to prevent any tampering, and made available for the community on heimdall-v2 repo
- Node operators can perform the migration on their own nodes using the script (or a modified version of it if the architecture is not supported)

### Overview

The heimdall-v2 deployment is a critical upgrade to the Polygon PoS stack, replacing heimdall with a new chain based on cometBFT `v0.38.x` and cosmos-sdk `v0.50.x`.  
This transition requires a structured and coordinated approach across multiple teams, ensuring minimal downtime and a seamless migration for node operators.

### Preparation

1. Preserve some RPC nodes for heimdall-v1 historical data (this can be done in advance)
2. **Governance Proposal Submission.** Includes:
   1. Agreed Heimdall v1 `halt_height`
   2. Release 'apocalypse' versions of Heimdall-v1, Bor and Erigon
   3. Release 'fallback' version of Heimdall-v1
   4. Release a final stable version of Heimdall-v2 (together with final stable versions of comet and cosmos-sdk)
   5. New Heimdall-v2 Chain ID
   6. Approximate time window for new genesis time (e.g., 1–2h after `halt_height`)
3. **Node Operators Update Bor/Erigon**  
4. **Node Operators Update Heimdall-v1**
5. **Confirmation of successful upgrade from all node operators**
6. **Advise all node operators to refrain from performing any actions on L1 involving the bridge until v2 is fully operational.** This precaution is intended to prevent any potential issues with the bridge during the migration process. As an additional safeguard, we could consider temporarily disabling the staking UI to further minimize the risk of unintended interactions.
7. Establish a real-time coordination hub for migration execution.

### Execution

The majority of the steps below are automated in the [migration script](migrate.sh), the following is just a runbook for manual execution.

1. Node operators confirm to be on latest Bor/Erigon version (compatible with v1 and v2)
2. Node operators confirm to be on latest Heimdall-v1 version (with `halt_height` embedded)
3. Node operators confirm to be on latest `heimdallcli` version (with `get-last-committed-height` embedded)
4. Node operators confirm they can execute `heimdalld` and `heimdallcli` by running `version` command for both of them
5. Node operators confirm the heimdall config files under `HEIMDALL_HOME/config` are correct and the files are properly formatted
6. Node operators confirm heimdall-v1 is down due to hitting `halt_height`  
   - This can be achieved with `heimdallcli get-last-committed-height --home HEIMDALLD_HOME --quiet` command
   - The output of this command should match the `halt_height`
   - If some nodes are not down, or a block's height mismatch is detected, it means they did not reach the apocalypse height  
   - This can happen, and the migration script handles this case by downloading the genesis.json from a trusted source (instead of generating it, to avoid risks of checksum mismatches and especially app hash errors in v2).  
7. Contract `RootChain` is updated on L1 via method `RootChain.setHeimdallId` with the chainId previously agreed (since this is not used, can be done in advance or after the migration)
8. **Genesis Export from Heimdall-v1**
   `heimdallcli export-heimdall --home HEIMDALLD_HOME`
   - operators should use the latest version of `heimdallcli`
   - `HEIMDALLD_HOME` is usually `/var/lib/heimdall` directory, and the command will generate `dump-genesis.json` there  
     It's recommended to run the process first on a fully synced pilot node.
9. **v1 Checksum Generation**
   The exported genesis file is checksummed.
10. **Backup Heimdall v1 Data** - Node operators back up their Heimdall v1 `HEIMDALL_HOME` directory (containing `config`, `data` and – for validators - `bridge`).
11. **Install Heimdall-v2** – Node operators install heimdall-v2 with the install script available at https://github.com/maticnetwork/install/blob/heimdall-v2/heimdall-v2.sh
12. **Verify Heimdall-v2 installation** – Node operators make sure `heimdall-v2` is successfully installed by running the `version` command
13. **Run Migration Command** - Converts the old genesis to v2 format using:
    ```bash
    heimdalld migrate dump-genesis.json --chain-id=<CHAIN_ID> --genesis-time=<TIME_IN_FORMAT_YYYY-MM-DDTHH:MM:SSZ) --initial-height=<H>
    ```
    where `H = v1_halt_height + 1`, whilst `--chain-id` and `genesis-time` are pre-agreed offline via governance proposal on v1.
14. **v2 Checksum Generation**
    The migrated genesis is used to generate its checksum.  
    At this point, script env vars can be set, then the script can be checksummed and distributed to the community (via git on heimdall-v2 repo).
    The genesis exports (v1 and migrated) will be made available to the community together with their checksums.
    The script will be distributed to the community together with its checksum.
15. Node operators delete (or rename) their `HEIMDALL_HOME` directory (**make sure it was backed up earlier**)
16. Node operators create a new `HEIMDALL_HOME` directory with the new Heimdall v2 binary by running `heimdalld init [moniker-name] --home=<V2_HOME> --chain-id=<CHAIN_ID>`.  
    The moniker is available in v1 at `HEIMDALL_HOME/config/config.toml`.
    The service file `heimdall.service` should reflect the same `User` (and possibly `Group`) coming from v1 service file.
17. Node operators edit their configuration files with config information from their backup, plus default values. For detailed info, see [here](../configs).
    1. v2 `app.toml` must reflect the “merge” from v1 `app.toml` and `heimdall-config.toml`
       - All values from v1 `app.toml` should NOT be carried over.
       - For `heimdall-config.toml`:
         - `heimdall_rest_server` key is no longer required in v2
    2. v2 `config.toml` also needs to be edited
       - Ensure `moniker` remains the same in v2.
       - Use `log_level = "info"` or `debug` (v2 default is `info`)
       - Do NOT port the following:
         - `upnp`
         - the entire `[fastsync]` section
         - the `[consensus]` section (v2 defaults are fine)
         - `index_tags` and `index_all_tags`
    3. v2 `priv_validator_state.json` must reflect v1 `priv_validator_state.json` and update the `round` from a string type to an int type. Also, `height` must be set to the agreed initial v2 height (`v1_halt_height + 1`)
    - v2 `addrbook.json` must match v1 `addrbook.json` (this is not mandatory but will help heimdall to peer faster)
    - v2 `node_key.json` must reflect v1 values (`priv_key.value`) and preserve v2 types
    - v2 `priv_validator_key.json` must reflect v1 values (`pub_key.value`, `priv_key.value` and `address`) and preserve v2 types
18. **Move New Genesis File** – Make sure the migrated genesis file is placed it in the correct directory.
19. **Reload daemon with** `sudo systemctl daemon-reload`
20. **Start Heimdall-v2 with** `sudo systemctl start heimdalld`
21. **Restart telemetry** (if needed) with `sudo systemctl restart telemetry`
22. **Internal Monitoring**
23. **Optional: WebSocket for Bor–Heimdall comm** - Edit bor `config.toml` file by adding the following under the [heimdall] section:
    ```toml
    [heimdall]
    ws-address = "ws://localhost:26657/websocket"
    ```
24. **Restart bor** Only in case the step above was done.  
25. (Internally) Resolve all the [POST-MIGRATION] tasks in JIRA under heimdall-v2 epic   

### Rollback strategy to restore v1
We decided not to enforce a HF in heimdall-v1, to avoid issues with rolling back to previous versions if the migration doesn’t work out as planned.  
The only changes to the code will enforce the `halt_height`. This theoretically means we will be releasing one 'fallback' version with the updated `halt_height`, and in case something goes wrong, the node operators will only need to install the previous version (without `halt_height` changes) and restart heimdall. If something doesn’t work with this procedure, snapshot restore would be safe to execute. The HF will make things much more complicated. The `halt_height` param is also available in the `heimdall-config.toml` file, but we want to avoid such changes, hence we are enforcing it through the code.  
Also, with this approach, the `halt-height` can be simply postponed (even if heimdall already stopped because of it) by simply changing the hardcoded `halt_height` to a future block.  
In case of issues with v2, node operators can roll back to the previous version of heimdall-v1 by following these steps:
   1. Install heimdall-v1 “fallback” version (with postponed or removed `halt_height`)
   2. Restore backed up `heimdall`-v1 folder
   3. Make sure no `symlink` or service for heimdall is bound to v2
   4. Restart the node with v1 commands
   5. If v1 still creates problems, we have the opportunity to roll back to pre-`halt_height`.
   6. If the rollback doesn't work, snapshot restore is safe to execute and the ultimate fallback.
