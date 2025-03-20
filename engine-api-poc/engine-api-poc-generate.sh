#!/bin/bash

if ! gomplate -v > /dev/null 2>&1; then 
    echo "gomplate is not installed or not working properly. Please install gomplate to continue."; 
    echo "You can do this via this link: https://docs.gomplate.ca/installing/"; 
    exit 1; 
fi; 
# Check if cast is installed and working 
if ! cast --version > /dev/null 2>&1; then 
    echo "- cast is not installed or not working properly. Please install cast to continue." 
    echo "- You can do this via this link: https://github.com/foundry-rs/foundry?tab=readme-ov-file#installation" 
    exit 1 
fi; 
if [ -z "$NODES" ]; then 
    echo "Enter number of nodes:"; 
    read _nodes; 
    export NODES="$_nodes"
else 
    echo "Using provided NODES=$NODES"
fi; 
echo "Building environment with $NODES nodes"; 
mkdir -p ./engine-api-poc/deployment
cp ./engine-api-poc/genesis-template.json ./engine-api-poc/deployment/genesis.json;
echo "[]" > ./engine-api-poc/deployment/wallets_for_test.json
for (( i=0; i<$NODES; i++ )); do
    NEWWALLET=$(cast wallet n --json)
    ADDRESSTOINSERT=$(echo "$NEWWALLET" | jq -r ".[0].address")
    WALLET=$(echo "$NEWWALLET" | jq -c '.[0]')
    jq --arg addr $ADDRESSTOINSERT --arg bal "0xFFFEEBE0B40E8000000" '.alloc[$addr] = {"balance": $bal}' ./engine-api-poc/deployment/genesis.json > tmp.json && mv tmp.json ./engine-api-poc/deployment/genesis.json;
    jq --argjson wallet "$WALLET" '. += [$wallet]' ./engine-api-poc/deployment/wallets_for_test.json > tmp.json && mv tmp.json ./engine-api-poc/deployment/wallets_for_test.json;
done

set -a 
. engine-api-poc/.env.e2e;
set +a
NODES=$NODES gomplate -f engine-api-poc/docker-compose.tmpl -o engine-api-poc/deployment/docker-compose.yaml; 
NODES=$NODES gomplate -f engine-api-poc/.env.tmpl -o engine-api-poc/deployment/.env
