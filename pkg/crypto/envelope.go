// Package crypto provides hybrid post-quantum encryption envelopes for
// the transfer agent, wrapping luxfi/crypto mlkem and mldsa primitives.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/mlkem"
)

// Algorithm identifiers for envelope metadata.
const (
	AlgoMLKEM1024_AES256GCM = "MLKEM1024-AES256GCM"
	AlgoMLDSA87             = "MLDSA87"
)

// Envelope is a hybrid PQ-encrypted message: ML-KEM key encapsulation + AES-256-GCM.
type Envelope struct {
	Ciphertext     []byte `json:"ciphertext"`
	EncapsulatedKey []byte `json:"encapsulated_key"`
	Nonce          []byte `json:"nonce"`
	Signature      []byte `json:"signature,omitempty"`
	SignerPubKey   []byte `json:"signer_pub_key,omitempty"`
	Algorithm      string `json:"algorithm"`
}

// Seal encrypts plaintext for recipientPubKey using ML-KEM-1024 + AES-256-GCM.
// recipientPubKey must be a serialized ML-KEM-1024 public key.
func Seal(plaintext []byte, recipientPubKey []byte) (*Envelope, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("crypto: plaintext is empty")
	}

	pk, err := mlkem.PublicKeyFromBytes(recipientPubKey, mlkem.MLKEM1024)
	if err != nil {
		return nil, fmt.Errorf("crypto: parse recipient public key: %w", err)
	}

	// ML-KEM encapsulate: produces ciphertext (encapsulated key) and shared secret.
	encapsulated, sharedSecret, err := pk.Encapsulate()
	if err != nil {
		return nil, fmt.Errorf("crypto: encapsulate: %w", err)
	}

	// Use the shared secret (32 bytes) as AES-256-GCM key.
	block, err := aes.NewCipher(sharedSecret[:32])
	if err != nil {
		return nil, fmt.Errorf("crypto: aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &Envelope{
		Ciphertext:      ciphertext,
		EncapsulatedKey: encapsulated,
		Nonce:           nonce,
		Algorithm:       AlgoMLKEM1024_AES256GCM,
	}, nil
}

// Open decrypts an Envelope using the recipient's ML-KEM-1024 private key.
func Open(env *Envelope, recipientPrivKey []byte) ([]byte, error) {
	if env == nil {
		return nil, errors.New("crypto: nil envelope")
	}
	if env.Algorithm != AlgoMLKEM1024_AES256GCM {
		return nil, fmt.Errorf("crypto: unsupported algorithm %q", env.Algorithm)
	}

	sk, err := mlkem.PrivateKeyFromBytes(recipientPrivKey, mlkem.MLKEM1024)
	if err != nil {
		return nil, fmt.Errorf("crypto: parse recipient private key: %w", err)
	}

	sharedSecret, err := sk.Decapsulate(env.EncapsulatedKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: decapsulate: %w", err)
	}

	block, err := aes.NewCipher(sharedSecret[:32])
	if err != nil {
		return nil, fmt.Errorf("crypto: aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, env.Nonce, env.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}

	return plaintext, nil
}

// Sign produces an ML-DSA-87 signature over data.
// signerPrivKey must be a serialized ML-DSA-87 private key.
func Sign(data []byte, signerPrivKey []byte) ([]byte, error) {
	sk, err := mldsa.PrivateKeyFromBytes(mldsa.MLDSA87, signerPrivKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: parse signer private key: %w", err)
	}

	sig, err := sk.Sign(rand.Reader, data, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: sign: %w", err)
	}

	return sig, nil
}

// Verify checks an ML-DSA-87 signature.
// signerPubKey must be a serialized ML-DSA-87 public key.
func Verify(data []byte, signature []byte, signerPubKey []byte) (bool, error) {
	pk, err := mldsa.PublicKeyFromBytes(signerPubKey, mldsa.MLDSA87)
	if err != nil {
		return false, fmt.Errorf("crypto: parse signer public key: %w", err)
	}

	return pk.VerifySignature(data, signature), nil
}
