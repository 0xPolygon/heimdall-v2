# Containerized Migration

As a prerequisite, make sure you have all the prerequisites in place, as described in [Migration Checklist](../containerized/1-MIGRATION-CHECKLIST.md).  
If you are using a containerized version of Heimdall (e.g. `docker` or inside a `kubernetes` cluster),
an image will be available to pull once the pilot node migration is successful.
You'd need to back up the content of v1 `HEIMDALL_HOME` folder, for future reference,
Then, pull and install the v2 docker image from https://hub.docker.com/r/0xpolygon/heimdall-v2/tags
```bash
docker pull 0xpolygon/heimdall-v2:<VERSION>
```
Where `VERSION` is for example `0.2.4`

# TODO remove the following step once configs are embedded in the image
Now install the default configs for Heimdall v2 by running the following command:
```bash
  heimdalld init <MONIKER> --chain-id <CHAIN_ID>
```
Where `MONIKER` is the name of your node (any string) and `CHAIN_ID` is the chain you are running Heimdall-v2 on
(e.g., `heimdallv2-80002` for `amoy`, and `heimdallv2-137` for `mainnet`)

Then apply your custom configuration to the `app.toml`, `config.toml` and `client.toml` files.
Then, you can customize the configs under `HEIMDALL_HOME/config` (`app.toml`, `client.toml`, `config.toml`),
based on your setup.
Templates for each supported network are available [here](https://github.com/0xPolygon/heimdall-v2/tree/develop/packaging/templates/config)

Now, download the `genesis.json` file from the GCP bucket URL.  
The URL is [this for mainnet](https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json),  
and [this for amoy](https://storage.googleapis.com/amoy-heimdallv2-genesis/migrated_dump-genesis.json).
Mainnet one will be available once the pilot node migration is successful.  
Name such genesis file `genesis.json` and place it in the `HEIMDALL_HOME/config` directory.

The file is going to be pretty large, especially for mainnet, where it is expected to be around 4–5 GB.
Hence, please make sure you have enough disk space available, and you have a fast internet connection.
At this point, you can run the heimdall-v2 image, with a command like the following:
```bash
docker run -d --name heimdall-v2 \
  -v "$HEIMDALL_HOME:/var/lib/heimdall \
  -p 26656:26656 -p 26657:26657 -p 1317:1317 \
  0xpolygon/heimdall-v2:<VERSION> \
  start
```
Make sure to customize the flags based on your setup.  
