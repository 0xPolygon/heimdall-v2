# Checklist for Heimdall v1 to v2 containerized migration

This checklist is for users running Heimdall nodes in Docker or Kubernetes containers, or any other environment using docker images. 
Adjustments are necessary due to volume mounts, ephemeral storage, container networking, etc...

1. Verify the environment
   - Ensure you are running Heimdall in Docker or Kubernetes etc...
   - Identify the container runtime (`docker`, `containerd`, etc).
   - Identify the volume mount path for Heimdall data and config (e.g., `-v /heimdall:/root/.heimdall`).
2. Prepare Backup
   - Back up the `HEIMDALL_HOME`, containing `config/` and `data/` folders outside the container. 
   - Example (Docker):
     ```bash
     docker cp <container_id>:/root/.heimdall /path/to/backup
     ```
3. Stop Existing Containers
   - Gracefully shut down Heimdall v1, e.g., using:
     ```bash
     docker stop <container_id>
     ```
     or for Kubernetes: 
     ```bash
     kubectl scale deployment heimdall --replicas=0
     ```
4. Free Required Ports (on Host Machine)
    * 26656 (P2P)
    * 26657 (RPC)
    * 26660 or 6060 (pprof)
    * 1317 (REST)
    * 9090 (gRPC)
    * 9091 (gRPC-Web, optional)
  For example, you can check that with:
      ```bash
      sudo lsof -i -P -n | grep LISTEN
      ```
5. Make sure your system has at least 30 GB of available RAM
6. Make sure your system has at least 3x current size (in GB) of `HEIMDALL_HOME/data` available disk space.
7. Make sure you have a stable and fast internet connection, as the migration process will download the genesis file from a trusted source.
   The file is going to be pretty large, especially for mainnet, where it is expected to be around 4–5 GB.
