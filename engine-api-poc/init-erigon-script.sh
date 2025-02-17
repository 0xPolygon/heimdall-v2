#!/bin/sh

for i in 0 1 2 3 4; do
  erigon init --datadir=./data/erigon-$i genesis.json
done