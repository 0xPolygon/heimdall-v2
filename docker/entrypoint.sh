#!/usr/bin/env sh

# TODO-HV2: check this file once we have a proper docker build

if [ "$1" = 'heimdallcli' ]; then
    shift
    exec heimdallcli --home=$HEIMDALL_DIR "$@"
fi

exec heimdalld --home=$HEIMDALL_DIR "$@"
