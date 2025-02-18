[//]: # (TODO HV2: https://polygon.atlassian.net/browse/POS-2757)

# Chainmanager module

## Table of Contents

* [Overview](#overview)
* [Query commands](#query-commands)

## Overview

The chainmanager module is responsible for fetching the PoS protocol parameters. These params include addresses of contracts deployed on mainchain (Ethereum) and borchain (Bor), chain ids, mainchain and borchain confirmation blocks.

```protobuf
// ChainParams contains contract addresses and other chain specific parameters
message ChainParams {
  option (gogoproto.equal) = true;
  string bor_chain_id = 1;
  // L1 Chain Contracts
  string pol_token_address = 2;
  string staking_manager_address = 3;
  string slash_manager_address = 4;
  string root_chain_address = 5;
  string staking_info_address = 6;
  string state_sender_address = 7;
  // Bor Chain Contracts
  string state_receiver_address = 8;
  string validator_set_address = 9;
}

// Params contains the chain params for chainmanager module
message Params {
  option (gogoproto.equal) = true;
  ChainParams chain_params = 1 [ (gogoproto.nullable) = false ];
  uint64 main_chain_tx_confirmations = 2;
  uint64 bor_chain_tx_confirmations = 3;
}
```

## Query commands

One can run the following query commands from the chainmanager module :

* `params` - Fetch the parameters associated to chainmanager module.

### CLI commands

```bash
./build/heimdalld query chainmanager params
```

### REST endpoints

```bash
curl localhost:1317/heimdallv2/chainmanager/params
```
