# heimdall-v2

Consensus client of the Polygon PoS chain,
using forks of [cometBFT](https://github.com/0xPolygon/cometBFT) and [cosmos-sdk](https://github.com/0xPolygon/cosmos-sdk).

## Pre-requisites

Make sure you have go1.24+ already installed

## Build
```bash 
$ make build
```
This will produce the binary `heimdalld` in the `build` directory.

## Initialize heimdall
```bash 
$ heimdalld init <MONIKER_NAME> --chain-id=<NETWORK_NAME>
```
You can skip `--chain-id` flag if you want to run this locally (it will default to `heimdall-local`).
Otherwise, use `heimdallv2-80002` for `amoy`, or `heimdallv2-137` for `mainnet`.
`<MONIKER_NAME>` is the name of your node, which can be any string you like.

This command will generate some folders and files in the heimdall home directory (default /var/lib/heimdall).

## Run heimdall
```bash 
$ heimdalld start
```

## How to use the keyring

Instructions on how to import your validator private key into the keyring and use it to sign transactions.

Get your `base64` encoded private key from:  
`cat /var/lib/heimdall/config/priv_validator_key.json`

Convert the `base64` encoded key to the hex encoded key:  
`echo "<PRIVATE_KEY_BASE64_ENCODED>" | base64 -d | xxd -p -c 256`

Import the `hex` encoded key to your keyring:  
`heimdalld keys import-hex <KEY_NAME> <PRIVATE_KEY_HEX_ENCODED> --home <HOME_DIR_PATH>`

When you first import a key into the keyring, you will be prompted for a password, which will be used every time you sign a transaction.

When running a `tx` command, specify the `--from` argument, by using the name of the key you have set above. Example:  
`heimdalld tx gov vote 1 yes --from <KEY_NAME>`
