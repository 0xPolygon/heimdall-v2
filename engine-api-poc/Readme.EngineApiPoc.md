

# Engine API Proof of Concept (POC)

This repository contains a Proof of Concept implementation of an Engine API using Docker Compose. The following commands are defined in the Makefile and **require sudo privileges** to run. For example, to destroy the current setup, run:

```bash
sudo make engine-api-poc-destroy
```

## Makefile Commands

### 1. Build

**Command:**

```bash
sudo make engine-api-poc-build
```

**Description:**

- **Purpose:** Builds the Docker images for `heimdalld-init` and `erigon-init` using the Compose file located at `./engine-api-poc/docker-compose.yaml`.
- **Usage Note:** This command should be run **only once** to create the images. Rebuilds are slow, so only run this if there have been changes to the project.

---

### 2. Init

**Command:**

```bash
sudo make engine-api-poc-init
```

**Description:**

- **Purpose:** Sets up the initial configuration files by running the `heimdalld-init` and `erigon-init` containers.
- **Usage Note:** Run this command after building the images to configure your environment.

---

### 3. Start

**Command:**

```bash
sudo make engine-api-poc-start
```

**Description:**

- **Purpose:** Starts the nodes (`node0`, `node1`, `node2`, `node3`, and `node4`) in detached mode.
- **Usage Note:** This command brings your Engine API POC environment online.

---

### 4. Stop

**Command:**

```bash
sudo make engine-api-poc-stop
```

**Description:**

- **Purpose:** Stops all running nodes and services defined in the Compose file.
- **Usage Note:** Use this command to gracefully stop the running containers.

---

### 5. Destroy

**Command:**

```bash
sudo make engine-api-poc-destroy
```

**Description:**

- **Purpose:** 
  - Stops and removes all containers using `docker compose down` with the `--remove-orphans` flag.
  - Deletes the build directory (`./engine-api-poc/build`), cleaning up all configuration and build artifacts.
- **Usage Note:** This command cleans up the entire setup. Use it when you want to completely remove all resources.

---

## Workflow Summary

1. **Build the Images**  
   *(Run only once or when project changes occur)*

   ```bash
   sudo make engine-api-poc-build
   ```

2. **Initialize Configuration**  
   *(Sets up config files)*

   ```bash
   sudo make engine-api-poc-init
   ```

3. **Start the Nodes**  
   *(Brings the services online)*

   ```bash
   sudo make engine-api-poc-start
   ```

4. **Stop the Nodes**  
   *(Gracefully stops the running services)*

   ```bash
   sudo make engine-api-poc-stop
   ```

5. **Destroy the Environment**  
   *(Cleans up containers and build artifacts)*

   ```bash
   sudo make engine-api-poc-destroy
   ```


## Troubleshooting

- **Permission Issues:**  
  All commands require sudo. If you face permission errors, confirm that you are using `sudo`.

- **Configuration Issues:**  
  If the initialization does not set up configuration files correctly, re-run the `engine-api-poc-init` command.
