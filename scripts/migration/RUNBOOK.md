# Process

### Overview

The heimdall-v2 deployment is a critical upgrade to the Polygon PoS stack, replacing heimdall with a new chain based on cometBFT `v0.38.x` and cosmos-sdk `v0.50.x`. This transition requires a structured and coordinated approach across multiple teams, ensuring minimal downtime and a seamless migration for node operators.

### Deployment Phases

The deployment consists of two main phases:

**1. Development Phase**

Finalizing code, testing, auditing, and preparing necessary documentation and tools.
All tools require internal approval.

**2. Operational Phase**

Executing the migration across different networks (Devnets, Mumbai, Amoy, and Mainnet), with coordinated steps involving PoS, DevOps, DevTools and Validators teams.

### Teams Involved

- **PoS Team**: Development, testing, release management, and operational coordination.
- **Governance Team**: Community communication, mainly through PIPs and PPGCs.
- **DevOps Team**: Infrastructure setup, monitoring, and migration execution on internal nodes, in collaboration with PoS team.
- **DevTools Team**: Support in tooling, configuration, and testing, in collaboration with PoS team.
- **Validators Team**: Coordination with node operators, ensuring correct upgrades and troubleshooting, in collaboration with PoS team.
- **Informal Systems**: External security audit and help in troubleshooting during migration execution, in collaboration with PoS team.
- **Documentation Team**: Updating API references and migration guides for third-party integrations, in collaboration with PoS team.

### Development Phase

### Code Completion

Development of Heimdall-v2 has been finalized. Some changes might be required due to audit report. The audit started on March 17th, 2025, and it’s expected to be completed by end of April.

### End-to-End Tests

Initiated towards the end of the development phase and continuing alongside other activities. Testing is considered complete once the migration steps are successfully executed in the Mumbai environment.

### Matic-CLI configuration:

```bash
BOR_REPO="<https://github.com/maticnetwork/bor.git>"
BOR_BRANCH=avalkov/bor-without-heimdall
HEIMDALL_REPO="<https://github.com/0xPolygon/heimdall-v2.git>"
HEIMDALL_BRANCH=develop
CONTRACTS_REPO="<https://github.com/0xPolygon/pos-contracts.git>"
CONTRACTS_BRANCH=anvil-pos
GENESIS_CONTRACTS_REPO="<https://github.com/maticnetwork/genesis-contracts.git>"
GENESIS_CONTRACTS_BRANCH=master
MATIC_CLI_REPO="<https://github.com/maticnetwork/matic-cli.git>"
MATIC_CLI_BRANCH=heimdall-v2

```

### Final tags for v2 audit:

- Heimdall-v2: https://github.com/0xPolygon/heimdall-v2/releases/tag/v0.1.8
- Cosmos-sdk: https://github.com/0xPolygon/cosmos-sdk/releases/tag/v0.1.16-beta-polygon
- CometBFT: https://github.com/0xPolygon/cometbft/releases/tag/v0.1.4-beta-polygon
- Bor (for reference and testing): https://github.com/maticnetwork/bor/releases/tag/v2.0.0-v2
- Matic-cli (for reference and testing): https://github.com/maticnetwork/matic-cli/releases/tag/v1.0.0-v2

### Upcoming Milestones:

1. **External Audit** - Performed by Informal Systems.
2. **Documentation and API Updates** - Ensuring all third-party applications relying on Heimdall are prepared.
3. **Release Bor “Apocalypse” Version** – Implement changes to Bor for backward compatibility with both Heimdall-v1 and v2. Additionally, we need to address the span production issue to ensure Bor can operate without Heimdall, implementing a workaround for this dependency.
4. **Release Erigon “Apocalypse” Version** – To be implemented by Erigon team based on Bor template
5. **Implement a Runbook and Tools to help node operators with migration** – A set of scripts and documents
6. **Internal Approval on Runbook, scripts and PIPs** - Includes migration runbook and rollback plan.
7. **PIP and Tools Finalization and Publication** - Community-facing document providing upgrade details and node operator instructions.
8. **Audit Fixes Implementation** - Time buffer to address any security findings.
9. **Final Stable Release of Heimdall-v2** - Ensuring compatibility across different architectures.
10. **Release Heimdall-v1 “Apocalypse” Version** – This version will hardcode the final halt_height in the v1 code. All the node operators will need to upgrade to this version before the migration process starts. The halt_height must be decided upwards.
    - Temp branch on v1 (for testing on devnets): mardizzone/apocalypse

