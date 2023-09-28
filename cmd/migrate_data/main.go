package main

import (
	"kryptcoin/database"
	"log"
	"os"
	"time"
)

func main() {
	cwd, _ := os.Getwd()
	state, err := database.NewStateFromDisk(cwd)
	if err != nil {
		log.Fatalf("Error reading state: %v\n", err)
	}
	defer state.Close()

	block0 := database.NewBlock(
		database.Hash{},
		uint64(time.Now().Unix()),
		[]database.Txn{
			database.NewTxn("gold_rodger", "gold_rodger", 3, ""),
			database.NewTxn("gold_rodger", "gold_rodger", 700, "reward"),
		},
	)
	err = state.AddBlock(block0)
	if err != nil {
		log.Fatalf("Error adding block: %v\n", err)
	}
	block0Hash, err := state.Persist()
	if err != nil {
		log.Fatalf("Error error saving state: %v\n", err)
	}

	block1 := database.NewBlock(
		block0Hash,
		uint64(time.Now().Unix()),
		[]database.Txn{
			database.NewTxn("gold_rodger", "white_beard", 2000, ""),
			database.NewTxn("gold_rodger", "gold_rodger", 100, "reward"),
			database.NewTxn("white_beard", "gold_rodger", 1, ""),
			database.NewTxn("white_beard", "rocks", 1000, ""),
			database.NewTxn("white_beard", "gold_rodger", 50, ""),
			database.NewTxn("gold_rodger", "gold_rodger", 600, "reward"),
		},
	)

	err = state.AddBlock(block1)
	if err != nil {
		log.Fatalf("Error adding block: %v\n", err)
	}
	_, err = state.Persist()
	if err != nil {
		log.Fatalf("Error error saving state: %v\n", err)
	}
}
