package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Define command-line flags.
	rpcURL := flag.String("rpc", "", "RPC URL of Ethereum node")
	numBlocks := flag.Int("blocks", 0, "Number of blocks to fetch")
	flag.Parse()

	if *rpcURL == "" || *numBlocks <= 0 {
		flag.Usage()
		return
	}

	// Connect to the Ethereum node.
	client, err := ethclient.Dial(*rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// Retrieve the latest block header.
	latestHeader, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to get the latest block header: %v", err)
	}
	latestBlockNumber := latestHeader.Number

	var totalTxs int
	var firstBlockTimestamp uint64 // timestamp of the latest block (first fetched)
	var lastBlockTimestamp uint64  // timestamp of the oldest block in our fetched list

	// Fetch the latest block first.
	block, err := client.BlockByNumber(context.Background(), latestBlockNumber)
	if err != nil {
		log.Fatalf("Failed to get block %v: %v", latestBlockNumber, err)
	}
	firstBlockTimestamp = block.Time() // assumed to be in milliseconds
	totalTxs += len(block.Transactions())

	// Iterate to fetch older blocks.
	// Starting from i=1 because the latest block is already fetched.
	for i := 1; i < *numBlocks; i++ {
		blockNumber := new(big.Int).Sub(latestBlockNumber, big.NewInt(int64(i)))
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Fatalf("Failed to get block %v: %v", blockNumber, err)
		}
		totalTxs += len(block.Transactions())
		// Keep updating lastBlockTimestamp; after loop, it will hold the oldest block's timestamp.
		lastBlockTimestamp = block.Time()
	}

	// If only one block is fetched, assign lastBlockTimestamp to firstBlockTimestamp.
	if *numBlocks == 1 {
		lastBlockTimestamp = firstBlockTimestamp
	}

	// Calculate the overall time interval between the first and last block.
	// Note: the latest block is expected to have a larger timestamp.
	var timeInterval uint64
	if firstBlockTimestamp > lastBlockTimestamp {
		timeInterval = firstBlockTimestamp - lastBlockTimestamp
	} else {
		timeInterval = lastBlockTimestamp - firstBlockTimestamp
	}

	// Convert interval from milliseconds to seconds.
	timeIntervalSeconds := float64(timeInterval) / 1000.0

	// Calculate TPS (transactions per second)
	var tps float64
	if timeIntervalSeconds > 0 {
		tps = float64(totalTxs) / timeIntervalSeconds
	} else {
		tps = 0
	}

	fmt.Printf("Fetched %d blocks (from block %s to %s)\n", *numBlocks,
		new(big.Int).Sub(latestBlockNumber, big.NewInt(int64(*numBlocks-1))).String(),
		latestBlockNumber.String())
	fmt.Printf("Total transactions: %d\n", totalTxs)
	fmt.Printf("Time interval between first (latest) and last (oldest) block: %d ms (%.2f seconds)\n", timeInterval, timeIntervalSeconds)
	fmt.Printf("TPS (transactions per second): %.2f\n", tps)
}
