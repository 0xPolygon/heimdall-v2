# Heimdall App

Heimdall is an [ABCI 2.0](https://github.com/cometbft/cometbft/blob/main/spec/abci/abci%2B%2B_basic_concepts.md) (Application Blockchain Interface) application built with [Cosmos SDK](https://github.com/cosmos/cosmos-sdk/tree/main) operating on [CometBFT consensus engine](https://github.com/cometbft/cometbft/tree/main).

We have segregated modules integrated with Heimdall into:

* Common modules: These modules are provided by Cosmos SDK out of the box (`auth`, `bank`, `gov`, etc.). We maintain a [fork of Cosmos SDK](https://github.com/0xPolygon/cosmos-sdk) on top of which we have made tweaks necessary for Heimdall's business logic.

* Custom modules: These modules have been written from scratch, specific to Heimdall's business logic. These are present in the `x/` directory.

Currently Heimdall app integrates the following custom modules:

* `bor`: [YET TO BE IMPLEMENTED] This module handles Bor block producer selection. Read more here[ADD LINK TO THE MODULE README].
* `chainmanager`: [YET TO BE IMPLEMENTED] This module is responsible for fetching important protocol parameters from Ethereum and Bor chain such as contract addresses, confirmation blocks, etc. Read more here[ADD LINK TO THE MODULE README].
* `checkpoint`: [YET TO BE IMPLEMENTED] This module takes care of creating and periodically submitting checkpoints (merkle root of Bor blocks) to L1 chain. Read more here[ADD LINK TO THE MODULE README].
* `clerk`: [YET TO BE IMPLEMENTED] This module manages the state sync mechanism, the process via which arbitrary messages are passed from L1 to Bor chain. Read more here[ADD LINK TO THE MODULE README].
* `stake`: [YET TO BE IMPLEMENTED] This module handles all things related to a validator's staking operations. Read more here[ADD LINK TO THE MODULE README].
* `topup`: [YET TO BE IMPLEMENTED] This module helps validators top up their accounts on heimdall which is used to pay fee when submitting transactions on Heimdall chain. Read more here[ADD LINK TO THE MODULE README].
