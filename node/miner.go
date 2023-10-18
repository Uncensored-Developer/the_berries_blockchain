package node

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"kryptcoin/database"
	"kryptcoin/fs"
	"log"
	"math/rand"
	"time"
)

type PendingBlock struct {
	parent database.Hash
	height uint64
	time   uint64
	miner  common.Address
	txns   []database.SignedTxn
}

func NewPendingBlock(parent database.Hash, height uint64, miner common.Address, txns []database.SignedTxn) PendingBlock {
	return PendingBlock{parent, height, uint64(time.Now().Unix()), miner, txns}
}

func generateNonce() uint32 {
	rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return rand.Uint32()
}

func Mine(ctx context.Context, pb PendingBlock) (database.Block, error) {
	if len(pb.txns) == 0 {
		return database.Block{}, fmt.Errorf("mining empty blocks is not allowed")
	}

	start := time.Now()
	attempts := 0
	var block database.Block
	var hash database.Hash
	var nonce uint32

	for !database.IsBlockHashValid(hash) {
		select {
		case <-ctx.Done():
			log.Printf("Mining cancelled!")
			return database.Block{}, fmt.Errorf("mining cancelled: %s", ctx.Err())
		default:
		}

		attempts++
		nonce = generateNonce()

		// log every 1 million attempts
		if attempts%1000000 == 0 || attempts == 1 {
			log.Printf("Mining %d pending TXNs, Attempt: %d\n", len(pb.txns), attempts)
		}

		block = database.NewBlock(pb.height, pb.parent, pb.time, nonce, pb.miner, pb.txns)
		blockHash, err := block.Hash()
		if err != nil {
			return database.Block{}, fmt.Errorf("counld not mine block: %s", err.Error())
		}
		hash = blockHash
	}

	log.Printf("\nMined new Block %x using PoW %s:\n", hash, fs.Unicode("\\U1F389"))
	log.Printf("\tHeight: '%v'\n", block.Header.Height)
	log.Printf("\tNonce: '%v'\n", block.Header.Nonce)
	log.Printf("\tCreated: '%v'\n", block.Header.Time)
	log.Printf("\tMiner: '%v'\n", block.Header.Miner)
	log.Printf("\tParent: '%v'\n\n", block.Header.Parent.Hex())

	log.Printf("\tAttempt: '%v'\n", attempts)
	log.Printf("\tTime: %s\n\n", time.Since(start))

	return block, nil
}
