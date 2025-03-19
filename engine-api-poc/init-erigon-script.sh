#!/bin/sh

for i in $(seq 0 $(expr $NODES - 1)); do
  erigon init --datadir=/data/erigon-$i /config/genesis.json
done