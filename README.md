# heimdall-v2

Consensus client of Polygon PoS chain, using a forks of [cometBFT](https://github.com/0xPolygon/cometBFT) and [cosmos-sdk](https://github.com/0xPolygon/cosmos-sdk).

## Pre-requisites

Make sure you have go1.24+ already installed

## Build
```bash 
$ make build
```
This will produce the binary `heimdalld` in the `build` directory.

## Initialize heimdall
```bash 
$ heimdalld init --moniker=<NODE_NAME> --chain=<NETWORK_NAME>
```

## Run heimdall
```bash 
$ heimdalld start
```

## How to use keyring

Instructions on how to import your validator private key into the keyring and use it to sign transactions.

Get your `base64` encoded private key from:  
`cat /var/lib/heimdall/config/priv_validator_key.json`

Convert the `base64` encoded key to hex encoded key:  
`echo "<PRIVATE_KEY_BASE64_ENCODED>" | base64 -d | xxd -p -c 256`

Import the `hex` encoded key to your keyring:  
`heimdalld keys import-hex <KEY_NAME> <PRIVATE_KEY_HEX_ENCODED> --home <HOME_DIR_PATH>`

When you first import a key into the keyring, you will be prompted for a password, which will be used every time you sign a transaction.

When running a `tx` command, just specify the `--from` argument, by using the name of the key you have set above. Example:  
`heimdalld tx gov vote 1 yes --from <KEY_NAME>`
