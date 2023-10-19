package database

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"time"
)

type Account string

func NewAccount(value string) common.Address {
	return common.HexToAddress(value)
}

type Txn struct {
	From common.Address `json:"from"`
	To   common.Address `json:"to"`

	Gas      uint `json:"gas"`
	GasPrice uint `json:"gasPrice"`

	Value uint   `json:"value"`
	Nonce uint   `json:"nonce"`
	Data  string `json:"data"`
	Time  uint64 `json:"time"`
}

type SignedTxn struct {
	Txn
	Sig []byte `json:"signature"`
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

func (t Txn) Encode() ([]byte, error) {
	return json.Marshal(t)
}

func (t Txn) GasCost() uint {
	return t.Gas * t.GasPrice
}

func (t Txn) TotalCost(isForkOIP1 bool) uint {
	if isForkOIP1 {
		return t.Value + t.GasCost()
	}
	return t.Value + TxnFee
}

func (t Txn) MarshalJSON() ([]byte, error) {
	// Prior OIP1
	if t.Gas == 0 {
		type legacyTxn struct {
			From  common.Address `json:"from"`
			To    common.Address `json:"to"`
			Value uint           `json:"value"`
			Nonce uint           `json:"nonce"`
			Data  string         `json:"data"`
			Time  uint64         `json:"time"`
		}
		return json.Marshal(legacyTxn{t.From, t.To, t.Value, t.Nonce, t.Data, t.Time})
	}
	type oip1Txn struct {
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		Gas      uint           `json:"gas"`
		GasPrice uint           `json:"gasPrice"`
		Value    uint           `json:"value"`
		Nonce    uint           `json:"nonce"`
		Data     string         `json:"data"`
		Time     uint64         `json:"time"`
	}
	return json.Marshal(oip1Txn{t.From, t.To, t.Gas, t.GasPrice, t.Value, t.Nonce, t.Data, t.Time})
}

func (s SignedTxn) MarshalJSON() ([]byte, error) {
	// Prior OIP1
	if s.Gas == 0 {
		type legacyTxn struct {
			From  common.Address `json:"from"`
			To    common.Address `json:"to"`
			Value uint           `json:"value"`
			Nonce uint           `json:"nonce"`
			Data  string         `json:"data"`
			Time  uint64         `json:"time"`
			Sig   []byte         `json:"signature"`
		}
		return json.Marshal(legacyTxn{s.From, s.To, s.Value, s.Nonce, s.Data, s.Time, s.Sig})
	}
	type oip1Txn struct {
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		Gas      uint           `json:"gas"`
		GasPrice uint           `json:"gasPrice"`
		Value    uint           `json:"value"`
		Nonce    uint           `json:"nonce"`
		Data     string         `json:"data"`
		Time     uint64         `json:"time"`
		Sig      []byte         `json:"signature"`
	}
	return json.Marshal(oip1Txn{s.From, s.To, s.Gas, s.GasPrice, s.Value, s.Nonce, s.Data, s.Time, s.Sig})
}

func (s SignedTxn) IsAuthentic() (bool, error) {
	txnHash, err := s.Txn.Hash()
	if err != nil {
		return false, err
	}

	// Verify if the signature is compatible with this msg
	recoveredPublicKey, err := crypto.SigToPub(txnHash[:], s.Sig)
	if err != nil {
		return false, err
	}

	// Convert the recovered public key to an account
	recoveredPublicKeyBytes := elliptic.Marshal(
		crypto.S256(),
		recoveredPublicKey.X,
		recoveredPublicKey.Y,
	)
	recoveredPublicKeyBytesHash := crypto.Keccak256(recoveredPublicKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPublicKeyBytesHash[12:])

	// Compare the signature owner with txn owner
	return recoveredAccount.Hex() == s.From.Hex(), nil
}

func NewTxn(from, to common.Address, gas, gasPrice, value, nonce uint, data string) Txn {
	return Txn{
		from,
		to,
		gas,
		gasPrice,
		value,
		nonce,
		data,
		uint64(time.Now().Unix()),
	}
}

func NewDefaultTxn(from, to common.Address, value, nonce uint, data string) Txn {
	return NewTxn(from, to, TxnGas, DefaultGasPrice, value, nonce, data)
}

func NewSignedTxn(txn Txn, sig []byte) SignedTxn {
	return SignedTxn{txn, sig}
}
