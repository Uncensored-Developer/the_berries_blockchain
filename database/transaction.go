package database

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
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
	log.Printf("==> %s", recoveredAccount.Hex())
	log.Printf("==> %s", s.From.Hex())
	// Compare the signature owner with txn owner
	return recoveredAccount.Hex() == s.From.Hex(), nil
}

func NewTxn(from, to common.Address, value uint, data string) Txn {
	return Txn{from, to, value, data, uint64(time.Now().Unix())}
}

func NewSignedTxn(txn Txn, sig []byte) SignedTxn {
	return SignedTxn{txn, sig}
}
