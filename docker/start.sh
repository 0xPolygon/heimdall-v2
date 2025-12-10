#!/usr/bin/env sh

# Start heimdall.
heimdalld start --all --bridge --rest-server --home $HEIMDALL_DIR >./logs/heimdalld.log

# Tail logs.
tail -f ./logs/heimdalld.log
