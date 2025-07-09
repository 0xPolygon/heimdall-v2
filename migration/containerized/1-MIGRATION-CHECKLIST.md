# Checklist for Heimdall v1 to v2 containerized migration

This checklist is for users running Heimdall nodes in Docker or Kubernetes containers, or any other environment using docker images.  
Adjustments are necessary due to volume mounts, ephemeral storage, container networking, etc...

## 1. Verify the environment
   - Identify the container runtime (`docker`, `containerd`, etc).
   - Identify the volume mount path for Heimdall `data`, `config`, and `bridge` (e.g., `-v /heimdall:/var/lib/heimdall`).
   - Make sure your system is equipped with `sha512sum` (to verify the checksum of the genesis file)
   - Heimdall v2 will use `go 1.24.x`, so ensure your environment supports it.
   - Make sure you're running the latest versions of `bor`/`erigon`, `heimdall` and `heimdallcli`:  
     * `bor v2.2.5` / `erigon v3.0.12`+
     * `heimdall v1.6.0`
     * `heimdallcli v1.6.0`

## 2. Validate Heimdall v1 Config Files

Verify that the files in `HEIMDALL_HOME/config` are present and correctly formatted.

## 3. Free Required Ports on the Host

Make sure you have the following ports free on the host machine, so that heimdall-v2 can use them.

| Port          | Use                       | Protocol | Direction        | Notes                                                                 |
|---------------|---------------------------|----------|------------------|-----------------------------------------------------------------------|
| 26656         | **P2P**                   | **TCP**  | Inbound/Outbound | CometBFT peer-to-peer gossiping of blocks and transactions.           |
| 26657         | **RPC (CometBFT)**        | **TCP**  | Inbound          | Public CometBFT RPC API (e.g. `/status`, `/broadcast_tx_async`, etc). |
| 26660 or 6060 | **pprof (Profiling)**     | **TCP**  | Inbound          | Go’s `net/http/pprof` for debugging/profiling.                        |
| 1317          | **REST (Cosmos-SDK API)** | **TCP**  | Inbound          | REST endpoint via gRPC-Gateway from Cosmos SDK.                       |
| 9090          | **gRPC (Cosmos)**         | **TCP**  | Inbound          | Protobuf-based gRPC queries against app state.                        |
| 9091          | **gRPC-Web**              | **TCP**  | Inbound          | gRPC-Web server for browser clients.                                  |

Port `26656` (P2P) must be open on the validator to the sentry IPs.  
This is how sentries connect to the validator for block propagation and voting.  
Port `26657` (RPC) is generally not exposed on the validator.  
However, if your architecture requires sentries to relay transactions via RPC to the validator,  
you can open it to trusted IPs.

  For example, you can check that with:
      ```bash
      sudo lsof -i -P -n | grep LISTEN
      ```
  Or
      ```bash
      sudo ss -tulpn
      ```

## 4. Memory Requirements 
Ensure your system has at least 20 GB of available RAM at the time of migration.

## 5. Disk Space Requirements
If you plan to back up v1 `HEIMDALL_HOME`,
ensure your system has at least 2× the current size of `HEIMDALL_HOME` in available disk space
(on the disk or external storage that you will use for the backup).

## 6. Internet Connectivity
Ensure a stable and fast internet connection.  
The migration will download the genesis file from a trusted source,
which may be around 4 GB in size for mainnet.

> The mainnet migration will occur on July 10th, between 2 and 5 PM UTC.  
> As soon as the halt height `24404500` is reached, the Polygon team will create the v2 genesis and distribute to the community.  
> The process should take around 30-45 min.  
> Once the genesis is available, you can start the migration.  
> Expect heimdall downtime of around 2h (bor will continue working).

**Reminder:** Please migrate the validators first, to ensure the stake is moved over to v2 as soon as possible, and avoid any potential issue with the new network.
