# Containerized Migration

As a prerequisite, make sure you have all the prerequisites in place, as described in [Migration Checklist](../containerized/1-MIGRATION-CHECKLIST.md).  
If you are using a containerized version of Heimdall (e.g. `docker` or inside a `kubernetes` cluster),
an image will be available to pull once the pilot node migration is successful.
You'd need to back up the content of v1 `HEIMDALL_HOME` folder, for future reference,
Then, pull and install the v2 docker image from https://hub.docker.com/r/0xpolygon/heimdall-v2/tags
```bash
docker pull 0xpolygon/heimdall-v2:<VERSION>
```
# TODO remove this step if confirmed that the image will come with the default configs
Now install the default configs for Heimdall v2 by running the following command:
```bash
  heimdalld init <MONIKER> --chain-id <CHAIN_ID>
```
`<MONIKER>` is the name of your node and `<CHAIN_ID>` is the chain you are running Heimdall-v2 on
(e.g., `heimdallv2-80002` for `amoy`, and `heimdallv2-137` for `mainnet`)

Then apply your custom configuration to the `app.toml`, `config.toml` and `client.toml` files.
Now, download the `genesis.json` file from the GCP bucket URL (https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json),
which will be available once the pilot node migration is successful,  
and place it in the `HEIMDALL_HOME/config` directory.

The file is going to be pretty large, especially for mainnet, where it is expected to be around 4–5 GB.
Hence, please make sure you have enough disk space available, and you have a fast internet connection.
At this point, you can run the heimdall-v2 image.
