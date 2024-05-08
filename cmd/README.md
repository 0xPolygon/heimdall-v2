# CMD

The `cmd` package is responsible for starting the heimdall application provides the CLI framework (based on [cobra](https://github.com/spf13/cobra)).

## heimdallcli

`heimdallcli` is the command line interface to interact with the heimdall application.

Apart from some command commands taken for the upstream, it has 2 customised commands:
- `stake`: Stake matic tokens for your account.
- `approve`: Approve the tokens to stake.

## heimdalld

`heimdalld` is the service that is responsible for starting the heimdall application.

Following are the available commands for heimdalld:
- `init`: Initialize genesis config, priv-validator file, and p2p-node file.
- `start`: Run the full node.
- `rollback`: rollback Cosmos SDK and CometBFT state by one height.
- `show-privatekey`: Print the account's private key.
- `verify-genesis`: Verify if the genesis matches.
- `create-testnet`: Initialize files for a Heimdall testnet.
