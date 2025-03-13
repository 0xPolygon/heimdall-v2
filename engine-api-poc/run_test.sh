#!/bin/bash

mnemonics=(
  "erupt oven loud noise rug proof sunset gas table era dizzy vault"
  "charge taxi rifle female calm mask sea holiday wheat paddle expand surprise"
  "clip avoid maze squeeze one chest space style define leave sing dignity"
  "slam lab bind click ice prepare online reason wedding resist process exclude"
  "again cinnamon vessel dignity wise bike creek escape master siren govern battle"
)

folder="./engine-api-poc/test-output"
rm -rf "$folder"
mkdir "$folder"

# Check if polycli is installed and working
if ! polycli version > /dev/null 2>&1; then
    echo "- polycli is not installed or not working properly. Please install polycli to continue."
    echo "- You can do this via this link: https://github.com/0xPolygon/polygon-cli?tab=readme-ov-file#install"
    exit 1
fi

# Check if polycli is installed and working
if ! cast --version > /dev/null 2>&1; then
    echo "- cast is not installed or not working properly. Please install cast to continue."
    echo "- You can do this via this link: https://github.com/foundry-rs/foundry?tab=readme-ov-file#installation"
    exit 1
fi

echo "Starting testing rountines..."

  
# Launch 5 processes concurrently
for i in {0..4}; do
  prefix=$((i + 8))
  port="${prefix}545"
  mnemonic="${mnemonics[i]}"
  out_file="./engine-api-poc/test-output/test_${i}.json"
  private_key=$(cast wallet private-key "${mnemonics[i]}")
  
  polycli loadtest -r "http://localhost:${port}" \
    --private-key "${private_key}" --verbosity 100 \
    --mode "t" --requests 50000 --rate-limit 350 > "$out_file" 2>&1 &
  
  pid=$!
  echo "Started polycli on port $port with PID: $pid"
done

# Wait for all background processes to finish for this iteration
wait
echo "All processes finished"

