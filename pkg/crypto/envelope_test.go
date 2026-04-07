package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/mlkem"
)

func TestSealOpen(t *testing.T) {
	pub, priv, err := mlkem.GenerateKey(mlkem.MLKEM1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	plaintext := []byte("transfer-agent-secret-data")

	env, err := Seal(plaintext, pub.Bytes())
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	if env.Algorithm != AlgoMLKEM1024_AES256GCM {
		t.Errorf("algorithm = %q, want %q", env.Algorithm, AlgoMLKEM1024_AES256GCM)
	}

	got, err := Open(env, priv.Bytes())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if string(got) != string(plaintext) {
		t.Errorf("open = %q, want %q", got, plaintext)
	}
}

func TestSealOpenWrongKey(t *testing.T) {
	pub, _, err := mlkem.GenerateKey(mlkem.MLKEM1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	env, err := Seal([]byte("secret"), pub.Bytes())
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	// Decrypt with a different key -- must fail.
	_, wrongPriv, err := mlkem.GenerateKey(mlkem.MLKEM1024)
	if err != nil {
		t.Fatalf("generate wrong key: %v", err)
	}

	_, err = Open(env, wrongPriv.Bytes())
	if err == nil {
		t.Fatal("open with wrong key should fail")
	}
}

func TestSealEmpty(t *testing.T) {
	pub, _, err := mlkem.GenerateKey(mlkem.MLKEM1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	_, err = Seal(nil, pub.Bytes())
	if err == nil {
		t.Fatal("seal with nil plaintext should fail")
	}

	_, err = Seal([]byte{}, pub.Bytes())
	if err == nil {
		t.Fatal("seal with empty plaintext should fail")
	}
}

func TestSignVerify(t *testing.T) {
	sk, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA87)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	data := []byte("transfer-record-hash")

	sig, err := Sign(data, sk.Bytes())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	ok, err := Verify(data, sig, sk.PublicKey.Bytes())
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Error("verify returned false, want true")
	}
}

func TestVerifyWrongData(t *testing.T) {
	sk, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA87)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	sig, err := Sign([]byte("original"), sk.Bytes())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	ok, err := Verify([]byte("tampered"), sig, sk.PublicKey.Bytes())
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Error("verify returned true for tampered data")
	}
}
