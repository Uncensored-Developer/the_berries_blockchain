package node

import (
	"context"
	"encoding/hex"
	"kryptcoin/database"
	"testing"
	"time"
)

func TestValidBlockHash(t *testing.T) {
	hexHash := "00074f8fd89346fgc8"
	hash := database.Hash{}

	hex.Decode(hash[:], []byte(hexHash))

	if !database.IsBlockHashValid(hash) {
		t.Fatalf("hash '%s' starting with 3 zeros is supposed to be valid", hexHash)
	}
}

func TestInvalidBlockHash(t *testing.T) {
	hexHash := "000074f8fd89346fgc8"
	hash := database.Hash{}

	hex.Decode(hash[:], []byte(hexHash))

	if database.IsBlockHashValid(hash) {
		t.Fatalf("hash '%s' is not supposed to be valid", hexHash)
	}
}

func TestMine(t *testing.T) {
	miner := database.NewAccount("gold_rodger")
	pendingBlock := createRandomPendingBlock(miner)

	ctx := context.Background()

	minedBlock, err := Mine(ctx, pendingBlock)
	if err != nil {
		t.Fatal(err)
	}

	minedBlockHash, err := minedBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !database.IsBlockHashValid(minedBlockHash) {
		t.Fatal("Invalid block hash produced.")
	}

	if minedBlock.Header.Miner != miner {
		t.Fatal("mined block miner should be the miner from pending block.")
	}
}

func createRandomPendingBlock(miner database.Account) PendingBlock {
	return NewPendingBlock(
		database.Hash{},
		0,
		miner,
		[]database.Txn{
			database.Txn{From: "gold_rodger", To: "white_beard", Value: 1, Time: uint64(time.Now().Unix())},
		},
	)
}
