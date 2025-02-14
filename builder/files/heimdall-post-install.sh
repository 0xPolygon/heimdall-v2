#!/bin/sh

# TODO-HV2: test this script once app is ready (is this needed at all?)

set -e

PKG="heimdall"

if ! getent passwd $PKG >/dev/null ; then
    adduser --disabled-password --disabled-login --shell /usr/sbin/nologin --quiet --system --no-create-home --home /nonexistent $PKG
    echo "Created system user $PKG"
fi

