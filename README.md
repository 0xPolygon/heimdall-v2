# heimdall-v2

[//]: # (TODO HV2: https://polygon.atlassian.net/browse/POS-2757)

Work in progress, upgrading Heimdall to the latest CometBFT and CosmosSDK.

## How to use keyring

Instructions on how to import your validator private key into the keyring and use it to sign transactions.

Get your base64 encoded private key from:  
`cat /var/lib/heimdall/config/priv_validator_key.json`

Convert the base64 encoded key to hex encoded key:  
`echo "<PRIVATE KEY BASE64 ENCODED>" | base64 -d | xxd -p -c 256`

Import the hex encoded key to your keyring:  
`heimdalld keys import-hex <KEY-NAME> <PRIVATE KEY HEX ENCODED> --home <HOME_DIR_PATH>`

When you `tx` command just specify in `--from` argument the key name you set above. Example:  
`heimdalld tx gov vote 1 yes --from mykey`