A roadmap / timeline can be found [here](https://www.notion.so/PoS-Roadmap-d90e12322c094fa4917d6ac8bd7d4b38?pvs=21)

**Open Points (TBD):**

- Re-enabled mumbai in the code for testing?
- The runbook and the script need to be reviewed and audited (at least internally)
- Defining seeds for heimdall-v2 in the code.
- Handling of genesis files in the repository (shall we keep v1 or provide migrated v2 versions – as this is going to be available only after the migrate command is executed).
- Security considerations for genesis json file dump
- Coordination or eventual changes to the current process due to business decisions related to the rollout of other components linked to heimdall-v2 (see note at the beginning of the page)
- Update the script with bor config changes required
- How to manage the live communication about the migration (e.g. sharing checksums, script, updates, genesis files, etc.)  

### Operational Phase

The operational phase involves identical migration steps across **Devnets, Mumbai, Amoy, and Mainnet**. We could even plan a “**Simulated migration”**, to encourage operators to join, on an ad hoc testnet (see [this convo](https://0xpolygon.slack.com/archives/C05F2JJEQF5/p1714056156344929) and its [relative task](https://polygon.atlassian.net/browse/POS-2572)).

Each migration will be executed following the following structured runbook.

# Migration Runbook

### Preparation

1. DevOps preserve some rpc nodes for heimdall-v1 historical data (this can be done in advance)
2. **Governance Proposal Submission.** Includes:
    1. agreed `halt_height`
    2. commit hash (or release version) of Heimdall-v2
    3. old and new chain IDs
        1. **TBD**: Is defining the new chain ID a risk? Someone could spin up a network with the same chain ID before we launch heimdall-v2. What happens in that case. Maybe -as we are planning to do with GENESIS_TIME - it would make sense to define in the script at the last moment, before running it
    4. genesis time
        1. **TBD**: We might have some issues with the genesis time being in the future (v2 won’t start until then). Maybe we just set this time at around `halt_height` time (needs some calculation and be defined in the script at the last moment, before running it). Anyway, also consulted with Informal, better to have this in the future than in the past, so maybe we might want to have 1-2h buffer for migration and then start
    5. abort procedure.
        1. We decided not to go enforce a HF in heimdall-v1, to avoid issues with rolling back to previous versions if the migration doesn’t work out as planned.
        2. The only changes to the code will enforce the halt_height (see mardizzone/apocalypse branch on v1)
        3. This theoretically means we will be releasing one version with the updated halt_height (for the migration), and in case something goes wrong, the node operators will only need to install the previous version (without halt_height changes) and restart heimdall. If something doesn’t work with this procedure, snapshot restore would be safe to execute. The HF will make things much more complicayed.The halt_height param is also available in the heimdall-config.toml file, but we want to avoid such changes, hence we are enforcing it through the code. The migration script will anyway handle the rollback plan (for the node operators using such script…)
        4. Also, with this approach, the halt-height can be simply postponed (even if heimdall already stopped because of it) by simply changing the hardcoded haltHeight to a future block.
    - **Reference Example:** [Stargate Proposal](https://www.mintscan.io/cosmos/proposals/37)
3. **Final Bor "Apocalypse" Release**
    1. A version that supports both Heimdall-v1 and v2, and work around the dependency of spans proposals, which would prevent bor to keep progressing without heimdall.
4. **Final Erigon Release**
    1. A version that supports both Heimdall v1 and v2
5. **Final Heimdall "Apocalypse" Release**
    - Enforces the agreed `halt_height`
6. **Node Operators Update Bor/Erigon** - Coordinated by the Validators Team.
7. **Node Operators Update Heimdall** v1 – Coordinated by the Validators Team
8. **Confirmation of successful upgrade from all node operators**
9. **War Room Setup**
    - Establish a real-time coordination hub for migration execution.

### Execution

It's recommended to run the process first on a pilot node, better to start with a sentry then with a low-staked validator, for amoy and mainnet.  
Then the exported `genesis.json` can be published together with the resulting checksum.  
Same for the migrated `genesis.json` checksum.    
All the steps below (except for point 6) are automated in the [migration script](migration.sh), the following is just a runbook for manual execution.    

1. Node operators confirm to be on latest Bor/Erigon version (compatible with v1 and v2)
2. Node operators confirm to be on latest Heimdall-v1 version (with `halt_height` embedded)
3. Node operators confirm to be on `heimdallcli` version greater than `v1.0.10`
4. Node operators confirm they can execute `heimdalld` and `heimdallcli` by running `version` command for both of them
5. Node operators confirm heimdall-v1 is down due to hitting `halt_height`
6. Contracts team updates `RootChain` contract on L1 via method `RootChain.setHeimdallId` with the chainId previously agreed (since this is not used, can be done in advance or after the migration)  
7. **Genesis Export from Heimdall-v1** - Performed by PoS & DevOps teams using
   `heimdallcli export-heimdall --home HEIMDALLD_HOME`
    - operators should use version of `heimdallcli` newer than `v1.0.10`
    - `HEIMDALLD_HOME` is usually `/var/lib/heimdall` directory, and the command will generate `dump-genesis.json` there  
   The first export will be made available to the community so that the script can use that to download the file (if the node operators decide not to generate their own genesis)  
   **TBD**: comms channel for this?  
8. **Checksum Generation** – DevOps/DevTools generates a checksum of the exported genesis file and distributes it to the community via Gov team or a call.  
   - **TBD**: comms channel for this?  
   - **TBD**: Do we need to take any security measure here? Shall the file be signed? What if a middle man intercepts the published genesis json file and alter it or its checksum?
9. **Checksum Validation -** Ensuring all node operators get the same checksum from their genesis exports.
10. **Backup Heimdall v1 Data** - Node operators back up their Heimdall v1 `HEIMDALL_HOME` directory (containing `config` , `data` and – for validators - `bridge`).
11. **Install Heimdall-v2** – Node operators install heimdall-v2 ****with the install script available at https://github.com/maticnetwork/install/blob/heimdall-v2/heimdall-v2.sh
12. **Verify Heimdall-v2 installation** – Node operators make sure heimdall-v2 is successfully installed by running the `version` command
13. **Run Migration Command** - Converts the old genesis to v2 format using:
    ```bash
    heimdalld migrate dump-genesis.json --chain-id=<CHAIN_ID> --genesis-time=<TIME_IN_FORMAT_YYYY-MM-DDTHH:MM:SSZ) --initial-height=<H>
    ```
    where `H = v1_halt_height + 1` and `genesis-time` is pre-agreed offline among all validator operators and is the time on which the new network will start to produce blocks, and it’s the result of the governance proposal at point 1.
14. **Checksum Generation** – DevOps/DevTools generates a checksum of the migrated genesis file and distributes it to the community via Gov team or a call.
    - **TBD**: Do we need to take any security measure here? Shall the file be signed? What if a middle man intercepts the published genesis json file and alter it or its checksum?  
    - **TBD**: comms channel for this?
15. **Checksum Validation -** Ensuring all node operators get the same checksum from their migrated genesis.
16. Node operators delete (or rename) their `HEIMDALL_HOME` directory (**make sure it was backed up earlier**)
17. Node operators create a new `HEIMDALL_HOME` directory with the new Heimdall v2 binary by running `heimdalld init [moniker-name]` . Rhe moniker is also available in v1 at `HEIMDALL_HOME/config/config.toml` and it’s recommended to keep the same value.
    The service file `heimdall.service` should reflect the same `User` (and possibly `Group`) coming from v1 service file.
18. Node operators edit their configuration files with config information from their backup, plus default values.
     1. v2 `app.toml` must reflect the “merge” from v1 `app.toml` and `heimdall-config.toml`
         - All values from v1 `app.toml` should NOT be carried over."
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
     3. v2 `priv_validator_state.json` must reflect v1 `priv_validator_state.json` and update the `round` from a string type to int type. Also, `height` must be set to the agreed initial v2 height (`v1_halt_height + 1`)
     - v2 `addrbook.json` must match v1 `addrbook.json` (this is not mandatory but will help heimdall to peer faster)
     - v2 `node_key.json` must reflect v1 values (`priv_key.value`) and preserve v2 types
     - v2 `priv_validator_key.json` must reflect v1 values (`pub_key.value`, `priv_key.value` and `address`) and preserve v2 types
19. **Move New Genesis File** – Make sure the migrated genesis file is placed it in the correct directory.
20. **Repetition – All the steps up until now can be repeated as many times as possible, on different testnets**, until node operators are comfortable with the operational procedure.
21. **Reload daemon with** `sudo systemctl daemon-reload`
22. **Start Heimdall-v2 with** `sudo systemctl start heimdalld`
23. **Restart telemetry** (if needed) with `sudo systemctl restart telemetry`
24. **Internal Monitoring** - DevOps team monitors the network for any issues.  
25. **v2 Dashboard** - See https://polygon.atlassian.net/browse/POS-2915 to enable the v2 dashboard on DataDog.  

### Challenges & Considerations

1. **Troubleshooting & Feedback Collection** – If operators report any issues, PoS team, in collaboration with DevOps, Validators team and – if needed – Informal Systems team, help during troubleshooting.
2. **State Sync & Checkpoints Validation** - Ensuring all the pieces work as expected on the PoS network, by making sure checkpoints and state syncs are going through.
3. Checks for lists of upgrade nodes, possibly check metrics and ensure the network is stable
4. **Validator Coordination:** Some node operators may be slow to upgrade, requiring proactive engagement.
5. **Rollback Strategy:** Ensuring a well-defined recovery process in case of failures.
   1. Remove `halt_height` configuration
   2. Restore backed up `heimdall`-v1 folder
   3. Make sure no `symlink` or service for heimdall is bound to v2
   4. Restart the node with v1 commands
   5. If v1 still creates problems, we have the opportunity to rollback to pre-haltHeight heights.
   6. If the rollback doesn't work, snapshot restore is safe to execute and the ultimate fallback (see [this](https://github.com/maticnetwork/heimdall/issues/1238#issuecomment-2656503681))
6. **Security & Integrity:** Maintaining correct state transition across different networks.
7. **Global Synchronization:** Scheduling the halt height and the v2 migration (including the war room) when most validators and involved teams are online. Also, do not schedule across weekend (Tuesday would be ideal).
8. **Comms**: We need to define a live and reliable channel where comms are maintained (e.g. to share checksums, script, updates, genesis files, etc.). Maybe this could be done via the v2 repo itself.  
9. **Removing heimdall-v1 backups:** After a successful migration, it is recommended to keep the old v1 backup for some weeks, then safe to deleted
10. **Security of the script itself**: To avoid any tampering, the script itself needs to be signed and checksummed before distribution. The checksum can be shared in the same channel where the genesis files are shared.  
