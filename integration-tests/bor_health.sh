#!/bin/bash

# TODO HV2: test this script once app is ready, and integrate into CI

set -e

while true
do
    peers=$(docker exec bor0 bash -c "bor attach /var/lib/bor/data/bor.ipc -exec 'admin.peers'")
    block=$(docker exec bor0 bash -c "bor attach /var/lib/bor/data/bor.ipc -exec 'eth.blockNumber'")

    if [[ -n "$peers" ]] && [[ -n "$block" ]]; then
        break
    fi
done

echo "$peers"
echo "$block"
