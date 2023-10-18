package node

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"kryptcoin/database"
	"kryptcoin/wallet"
	"testing"
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
	minerPrivateKey, _, miner, err := generateKey()
	if err != nil {
		t.Fatal(err)
	}

	pendingBlock, err := createRandomPendingBlock(minerPrivateKey, miner)
	if err != nil {
		t.Fatal(err)
	}

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

	if minedBlock.Header.Miner.String() != miner.String() {
		t.Fatal("mined block miner should be the miner from pending block.")
	}
}

func createRandomPendingBlock(privateKey *ecdsa.PrivateKey, miner common.Address) (PendingBlock, error) {
	txn := database.NewTxn(miner, database.NewAccount(testKeystoreWhiteBeardAccount), 1, 1, "")
	signedTxn, err := wallet.SignTxn(txn, privateKey)
	if err != nil {
		return PendingBlock{}, err
	}
	return NewPendingBlock(
		database.Hash{},
		0,
		miner,
		[]database.SignedTxn{signedTxn},
	), nil
}

func generateKey() (*ecdsa.PrivateKey, ecdsa.PublicKey, common.Address, error) {
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, ecdsa.PublicKey{}, common.Address{}, err
	}

	publicKey := privateKey.PublicKey
	publicKeyBytes := elliptic.Marshal(crypto.S256(), publicKey.X, publicKey.Y)
	publicKeyBytesHash := crypto.Keccak256(publicKeyBytes[1:])
	account := common.BytesToAddress(publicKeyBytesHash[12:])

	return privateKey, publicKey, account, nil
}
