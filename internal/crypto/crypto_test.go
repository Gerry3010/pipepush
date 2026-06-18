package crypto_test

import (
	"testing"

	"github.com/Gerry3010/pipepush/internal/crypto"
)

func TestECIESRoundTrip(t *testing.T) {
	alice, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := "Hello, pipepush! Status: success, branch: main, commit: abc123def456"

	ciphertext, err := crypto.EncryptString(alice.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := crypto.DecryptString(alice.PrivateKey, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestECIESDifferentKeyFails(t *testing.T) {
	alice, _ := crypto.GenerateKeyPair()
	bob, _ := crypto.GenerateKeyPair()

	ciphertext, _ := crypto.EncryptString(alice.PublicKey, "secret")
	_, err := crypto.DecryptString(bob.PrivateKey, ciphertext)
	if err == nil {
		t.Error("expected decryption with wrong key to fail")
	}
}

func TestKeySerializationRoundTrip(t *testing.T) {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	pubB64 := crypto.PublicKeyToBase64(kp.PublicKey)
	privB64 := crypto.PrivateKeyToBase64(kp.PrivateKey)

	pub2, err := crypto.PublicKeyFromBase64(pubB64)
	if err != nil {
		t.Fatalf("parsing public key: %v", err)
	}

	priv2, err := crypto.PrivateKeyFromBase64(privB64)
	if err != nil {
		t.Fatalf("parsing private key: %v", err)
	}

	// Encrypt with original public key, decrypt with restored private key
	ct, err := crypto.EncryptString(kp.PublicKey, "test")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := crypto.DecryptString(priv2, ct)
	if err != nil {
		t.Fatalf("decrypt with restored key: %v", err)
	}
	if pt != "test" {
		t.Error("round-trip failed")
	}

	_ = pub2
}

func TestKDFPrivateKeyRoundTrip(t *testing.T) {
	kp, _ := crypto.GenerateKeyPair()
	privBytes := kp.PrivateKey.Bytes()
	password := "hunter2-super-secret"

	encrypted, salt, err := crypto.EncryptPrivateKey(privBytes, password)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptPrivateKey(encrypted, salt, password)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(privBytes) {
		t.Error("private key round-trip failed")
	}
}

func TestKDFWrongPasswordFails(t *testing.T) {
	kp, _ := crypto.GenerateKeyPair()
	encrypted, salt, _ := crypto.EncryptPrivateKey(kp.PrivateKey.Bytes(), "correct-password")

	_, err := crypto.DecryptPrivateKey(encrypted, salt, "wrong-password")
	if err == nil {
		t.Error("expected decryption with wrong password to fail")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	kp, _ := crypto.GenerateKeyPair()
	msg := []byte(`{"status":"success","pipeline":"deploy","branch":"main","commit":"abc123"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = crypto.Encrypt(kp.PublicKey, msg)
	}
}

func BenchmarkKDF(b *testing.B) {
	kp, _ := crypto.GenerateKeyPair()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = crypto.EncryptPrivateKey(kp.PrivateKey.Bytes(), "password")
	}
}
