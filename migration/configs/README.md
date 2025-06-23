# Config changes required to migrate from heimdall v1 to v2

## IMPORTANT
[The migration script](../script/migrate.sh) will handle these changes automatically, by porting the minimum required key/values.  
You'd need to port your own custom changes to the new config files ONLY if you are NOT using the script to migrate (e.g., `docker` based infra).

## Config changes
In this folder, you can find some example configuration files for heimdall-v2.
Search for `TODO-MIGRATION` under `heimdall-v1/config` and `heimdall-v2/config` to find the minimal changes  
that need to be made to migrate from v1 to v2 config files.
In v1, these files are named `app.toml`, `config.toml` and `heimdall-config.toml`.
In v2, these files are named `app.toml`, `config.toml` and `client.toml`, where `app.toml` is the merge  
between `heimdall-config.toml` and `config.toml` from v1.
All the key/values not marked with `TODO-MIGRATION` can be skipped and default v2 values accepted.  

## Key changes
The types of the pub/priv key have changed from [/heimdall-v1/config/priv_validator_key.json](./heimdall-v1/config/priv_validator_key.json) to
[/heimdall-v2/config/priv_validator_key.json](./heimdall-v2/config/priv_validator_key.json). The change is:  
- from `tendermint/PubKeySecp256k1` and `tendermint/PrivKeySecp256k1` in v1
- to `cometbft/PubKeySecp256k1eth` and `cometbft/PrivKeySecp256k1eth` in v2
Keep the new v2 types and use your current values from v1, as they are still compatible with v2.

## State changes
The types of one key has changed from [/heimdall-v1/data/priv_validator_state.json](./heimdall-v1/data/priv_validator_state.json) to
[/heimdall-v2/data/priv_validator_state.json](./heimdall-v2/data/priv_validator_state.json).  
Copy the v2's [/heimdall-v2/data/priv_validator_state.json](./heimdall-v2/data/priv_validator_state.json) and place it under `HEIMDALL_HOME/data/priv_validator_state.json`
