#!/bin/bash


heimdalld create-testnet --v $NODES --n 0 --output-dir /data --home /data

for (( i=1; i<$NODES; i++ )); do
    cp /data/node0/heimdalld/config/genesis.json /data/node$i/heimdalld/config/genesis.json 
done

# for i in {0..4}
# do 
#     P2P_PORT=$((20000+$i))
#     TCP_PORT=$((30000+$i))

#     # init each node
#     mkdir -p build/node-$i; 
#     heimdalld init node-$i --home /data/node-$i

#     PEERS=$PEERS,$(heimdalld comet show-node-id --home /data/node-$i)@heimdalld-$i:$P2P_PORT
#     RPCS=$RPCS,heimdalld-$i:$TCP_PORT

# done

# PEERS="${PEERS:1}"
# RPCS="${RPCS:1}"

# for i in {0..4}
# do 
#     P2P_PORT=$((20000+$i))
#     TCP_PORT=$((30000+$i))

#     sed -i "s/26656/$P2P_PORT/g" /data/node-$i/config/config.toml
#     sed -i "s|127.0.0.1:26657|0.0.0.0:$TCP_PORT|g" /data/node-$i/config/config.toml
#     sed -i 's|addr_book_strict = true|addr_book_strict = false|' /data/node-$i/config/config.toml
#     sed -i 's|proxy_app = "tcp://127.0.0.1:26658"|proxy_app = "heimdalld"|' /data/node-$i/config/config.toml
#     sed -i "s|persistent_peers = \"\"|persistent_peers = \"$PEERS\"|" /data/node-$i/config/config.toml
#     sed -i "s|rpc_servers = \"\"|rpc_servers = \"$RPCS\"|" /data/node-$i/config/config.toml
# done



