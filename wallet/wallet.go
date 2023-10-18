package wallet

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"kryptcoin/database"
	"os"
	"path/filepath"
)

const keystoreDirName = "keystore"

func GetKeystoreDirPath(dataDir string) string {
	return filepath.Join(dataDir, keystoreDirName)
}

func Sign(msg []byte, privateKey *ecdsa.PrivateKey) (sig []byte, err error) {
	msgHash := sha256.Sum256(msg)
	return crypto.Sign(msgHash[:], privateKey)
}

func SignTxn(txn database.Txn, privateKey *ecdsa.PrivateKey) (database.SignedTxn, error) {
	rawTxn, err := txn.Encode()
	if err != nil {
		return database.SignedTxn{}, err
	}

	sig, err := Sign(rawTxn, privateKey)
	if err != nil {
		return database.SignedTxn{}, err
	}
	return database.NewSignedTxn(txn, sig), nil
}

func Verify(msg, sig []byte) (*ecdsa.PublicKey, error) {
	msgHash := sha256.Sum256(msg)

	recoveredPublicKey, err := crypto.SigToPub(msgHash[:], sig)
	if err != nil {
		return nil, fmt.Errorf("unable to verify message signature: %s", err.Error())
	}
	return recoveredPublicKey, nil
}

func SignWithKeystoreAccount(txn database.Txn, acct common.Address, password, keystoreDir string) (database.SignedTxn, error) {
	ks := keystore.NewKeyStore(keystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
	ksAccount, err := ks.Find(accounts.Account{Address: acct})
	if err != nil {
		return database.SignedTxn{}, err
	}

	ksAccountJson, err := os.ReadFile(ksAccount.URL.Path)
	if err != nil {
		return database.SignedTxn{}, err
	}

	key, err := keystore.DecryptKey(ksAccountJson, password)
	if err != nil {
		return database.SignedTxn{}, err
	}
	signedTxn, err := SignTxn(txn, key.PrivateKey)
	if err != nil {
		return database.SignedTxn{}, err
	}
	return signedTxn, nil
}
