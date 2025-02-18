#!/bin/bash

mnemonics=(
  "erupt oven loud noise rug proof sunset gas table era dizzy vault"
  "charge taxi rifle female calm mask sea holiday wheat paddle expand surprise"
  "clip avoid maze squeeze one chest space style define leave sing dignity"
  "slam lab bind click ice prepare online reason wedding resist process exclude"
  "again cinnamon vessel dignity wise bike creek escape master siren govern battle"
)

folder="./engine-api-poc/test-output"
if [ ! -d "$folder" ]; then
  mkdir "$folder"
fi

echo "Starting pandoras-box processes..."

# Run 10 iterations
for iteration in {1..10}; do
  echo "=== Iteration $iteration ==="
  
  # Launch 5 processes concurrently
  for i in {0..4}; do
    prefix=$((i + 8))
    port="${prefix}545"
    mnemonic="${mnemonics[i]}"
    temp_file="./engine-api-poc/test-output/temp_${i}.json"
    
    # Run pandoras-box, writing its output to a temporary file
    pandoras-box -url "http://localhost:${port}" \
      -m "$mnemonic" \
      -t 100 -b 500 -o "$temp_file" > /dev/null 2>&1 &
    
    pid=$!
    echo "Started pandoras-box on port $port with PID: $pid"
  done
  
  # Wait for all background processes to finish for this iteration
  wait
  echo "All processes finished for iteration $iteration."

  # Append the new results to each cumulative JSON output file
  for i in {0..4}; do
    cumulative_file="./engine-api-poc/test-output/myOutput_${i}.json"
    temp_file="./engine-api-poc/test-output/temp_${i}.json"

    # If the cumulative file doesn't exist, create it as an empty JSON array
    if [ ! -f "$cumulative_file" ]; then
      echo "[]" > "$cumulative_file"
    fi

    # Append the new result to the cumulative array using jq
    jq --argjson new "$(cat "$temp_file")" '. += [$new]' "$cumulative_file" > "${cumulative_file}.tmp" \
      && mv "${cumulative_file}.tmp" "$cumulative_file"
    
    # Remove the temporary file
    rm -f "$temp_file"
  done

  echo "Aggregated results for iteration $iteration."
  echo ""
done

echo "All iterations complete."
