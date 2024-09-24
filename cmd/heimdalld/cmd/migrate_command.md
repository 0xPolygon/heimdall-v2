# Migration command

Progress regarding the migration command.

## Modules status

- [x] auth
- [x] checkpoint
- [x] clerk
- [x] bor

## Modules app state size on mainnet

- **auth**: 0.10 MB (107260 bytes)
- **bank**: 0.00 MB (27 bytes)
- **gov**: 0.01 MB (13407 bytes)
- **chainmanager**: 0.00 MB (772 bytes)
- **bor**: 424.81 MB (445450568 bytes)
- **topup**: 0.01 MB (7843 bytes)
- **clerk**: 2355.08 MB (2469477639 bytes)
- **checkpoint**: 15.85 MB (16620086 bytes)

## Modules migration steps

### checkpoint

Parsing ```checkpoint_buffer_time``` which is in nano-seconds and converting it into seconds with suffix ```s```, otherwise it cannot be unmarshaled into ```time.Duration```.
Renaming ```child_chain_block_interval``` to ```child_block_interval```.
Iterating over all the ```checkpoints``` decoding the hex-encoded ```root_hash``` and encoding it as base64.

### clerk

Iterating over all the ```records```, decoding the hex-encoded ```data``` property and encoding it as base64.

### bor

Iterating over all the ```spans``` and renaming ```bor_chain_id``` to ```chain_id```, ```span_id``` to ```id```.
Converting every ```validator``` in every ```span``` by renaming following properties, ```power``` to ```voting_power```,
```accum``` to ```proposer_priority```, ```ID``` to ```val_id``` and converting each ```pubKey``` from plain string to ```secp256k1.PubKey```.  

### auth

In a lot of the migrations we can see logic regarding vesting accounts. Thats not of interest to us since vesting logic is disabled in heimdall and we don't have such accounts.
We skip the modules accounts during the migration because the heimdall v2 will initialize them from zero anyways.

#### v0.37 - v0.38

No migration required, genesis files are comptabile. [Link](https://github.com/cosmos/cosmos-sdk/blob/37b7221abdda540270adb2d51bdc87a22e417339/x/genutil/client/cli/migrate.go#L31)

#### v0.38 - v0.39

None of the changes concern us in this migration, they refactored vesting accounts, added additional vesting account type and switched from the default Go json marshal/unmarshal package to using ```legacy.Cdc```. [Link](https://github.com/cosmos/cosmos-sdk/blob/v0.39.3/x/auth/legacy/v0_39/migrate.go)

#### v039 - v0.40

In this migration the ```Coins``` property was dropped and accounts start to get encoded into ```codectypes.Any```. [Link](https://github.com/cosmos/cosmos-sdk/blob/v0.41.4/x/auth/legacy/v040/migrate.go#L106)

#### v0.40 - v0.43

None of the changes concern us in this migration, its just about vesting accounts. [Link](https://github.com/cosmos/cosmos-sdk/blob/v0.45.16/x/auth/legacy/v043/store.go)

#### v0.43 - v0.45

No migration.

#### v0.45 - v0.46

There migration executed on the database that is just creating mapping from account address to account id. This is not interesting to us because such mappings are internal to the module and never exported. [Link](https://github.com/cosmos/cosmos-sdk/blob/v0.45.16/x/auth/legacy/v043/store.go)

#### v0.46 - v0.47

There is migration executed on the database, it moves params from x/params module to the x/auth module. Its not interesting to us, because we already store params in the auth module genesis. [Link](https://github.com/cosmos/cosmos-sdk/blob/v0.47.14/x/auth/migrations/v4/migrate.go)

#### v0.47 - v0.50

There is migration executed on the database, that changes how global account number is stored. Its not interesting to us because during the import of the genesis, the auth module will find the highest account number and store it in the appropriate key. [Link](https://github.com/cosmos/cosmos-sdk/blob/v0.50.6/x/auth/migrations/v5/migrate.go)