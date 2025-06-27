# Checklist for Heimdall v1 to v2 automated migration through the script

1. Make sure your platform is supported by the migration script.

   | OS     | Arch    | Package Manager | Supported | Notes                   |
   |--------|---------|-----------------|-----------|-------------------------|
   | Linux  | x86_64  | `dpkg` (Debian) | ✅         | Uses `.deb` package     |
   | Linux  | aarch64 | `dpkg`          | ✅         | Uses ARM `.deb` package |

2. Make sure the following tools are installed on your system.  
   The migration script will anyway fail early if such tools are not installed.

    | Tool        | Purpose               | Install Command (Ubuntu/Debian) |
    |-------------|-----------------------|---------------------------------|
    | `curl`      | Downloading binaries  | `sudo apt install curl`         |
    | `tar`       | Extracting archives   | `sudo apt install tar`          |
    | `jq`        | JSON manipulation     | `sudo apt install jq`           |
    | `sha512sum` | File integrity checks | `sudo apt install coreutils`    |
    | `file`      | File type detection   | `sudo apt install file`         |
    | `awk`       | Text processing       | `sudo apt install gawk`         |
    | `sed`       | Stream editing        | `sudo apt install sed`          |
    | `systemctl` | Service management    | Pre-installed on most distros   |
    | `grep`      | Text searching        | Pre-installed on most distros   |
    | `id`        | User information      | Pre-installed on most distros   |

3. Make sure your system has `bash` installed, as the script uses `bash` features (`sh` won't work)
4. Make sure the v1 configs files under `HEIMDALL_HOME/config` are correct and properly formatted. 
5. Make sure your system has at least 30 GB of available RAM
6. Make sure your system has at least 3x current size (in GB) of `HEIMDALL_HOME/data` available disk space. 
   This is needed for v1 backup, which can be deleted later on (some weeks after the migration is successful).
7. Make sure you have a stable and fast internet connection, as the migration process will download the genesis file from a trusted source.
   The file is going to be pretty large, especially for mainnet, where it is expected to be around 4–5 GB.
8. Make sure you don't have any other processes running on ports that Heimdall v2 will use. 
   In case you do, you will need to kill such processes (and potentially migrate to other ports) before running the migration script.
   Note that v2 uses more ports than v1, so you might need to check that you don't have any other processes running on the following ports:

   | Port  | Protocol  | Description                                                                     |
   |-------|-----------|---------------------------------------------------------------------------------|
   | 26656 | TCP       | P2P communication between CometBFT nodes (gossiping blocks, votes, txs).        |
   | 26657 | HTTP      | RPC server (CometBFT): for `/status`, `/broadcast_tx_sync`, `/abci_query`, etc. |
   | 26660 | HTTP      | CometBFT pprof profiling endpoint (if enabled).                                 |
   | 6060  | HTTP      | Alternative pprof port (Go default) for performance profiling.                  |
   | 1317  | HTTP      | REST API (gRPC-Gateway) — Cosmos SDK module HTTP interface.                     |
   | 9090  | gRPC      | gRPC server (Cosmos SDK) — query state, txs, and custom services.               |
   | 9091  | gRPC      | gRPC-web server (optional, for browser clients).                                |

   You can check that with `sudo netstat -tuln | grep LISTEN` or `sudo ss -tuln` or `sudo lsof -i -P -n | grep LISTEN` commands.
9. Make sure the user running your current Heimdall v1 exists on the machine.
   You can check that with `systemctl status heimdalld` and inspect the `User=` field and confirm it's correct by 
   checking the user currently running the process (e.g., with `ps -o user= -C heimdalld`
10. Retrieve the parameters needed by the script and store them somewhere in a file

     | Flag                 | Description                                                                                                    |
     |----------------------|----------------------------------------------------------------------------------------------------------------|
     | `--heimdall-v1-home` | Path to Heimdall v1 home (must contain `config` and `data`)                                                    |
     | `--heimdallcli-path` | Path to `heimdallcli` (must be latest stable version). It can be retrieved with `which heimdallcli`            |
     | `--heimdalld-path`   | Path to `heimdalld` (must be latest stable version). It can be retrieved with `which heimdalld`                |
     | `--network`          | `mainnet` or `amoy`                                                                                            |
     | `--node-type`        | `sentry` or `validator`                                                                                        |
     | `--service-user`     | System user running Heimdall (e.g., `heimdall`).                                                               |
     |                      | Check with: `systemctl status heimdalld` and inspect the `User=` field.                                        |
     |                      | Confirm it's correct by checking the user currently running the process (e.g., with `ps -o user= -C heimdalld` |
     |                      | This is critical to avoid permission issues in v2!                                                             |
     | `--generate-genesis` | Whether to generate genesis using `heimdalld` (recommended: `false` because the export is a heavy process).    |
     |                      | Note that this value will be anyway overwritten by the script.                                                 |
     |                      | This happens when the node was not able to commit to the latest block's heigh needed for the migration,        |
     |                      | hence generate-genesis will be set to `false` and the `genesis.json` file downloaded from trusted source.      |

11. Once you know all the details, you can prepare in advance the command to run.
    Remember to use `sudo` and `bash`, e.g.:
     ```bash
     sudo bash migrate.sh \
       --heimdall-v1-home=/var/lib/heimdall \
       --heimdallcli-path=/usr/bin/heimdallcli \
       --heimdalld-path=/usr/bin/heimdalld \
       --network=mainnet \
       --node-type=validator \
       --service-user=heimdall \
       --generate-genesis=false
     ```
