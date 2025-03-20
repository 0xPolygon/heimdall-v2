#!/bin/bash
#
###################
# Remove heimdall profile installation
###################
sudo rm /var/lib/heimdall/config/app.toml
sudo rm /var/lib/heimdall/config/config.toml
sudo rm /var/lib/heimdall/config/client.toml
sudo systemctl daemon-reload
