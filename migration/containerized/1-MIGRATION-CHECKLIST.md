# Checklist for Heimdall v1 to v2 containerized migration

This checklist is for users running Heimdall nodes in Docker or Kubernetes containers, or any other environment using docker images. 
Adjustments are necessary due to volume mounts, ephemeral storage, container networking, etc...

## 1. Verify the environment
   - Ensure you are running Heimdall in Docker or Kubernetes etc...
   - Identify the container runtime (`docker`, `containerd`, etc).
   - Identify the volume mount path for Heimdall data and config (e.g., `-v /heimdall:/var/lib/heimdall`).
   - Make sure your system is equipped with `sha512sum` (to verify the checksum of the genesis file)

## 2. Validate Heimdall v1 Config Files

Verify that the files in `HEIMDALL_HOME/config` are present and correctly formatted.

## 3. Free Required Ports on the Host
   - Make sure you have the following ports free on the host machine, so that heimdall-v2 can use them.
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
## 4. Memory Requirements 
Ensure your system has at least 20 GB of available RAM at the time of migration.

## 5. Disk Space Requirements
Ensure your system has at least 2× the current size of `HEIMDALL_HOME` in available disk space.

## 6. Internet Connectivity
Ensure a stable and fast internet connection.
The migration will download the genesis file from a trusted source,
which may be around 4 GB in size for mainnet.
