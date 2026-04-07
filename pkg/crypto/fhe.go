package crypto

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/luxfi/fhe"
)

// FHEKey holds the secret key material for FHE encryption/decryption.
type FHEKey struct {
	params    fhe.Parameters
	intParams *fhe.IntegerParams
	sk        *fhe.SecretKey
	pk        *fhe.PublicKey
}

// FHEEvalKey holds the evaluation keys for homomorphic operations.
type FHEEvalKey struct {
	intParams *fhe.IntegerParams
	intEval   *fhe.IntegerEvaluator
	// sk is required to decrypt comparison results locally.
	// In production MPC this decryption would be distributed.
	sk *fhe.SecretKey
	params fhe.Parameters
}

// FHECiphertext is an encrypted 64-bit integer field value.
type FHECiphertext struct {
	rc *fhe.RadixCiphertext
}

// NewFHEKey generates a new FHE key pair.
func NewFHEKey() (*FHEKey, error) {
	params, err := fhe.NewParametersFromLiteral(fhe.PN10QP27)
	if err != nil {
		return nil, fmt.Errorf("fhe: params: %w", err)
	}

	intParams, err := fhe.NewIntegerParams(params, 2)
	if err != nil {
		return nil, fmt.Errorf("fhe: int params: %w", err)
	}

	kg := fhe.NewKeyGenerator(params)
	sk, pk := kg.GenKeyPair()

	return &FHEKey{
		params:    params,
		intParams: intParams,
		sk:        sk,
		pk:        pk,
	}, nil
}

// NewFHEEvalKey generates an evaluation key from a secret key.
func NewFHEEvalKey(key *FHEKey) (*FHEEvalKey, error) {
	kg := fhe.NewKeyGenerator(key.params)
	bsk := kg.GenBootstrapKey(key.sk)

	intEval := fhe.NewIntegerEvaluator(key.intParams, bsk)

	return &FHEEvalKey{
		intParams: key.intParams,
		intEval:   intEval,
		sk:        key.sk,
		params:    key.params,
	}, nil
}

// EncryptField encrypts a plaintext byte slice as a 64-bit FHE ciphertext.
// Input is interpreted as little-endian uint64 (zero-padded if shorter than 8 bytes).
func EncryptField(plaintext []byte, key *FHEKey) (*FHECiphertext, error) {
	if key == nil || key.sk == nil {
		return nil, errors.New("fhe: nil key")
	}
	if len(plaintext) > 8 {
		return nil, errors.New("fhe: plaintext exceeds 64 bits; use multiple fields")
	}

	buf := make([]byte, 8)
	copy(buf, plaintext)
	val := binary.LittleEndian.Uint64(buf)

	enc := fhe.NewIntegerEncryptor(key.intParams, key.sk)
	rc, err := enc.EncryptUint64(val, fhe.FheUint64)
	if err != nil {
		return nil, fmt.Errorf("fhe: encrypt: %w", err)
	}

	return &FHECiphertext{rc: rc}, nil
}

// DecryptField decrypts an FHE ciphertext back to plaintext bytes (little-endian uint64).
func DecryptField(ct *FHECiphertext, key *FHEKey) ([]byte, error) {
	if ct == nil {
		return nil, errors.New("fhe: nil ciphertext")
	}
	if key == nil || key.sk == nil {
		return nil, errors.New("fhe: nil key")
	}

	dec := fhe.NewIntegerDecryptor(key.intParams, key.sk)
	val := dec.Decrypt64(ct.rc)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, val)
	return buf, nil
}

// CompareEncrypted performs a < b on encrypted values. Returns:
//
//	-1 if a < b, 0 if a == b, 1 if a > b.
//
// The comparison is computed homomorphically, then single-bit results are
// decrypted. In production the final decryption would be done via MPC.
func CompareEncrypted(a, b *FHECiphertext, key *FHEEvalKey) (int, error) {
	if a == nil || b == nil {
		return 0, errors.New("fhe: nil ciphertext")
	}
	if key == nil || key.intEval == nil {
		return 0, errors.New("fhe: nil eval key")
	}

	lt, err := key.intEval.Lt(a.rc, b.rc)
	if err != nil {
		return 0, fmt.Errorf("fhe: lt: %w", err)
	}

	eq, err := key.intEval.Eq(a.rc, b.rc)
	if err != nil {
		return 0, fmt.Errorf("fhe: eq: %w", err)
	}

	dec := fhe.NewIntegerDecryptor(key.intParams, key.sk)
	ltVal := dec.DecryptBool(lt)
	eqVal := dec.DecryptBool(eq)

	if eqVal {
		return 0, nil
	}
	if ltVal {
		return -1, nil
	}
	return 1, nil
}

// SumEncrypted adds a slice of encrypted values homomorphically.
func SumEncrypted(values []*FHECiphertext, key *FHEEvalKey) (*FHECiphertext, error) {
	if len(values) == 0 {
		return nil, errors.New("fhe: empty values")
	}
	if key == nil || key.intEval == nil {
		return nil, errors.New("fhe: nil eval key")
	}

	acc := values[0].rc
	for i := 1; i < len(values); i++ {
		var err error
		acc, err = key.intEval.Add(acc, values[i].rc)
		if err != nil {
			return nil, fmt.Errorf("fhe: sum step %d: %w", i, err)
		}
	}

	return &FHECiphertext{rc: acc}, nil
}
