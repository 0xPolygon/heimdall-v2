#!/bin/bash
. ./engine-api-poc/deployment/.env; 

folder="./engine-api-poc/test-output"
rm -rf "$folder"
mkdir "$folder"

# Check if polycli is installed and working
if ! polycli version > /dev/null 2>&1; then
    echo "- polycli is not installed or not working properly. Please install polycli to continue."
    echo "- You can do this via this link: https://github.com/0xPolygon/polygon-cli?tab=readme-ov-file#install"
    exit 1
fi

echo "Starting testing rountines..."

  
# Launch processes concurrently
for (( i=0; i<$NODES; i++ )); do
  prefix=$((i + 8))
  port="${prefix}545"
  mnemonic="${mnemonics[i]}"
  out_file="./engine-api-poc/test-output/test_${i}.json"
  private_key=$(jq -r '.['"$i"'].private_key' ./engine-api-poc/deployment/wallets_for_test.json)
  
  polycli loadtest -r "http://localhost:${port}" \
    --private-key "${private_key}" --verbosity 100 \
    --mode "t" --requests 50000 --rate-limit 350 > "$out_file" 2>&1 &
  
  pid=$!
  echo "Started polycli on port $port with PID: $pid"
done

# Wait for all background processes to finish for this iteration
wait
echo "All processes finished"

