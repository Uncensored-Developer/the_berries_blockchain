package wallet

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"path/filepath"
)

const keystoreDirName = "keystore"

func GetKeystoreDirPath(dataDir string) string {
	return filepath.Join(dataDir, keystoreDirName)
}

func Sign(msg []byte, privateKey *ecdsa.PrivateKey) (sig []byte, err error) {
	// Hash msg to 32 bytes
	msgHash := crypto.Keccak256(msg)

	// Sign msgHash using private key
	sig, err = crypto.Sign(msgHash, privateKey)
	if err != nil {
		return nil, err
	}

	if len(sig) != crypto.SignatureLength {
		return nil, fmt.Errorf(
			"wrong size for signatureL got %d, expects %d",
			len(sig),
			crypto.SignatureLength,
		)
	}
	return sig, nil
}

func Verify(msg, sig []byte) (*ecdsa.PublicKey, error) {
	msgHash := crypto.Keccak256(msg)

	recoveredPublicKey, err := crypto.SigToPub(msgHash, sig)
	if err != nil {
		return nil, fmt.Errorf("unable to verify message signature: %s", err.Error())
	}
	return recoveredPublicKey, nil
}
