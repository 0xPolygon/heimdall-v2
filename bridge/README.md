# Bridge

## Table of Contents

- [Overview](#overview)
- [Listener](#listener)
- [Processor](#processor)
- [How to start bridge](#how-to-start-bridge)
- [Reset](#reset)
- [Common Issues (FAQ)](#common-issues-faq)

## Overview
Bridge module is responsible for listening to multiple chains and processing the events emitted by them.
It converts the emitted data into heimdall messages and sends them to the heimdall chain.
There are `listener` and `processor` components in the bridge module
which are responsible for listening and processing the events respectively as per their module.
For example `listener/rootchain.go` is responsible
for listening to events coming from the root chain or L1 i.e., Ethereum chain in our case
and `listener/polygonposchain.go` is responsible for listening to events coming from the bor chain

To process the events emitted by the chains, bridge module uses `processor` component,
which is responsible for processing the events emitted by the chains.
For example `processor/clerk.go` is responsible for processing the events related to clerk module,
`processor/staking.go` is responsible for processing the events related to staking module and so on.

Other components of the bridge module includes `queue` which is used for queuing the messages between listener and processors, `broadcaster` which is responsible for broadcasting the messages to the heimdall chain.

Polygon PoS bridge provides a bridging mechanism that is near-instant, low-cost, and quite flexible.

There is no change to the circulating supply of your token when it crosses the bridge:
- Tokens that leave the Ethereum network are locked, and the same tokens are minted on Polygon PoS as a pegged token (1:1).
- To move the tokens back to the Ethereum network, tokens are burned on Polygon PoS network and unlocked on the Ethereum network.

## Listener

The bridge module has a `BaseListener` which is extended by `RootChainListener`,
`BorChainListener` and `HeimdallListener`.
It has all the methods required to start polling,
listening to incoming headers(blocks) and stopping the process if required.
All the listeners use these properties with their individual implementations
on how to handle the incoming header once received.

For example,
in the `RootChainListener` the incoming header is used to determine the current height of the root chain
and calculate the `from` and `to` block numbers using which the events are fetched from the root chain.
These events are then sent to `handleLog` where based on their event signature they are added to queue as tasks for further processing by their respective processors.

## Processor

There is a `BaseProcessor` which is extended by every processor in the processor package.
It has all the methods required to start processing the events,
registering for the tasks which are there in the queue, and stopping the process if required.
All the processors use these properties with their individual implementations
on how to handle the incoming events once received.

Once the event is added to the queue by the `Listener`,
the `Processor` takes over and processes the events which are added into the queue based on their event signature.
Each processor has `RegisterTasks` method using
which they register to process specific tasks added to the queue based on the module they serve.
For example `StakingProcessor` registers for `Staking` related tasks,
`ClerkProcessor` registers for `Clerk` related tasks, and so on.
You can look into each processor to check which tasks they are registered for.

## How to start bridge

The bridge should only be used by validator nodes, as they are the ones who can send txs on the heimdall chain.
So if a non-validator node is running the bridge,
then it's of no use as it won't be able to send txs to the heimdall chain.

To start bridge as a validator, you have to add `--bridge` command to your heimdall node command.
You can also control which services you want to run from processor by using `--all` or `--only` flags.

To run all the services:

```bash
heimdalld start --bridge --all
```

## Reset

> :warning: Do this only when you are advised so, and you understand the impact of this command. 

If you want to reset the bridge server data, you can use the following command:

```bash
./heimdalld unsafe-reset-all
```

## Common Issues (FAQ)

### Connection reset by peer

If the node shows up the logs as below

```
panic: Exception (501) Reason: "read tcp 127.0.0.1:35218->127.0.0.1:5672: read: connection reset by peer"
goroutine 1 [running]:
github.com/0xPolygon/heimdall-v2/bridge/queue. NewQueueConnector({0xc0002ccf60,0x22})
```
Your Heimdall Bridge has issues please follow the steps below to fix this

```
sudo service rabbitmq-server stop
rm /var/lib/rabbitmq/mnesia
sudo service rabbitmq-server start
```

### Validator is unable to propose checkpoints

Please check the validator heimdall service and ensure the below flag is set and ensure rabbitmq is installed on your validator.

```
--bridge --all
```

Once done, restart the services.



