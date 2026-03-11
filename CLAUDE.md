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
├── keeper/ # State management and business logic
├── types/ # Messages, events, genesis, queries
├── client/ # CLI commands and query handlers
├── testutil/ # Mock interfaces and test setup
├── module.go # Module registration
├── depinject.go # Dependency injection config
└── README.md # Module documentation
```

## Testing Guidelines

1. **Unit Tests**: Test individual functions

   ```bash
   go test -v ./path/to/package
   ```

## Security Review

Security rules are in `.claude/rules/`. They load automatically based on which files are being edited:

- **`security.md`** -- always active: pre-commit checklist, Go security patterns, dependency checks
- **`consensus-critical.md`** -- `app/`, `sidetxs/`, `x/checkpoint/`, `x/milestone/`, `x/bor/`: determinism, vote extension integrity, tallying thresholds
- **`cross-chain.md`** -- `bridge/`, `helper/`, `contracts/`, `x/stake/`, `x/topup/`, `x/clerk/`, `x/chainmanager/`: L1 receipt validation, nonce replay, event log verification
- **`contract-interactions.md`** -- `helper/call.go`, `helper/tx.go`, `contracts/`, `x/bor/grpc/`, keepers, bridge processors: IContractCaller security, ABI encoding, tx construction, gRPC client
- **`p2p-and-networking.md`** -- `helper/config.go`, `bridge/listener/`, `bridge/broadcaster/`, `cmd/`, `packaging/templates/`: CometBFT P2P config, RPC endpoint security, bridge networking, RabbitMQ
- **`state-and-migration.md`** -- `migration/`, `types/`, `common/`, `proto/`, `x/*/types/`: state migration safety, proto compatibility, shared type integrity

### Security-Critical Areas (ranked by impact)

1. **ABCI++ handlers** (`app/abci.go`, `app/vote_ext_utils.go`) -- consensus break, chain halt
2. **Vote extension tallying** -- forged approvals, unauthorized state changes
3. **L1 receipt validation** (`helper/call.go`) -- fake validator joins, stolen funds
4. **Staking nonce checks** (`x/stake/keeper/side_msg_server.go`) -- replay attacks
5. **Checkpoint verification** (`x/checkpoint/keeper/side_msg_server.go`) -- forged checkpoints
6. **Bridge event processing** (`bridge/`) -- state sync corruption

### When to Trigger Security Review

- Any change to files matched by `consensus-critical.md` or `cross-chain.md`
- New RPC endpoints or API handlers
- Dependency updates (especially forked `cosmos-sdk`, `cometbft`, `bor`)
- Changes to validation logic, threshold calculations, or signature verification
- Any change touching `helper/call.go` (the L1 interface)

## Before Making Changes

1. **Identify impact**: What other modules or components depend on this code?
2. **Plan implementation**: Outline the approach before writing code
3. **Plan testing**: How will you verify correctness? What edge cases exist?
4. **Check for breaking changes**: Will this affect APIs, proto definitions, or stored state?
5. **Check security implications**: Does this touch consensus, cross-chain, or validator logic? See `.claude/rules/`

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

## Maintaining This File

Update CLAUDE.md when:

- Claude makes a mistake or wrong assumption → Add clarifying context
- New patterns or conventions are established → Document them
- Frequently asked questions arise → Add answers here

This file should evolve over time to capture project-specific knowledge that helps AI agents work more effectively.
