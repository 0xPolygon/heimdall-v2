[//]: # (TODO HV2: https://polygon.atlassian.net/browse/POS-2757)

# cmd


The `cmd` package is responsible for starting the heimdall application and provides the CLI framework (based on [cobra](https://github.com/spf13/cobra)).

## heimdalld

`heimdalld` is the service that is responsible for starting the heimdall application and also to interact with the heimdall application.

Apart from some commands taken for the upstream, it has the following customised commands:
- `stake`: Stake pol tokens for your account.
- `approve`: Approve the tokens to stake.
- `init`: Initialize genesis config, priv-validator file, and p2p-node file.
- `start`: Run the full node.
- `rollback`: rollback Cosmos SDK and CometBFT state by one height.
- `show-private-key`: Print the account's private key.
- `verify-genesis`: Verify if the genesis matches.
- `create-testnet`: Initialize files for a Heimdall testnet.
