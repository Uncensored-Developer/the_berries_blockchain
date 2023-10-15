package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
)

func TestSignCryptoParams(t *testing.T) {
	// Generate new key on the fly
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(privateKey)

	// Prepare fake message to digitally sign
	msg := []byte("This is a test fake message for this test")

	sig, err := Sign(msg, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the length is 65 bytes
	if len(sig) != crypto.SignatureLength {
		t.Fatalf(
			"wrong size for signatureL got %d, expects %d",
			len(sig),
			crypto.SignatureLength,
		)
	}
}

func TestSign(t *testing.T) {
	// Generate new key on the fly
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Convert the public Key to bytes with the elliptic curve settings
	publicKey := privateKey.PublicKey
	publicKeyBytes := elliptic.Marshal(crypto.S256(), publicKey.X, publicKey.Y)

	// Hash the Public Key to 32 bytes
	publicKeyBytesHash := crypto.Keccak256(publicKeyBytes[1:])

	// The last 20 bytes of the public key hash would be the public username
	account := common.BytesToAddress(publicKeyBytesHash[12:])

	msg := []byte("This is a test fake message for this test")

	sig, err := Sign(msg, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	// Recover public key from the signature
	recoveredPublicKey, err := Verify(msg, sig)
	if err != nil {
		t.Fatal(err)
	}

	// Convert the public key to username again
	recoveredPublicKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPublicKey.X, recoveredPublicKey.Y)
	recoveredPublicKeyBytesHash := crypto.Keccak256(recoveredPublicKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPublicKeyBytesHash[12:])

	// Compare the username matches meaning, The signature generation and account
	// verification by extracting the public key from signature works
	if account.Hex() != recoveredAccount.Hex() {
		t.Fatalf(
			"msg was signed by account %s but signature recovery produced an account %s",
			account.Hex(),
			recoveredAccount.Hex(),
		)
	}
}
