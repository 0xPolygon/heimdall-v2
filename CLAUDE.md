# Heimdall Development Guide for AI Agents

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

This guide provides comprehensive instructions for AI agents working on the Heimdall codebase. It covers the architecture, development workflows, and critical guidelines for effective contributions.

## Project Overview

Heimdall is the **consensus client** of Polygon PoS, built on Cosmos SDK and CometBFT. It manages validator selection, checkpointing to Ethereum L1, and span/sprint coordination. **Bor** is the separate **execution client** that handles block production and transaction execution. Together they form the complete Polygon PoS stack.

Heimdall focuses on BFT consensus, cross-chain communication, and validator management.

## Architecture Overview

### Core Components

1. **Checkpoint** (`x/checkpoint/`): Multi-stage L1 checkpoint submission with vote extension verification
2. **Stake** (`x/stake/`): Validator staking, delegation, and slashing management
3. **Bor** (`x/bor/`): Producer set management and span configuration for Bor chain
4. **Milestone** (`x/milestone/`): Milestone tracking for Bor finality guarantees
5. **Clerk** (`x/clerk/`): Event listening and state sync record processing
6. **Topup** (`x/topup/`): Fee top-up operations for validators
7. **ChainManager** (`x/chainmanager/`): Chain configuration and contract address management
8. **Bridge** (`bridge/`): Cross-chain event listener and processor for L1/L2 communication
9. **SideTxs** (`sidetxs/`): Side transaction system for validator-verified external data
10. **App** (`app/`): Core application setup with ABCI++ handlers and module orchestration

### Key Design Principles

- **Cosmos SDK Patterns**: Standard module structure (keeper/types/client), dependency injection
- **ABCI++ Integration**: Vote extensions for side transactions, PrepareProposal for message inclusion
- **Cross-chain Safety**: Multi-signature verification for checkpoints, validator-attested state syncs
- **Go Idioms**: Explicit error handling, interfaces for testability, structured logging

## Development Workflow

### Essential Commands

1. **Build**: Build the heimdalld binary

   ```bash
   make build
   ```

2. **Lint**: Run golangci-lint

   ```bash
   make lint-deps && make lint
   ```

3. **Test**: Run tests with vulnerability check

   ```bash
   make test
   ```

4. **Proto**: Regenerate protobuf code (requires Docker)

   ```bash
   make proto-all
   ```

## Module Structure

Each module in `x/` follows standard Cosmos SDK layout:

```markdown
x/<module>/
├── keeper/       # State management and business logic
├── types/        # Messages, events, genesis, queries
├── client/       # CLI commands and query handlers
├── testutil/     # Mock interfaces and test setup
├── module.go     # Module registration
├── depinject.go  # Dependency injection config
└── README.md     # Module documentation
```

## Testing Guidelines

1. **Unit Tests**: Test individual functions

   ```bash
   go test -v ./path/to/package
   ```

## Common Pitfalls

1. **Proto Changes**: Always run `make proto-all` after modifying `.proto` files
2. **Keeper Dependencies**: Update `expected_keepers.go` when adding cross-module calls
3. **Vote Extensions**: Side tx results must be deterministic across all validators
4. **Bridge Events**: New event types need both listener and processor implementations
5. **State Changes**: Only modify state in keeper methods, never in ABCI handlers directly

## What to Avoid

1. **Large, sweeping changes**: Keep PRs focused and reviewable
2. **Mixing unrelated changes**: One logical change per PR
3. **Ignoring CI failures**: All checks must pass
4. **Skipping proto generation**: Proto/Go mismatch causes runtime panics

## When to Comment

### DO Comment

- **Non-obvious behavior or edge cases**
- **Cross-chain assumptions** that depend on L1/Bor state
- **Consensus-critical logic** where bugs affect network liveness
- **Vote extension handling** and determinism requirements
- **Why simpler alternatives don't work**

```go
// Checkpoint interval must match L1 contract config, otherwise submissions fail.
const CheckpointInterval = 256

// FetchValidatorSet at span start, not current block, to ensure
// all validators agree on the producer set for this span.
func (k Keeper) GetSpanValidators(ctx sdk.Context, spanID uint64) ([]Validator, error)

// ProcessCheckpoint must be deterministic - all validators must compute
// the same result from the same inputs, or consensus breaks.
func (k Keeper) ProcessCheckpoint(ctx sdk.Context, checkpoint *Checkpoint) error
```

### DON'T Comment

- **Self-explanatory code** - if the code is clear, don't add noise
- **Restating code in English** - `// increment counter` above `counter++`
- **Describing what changed** - that belongs in commit messages, not code

### The Test

#### "Will this make sense in 6 months?"

Before adding a comment, ask: Would someone reading just the current code (no PR, no git history) find this helpful?

## Debugging Tips

1. **Logging**: Use zerolog with appropriate levels

   ```go
   helper.Logger.Debug().Uint64("span", spanID).Msg("Processing span")
   ```

2. **Metrics**: Add prometheus metrics for monitoring

   ```go
   metrics.CheckpointCount.Inc()
   ```

3. **Bridge Debugging**: Check RabbitMQ queues for stuck events

   ```bash
   rabbitmqctl list_queues
   ```

## Commit Style

Prefix with module name: `x/checkpoint: fix vote extension validation`

## CI Requirements

- All tests pass (`make test`)
- Linting passes (`make lint`)
- Proto files in sync (`make proto-all`)

## Branch Strategy

- **develop** - Main development branch, PRs target here
- **main** - Stable release branch
