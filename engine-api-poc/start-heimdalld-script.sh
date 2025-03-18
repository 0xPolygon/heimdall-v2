#!/bin/bash

heimdalld start --home /data --bor_engine_jwt $BOR_ENGINE_JWT --bor_engine_url $BOR_ENGINE_URL --bor_rpc_url $BOR_RPC_URL | tee /data/app.log


