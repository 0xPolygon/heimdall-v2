[//]: # (TODO HV2: https://polygon.atlassian.net/browse/POS-2757)
[//]: # (TODO HV2: https://polygon.atlassian.net/browse/POS-2780)

# Bor module

## Table of Contents

* [Preliminary terminology](#preliminary-terminology)
* [Overview](#overview)
* [How does it work](#how-does-it-work)
	* [How to propose a span](#how-to-propose-a-span)
* [Query commands](#query-commands)


## Preliminary terminology

* A `side-transaction` is a normal heimdall transaction but the data with which the message is composed needs to be voted on by the validators since the data is obscure to the consensus protocol itself, and it has no way of validating the data's correctness.
* A `sprint` comprises of 16 bor blocks (configured in [bor](https://github.com/maticnetwork/launch/blob/fe86ba6cd16e5c36067a5ae49c0bad62ce8b1c3f/mainnet-v1/sentry/validator/bor/genesis.json#L26C18-L28)).
* A `span` comprises of 400 sprints in bor (check heimdall's bor [params](https://heimdall-api.polygon.technology/bor/params) endpoint ).

## Overview

The validators on bor chain produce blocks in sprints and spans. Hence, it is imperative for the protocol to formalise the validators who will be producers in a range of blocks (`span`). The `bor` module in heimdall facilitates this by pseudo-randomly selecting validators who will producing blocks (producers) from the current validator set. The bor chain fetches and persists this information before the next span begins. `bor` module is a crucial component in heimdall since the PoS chain "liveness" depends on it.

## How does it work

A `Span` is defined by the data structure:

```protobuf
message Span {
	uint64 id = 1;
	uint64 start_block = 2;
	uint64 end_block = 3;
	heimdallv2.stake.ValidatorSet validator_set = 4
	[ (gogoproto.nullable) = false ];
	repeated heimdallv2.stake.Validator selected_producers = 5
	[ (gogoproto.nullable) = false ];
	string chain_id = 6;
}
```
where ,

* `id` means the id of the span, calculated by monotonically incrementing the id of the previous span.
* `start_block` corresponds to the block in bor from which the given span would commence.
* `end_block` corresponds to the block in bor at which the given span would conclude.
* `validator_set` defines the set of active validators.
* `selected_producers` are the validators selected to produce blocks in bor from the validator set.
* `chain_id` corresponds to bor chain ID.

A validator on heimdall can construct a span proposal message:

```protobuf
message MsgProposeSpan {
	option (cosmos.msg.v1.signer) = "proposer";

	uint64 span_id = 1;
	string proposer = 2 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
	uint64 start_block = 3;
	uint64 end_block = 4;
	string chain_id = 5;
	bytes seed = 6;
}
```

The msg is generally constructed and broadcast by the validator's bridge process periodically, but the CLI can also be leveraged to do the same manually (see [below](#how-does-it-work)). Upon broadcasting the message, it is initially checked by `ProposeSpan` handler for basic sanity (verify whether the proposed span is in continuity, appropriate span duration, correct chain ID, etc.). Since this is a side-transaction, the validators then vote on the data present in `MsgProposeSpan` on the basis of its correctness. All these checks are done in `SideHandleMsgSpan` (verifying `seed`, span continuity, etc.) and if correct, the validator would vote `YES`.
Finally, if there are 2/3+ `YES` votes, the `PostHandleMsgSpan` persists the proposed span in the state via the keeper :  

```go
// freeze for new span
err = s.k.FreezeSet(ctx, msg.SpanId, msg.StartBlock, msg.EndBlock, msg.ChainId, common.Hash(msg.Seed))
if err != nil {
	s.k.Logger(ctx).Error("Unable to freeze validator set for span", "span id", msg.SpanId, "error", err)
	return

}
```

`FreezeSet` internally invokes `SelectNextProducers`, which pseudo-randomly picks producers from the validator set, leaning more towards validators with higher voting power based on stake:

```go
// select next producers
newProducers, err := k.SelectNextProducers(ctx, seed)
if err != nil {
	return err
}
```

and then initialises and stores the span:

```go
newSpan := &types.Span{
	Id:                id,
	StartBlock:        startBlock,
	EndBlock:          endBlock,
	ValidatorSet:      k.sk.GetValidatorSet(ctx),
	SelectedProducers: newProducers,
	ChainId:           borChainID,
}

return k.AddNewSpan(ctx, newSpan)
```

### How to propose a span

A validator can leverage the CLI to propose a span like so :

```bash
./build/heimdalld tx bor propose-span --proposer <VALIDATOR_ADDRESS> --start-block <BOR_START_BLOCK> --span-id <SPAN_ID> --bor-chain-id <BOR_CHAIN_ID>
```

## Query commands

One can run the following query commands from the bor module :

* `span` - Query the span corresponding to the given span id.
* `latest span` - Query the latest span.
* `params` - Fetch the parameters associated to bor module.
* `spanlist` - Fetch span list.
* `next-span-seed` - Query the seed for the next span.
* `propose-span` - Print the `propose-span` command.

### CLI commands

```bash
./build/heimdalld query bor span-by-id <SPAN_ID>
```

```bash
./build/heimdalld query bor latest-span
```

```bash
./build/heimdalld query bor params
```

```bash
./build/heimdalld query bor span-list
```

```bash
./build/heimdalld query bor next-span-seed
```

```bash
./build/heimdalld tx bor propose-span --proposer <VALIDATOR_ADDRESS> --start-block <BOR_START_BLOCK> --span-id <SPAN_ID> --bor-chain-id <BOR_CHAIN_ID>
```

### REST endpoints

```bash
curl localhost:1317/bor/span/<SPAN_ID>
```

```bash
curl localhost:1317/bor/span/list
```

```bash
curl localhost:1317/bor/span/latest
```

```bash
curl localhost:1317/bor/params
```

```bash
curl localhost:1317/bor/span/seed
```

```bash
curl "localhost:1317/bor/span/prepare?span_id=<SPAN_ID>&start_block=<BOR_START_BLOCK>&chain_id="<BOR_CHAIN_ID>""
```
