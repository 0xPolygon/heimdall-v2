# Checklist for Heimdall v1 to v2 containerized migration

This checklist is for users running Heimdall nodes in Docker or Kubernetes containers, or any other environment using docker images. 
Adjustments are necessary due to volume mounts, ephemeral storage, container networking, etc...

## 1. Verify the environment
   - Ensure you are running Heimdall in Docker or Kubernetes etc...
   - Identify the container runtime (`docker`, `containerd`, etc).
   - Identify the volume mount path for Heimdall data and config (e.g., `-v /heimdall:/var/lib/.heimdall`).
## 2. Prepare Backup
   - Back up the `HEIMDALL_HOME` (default `/var/lib/heimdall`), containing `config/` and `data/` folders outside the container. 
   - Example (Docker):
     ```bash
     docker cp <container_id>:/var/lib/heimdall /path/to/backup
     ```
## 3.  Stop Existing Heimdall v1 Containers
   - Gracefully shut down Heimdall v1, for docker:
     ```bash
     docker stop <container_id>
     ```
     for Kubernetes: 
     ```bash
     kubectl scale deployment heimdall --replicas=0
     ```
## 4. Free Required Ports on the Host
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
## 5. Memory Requirements 
Ensure your system has at least 20 GB of available RAM.

## 6. Disk Space Requirements
Ensure your system has at least 3× the current size of `HEIMDALL_HOME` in available disk space.

## 7. Internet Connectivity
Ensure a stable and fast internet connection.
The migration will download the genesis file from a trusted source,
which may be 4–5 GB in size for mainnet.
