package database

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"regexp"
)

const Reward = 100
const blockDifficulty = 3

type Hash [32]byte

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

func (h Hash) Hex() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) IsEmpty() bool {
	emptyHash := Hash{}
	return bytes.Equal(emptyHash[:], h[:])
}

type BlockHeader struct {
	Height uint64         `json:"height"`
	Parent Hash           `json:"parent"`
	Time   uint64         `json:"time"`
	Nonce  uint32         `json:"nonce"`
	Miner  common.Address `json:"miner"`
}
type Block struct {
	Header BlockHeader `json:"header"`
	Txns   []SignedTxn `json:"txns"`
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}

func NewBlock(height uint64, parent Hash, time uint64, nonce uint32, miner common.Address, txns []SignedTxn) Block {
	return Block{BlockHeader{height, parent, time, nonce, miner}, txns}
}

func (b Block) Hash() (Hash, error) {
	blockJson, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(blockJson), nil
}

func (b Block) GasReward() uint {
	reward := uint(0)
	for _, txn := range b.Txns {
		reward += txn.GasCost()
	}
	return reward
}

// IsBlockHashValid Validates that the block hash starts with 2 leading zeros
func IsBlockHashValid(hash Hash) bool {
	hexHash := hash.Hex()
	pattern := "^0*"

	re := regexp.MustCompile(pattern)
	match := re.FindString(hexHash)
	return len(match) == blockDifficulty
}
