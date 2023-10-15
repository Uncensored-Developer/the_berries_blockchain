package database

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

type Account string

func NewAccount(value string) common.Address {
	return common.HexToAddress(value)
}

type Txn struct {
	From  common.Address `json:"from"`
	To    common.Address `json:"to"`
	Value uint           `json:"value"`
	Data  string         `json:"data"`
	Time  uint64         `json:"time"`
}

func (t Txn) IsReward() bool {
	return t.Data == "reward"
}

func (t Txn) Hash() (Hash, error) {
	txnJson, err := json.Marshal(t)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(txnJson), nil
}

func NewTxn(from, to common.Address, value uint, data string) Txn {
	return Txn{from, to, value, data, uint64(time.Now().Unix())}
}
