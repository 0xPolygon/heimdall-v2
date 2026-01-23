# Heimdall v2 Protocol Buffers

This directory contains Protocol Buffer definitions for Heimdall v2, the validator layer for Polygon's Proof-of-Stake network.

## Overview

The proto definitions are organized by module, with each module representing a core component of the Heimdall system:

- **bor**: Manages spans and block producer selection for the Bor (child) chain
- **checkpoint**: Handles checkpoint submissions and acknowledgments for state finality
- **clerk**: Manages event records from the root chain to the bor child chain (state syncs)
- **milestone**: Manages milestone submissions for fast finality
- **stake**: Validator management, staking, and validator set coordination
- **topup**: Validator fee topup/withdraw mechanism
- **chainmanager**: Chain configuration and contract addresses
- **sidetxs**: Side transaction handling via vote extensions
- **types**: Shared types used across modules

## Code Generation

This project uses [Buf](https://buf.build) for proto code generation and management.

### Generate Go Code

From the root directory, run:  
```bash
make proto-all
```

This generates:
- Go types with gogoproto extensions
- gRPC service definitions
- gRPC-Gateway REST endpoints
- Swagger/OpenAPI documentation

### Configuration Files

- `buf.yaml`: Buf configuration, dependencies, and linting rules
- `buf.lock`: Locked dependencies for reproducible builds
- `buf.gen.gogo.yaml`: Code generation config for Go/gRPC
- `buf.gen.pulsar.yaml`: Code generation config for the Pulsar client
- `buf.gen.swagger.yaml`: OpenAPI/Swagger generation config

## Dependencies

The protos depend on the following packages:
- `cosmos/gogo-proto`: Protobuf extensions for Go
- `cosmos/cosmos-proto`: Cosmos-specific proto annotations
- `cosmos/cosmos-sdk`: Cosmos SDK base types
- `googleapis/googleapis`: Google API annotations for REST endpoints
