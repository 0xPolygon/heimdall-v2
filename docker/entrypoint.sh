#!/usr/bin/env sh

# TODO-HV2: check this file once we have a proper docker build

exec heimdalld --home="$HEIMDALL_DIR" "$@"
