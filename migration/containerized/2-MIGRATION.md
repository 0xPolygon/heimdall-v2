# Containerized Migration

Ensure you have completed all prerequisites
outlined in the [Migration Checklist](../containerized/1-MIGRATION-CHECKLIST.md).

If you are using Heimdall in a containerized setup (e.g., Docker or a Kubernetes cluster),
a v2 container image will be made available after the pilot node migration is confirmed successful.
Before proceeding, back up your v1 `HEIMDALL_HOME` directory for future reference.

## Step 1: Pull the Heimdall v2 Image

Download the v2 Docker image from Docker Hub:

```bash
docker pull 0xpolygon/heimdall-v2:<VERSION>
````

Replace `<VERSION>` with the target version tag (e.g., `0.2.4`).

---


**TODO** Remove this step once configuration files are embedded in the Docker image.

## Step 2: Initialize Default Configuration

Run the following command to generate default config files:

```bash
heimdalld init <MONIKER> --chain-id <CHAIN_ID>
```

* `<MONIKER>` is your node name (any string).
* `<CHAIN_ID>` depends on your network:

    * `heimdallv2-80002` for Amoy
    * `heimdallv2-137` for Mainnet

After initialization, customize the following files under `HEIMDALL_HOME/config`:

* `app.toml`
* `config.toml`
* `client.toml`

Templates for each supported network are available in the [Heimdall v2 GitHub repository](https://github.com/0xPolygon/heimdall-v2/tree/develop/packaging/templates/config).

---

## Step 3: Download the Genesis File

Download the appropriate `genesis.json` from the following GCP bucket:

* [Mainnet genesis](https://storage.googleapis.com/mainnet-heimdallv2-genesis/migrated_dump-genesis.json) *(available after pilot migration)*

Save the file as `genesis.json` and place it in your `HEIMDALL_HOME/config` directory.

**Note:** The genesis file is large (4–5 GB on Mainnet). 
Ensure you have sufficient disk space and a reliable, fast internet connection.

---

## Step 4: Start the Heimdall v2 Container

Run the container with the appropriate configuration. Example:

```bash
docker run -d --name heimdall-v2 \
  -v "$HEIMDALL_HOME:/var/lib/heimdall" \
  -p 26656:26656 -p 26657:26657 -p 1317:1317 \
  0xpolygon/heimdall-v2:<VERSION> \
  start
```

Adjust the `-v` and `-p` options based on your deployment needs.

---

## Final Notes

* Verify that all ports are correctly mapped and not in use.
* Ensure sufficient system memory and disk availability before running the container.
* Monitor logs to confirm successful startup.
