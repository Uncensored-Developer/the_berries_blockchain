package database

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

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
	Height uint64 `json:"height"`
	Parent Hash   `json:"parent"`
	Time   uint64 `json:"time"`
}
type Block struct {
	Header BlockHeader `json:"header"`
	Txns   []Txn       `json:"txns"`
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}

func NewBlock(height uint64, parent Hash, time uint64, txns []Txn) Block {
	return Block{BlockHeader{height, parent, time}, txns}
}

func (b Block) Hash() (Hash, error) {
	blockJson, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(blockJson), nil
}
