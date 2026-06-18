package crypto_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gerry3010/pipepush/internal/crypto"
)

// TestInteropWithNode verifies the Go and browser (noble) ECIES implementations
// are wire-compatible in both directions.
//
// Enable with: PIPEPUSH_INTEROP=1 go test ./internal/crypto/ -run Interop -v
// Requires Node and `npm install` in ../../web.
func TestInteropWithNode(t *testing.T) {
	if os.Getenv("PIPEPUSH_INTEROP") == "" {
		t.Skip("set PIPEPUSH_INTEROP=1 to run the Node interop test")
	}

	webDir, err := filepath.Abs("../../web")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(webDir, "interop.mjs")

	// 1. Node encrypts -> Go decrypts.
	out, err := exec.Command("node", script, "gen").Output()
	if err != nil {
		t.Fatalf("node gen: %v", err)
	}
	var gen struct {
		PrivB64    string `json:"privB64"`
		PubB64     string `json:"pubB64"`
		Plaintext  string `json:"plaintext"`
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.Unmarshal(out, &gen); err != nil {
		t.Fatalf("parsing node output: %v", err)
	}

	priv, err := crypto.PrivateKeyFromBase64(gen.PrivB64)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := crypto.DecryptString(priv, gen.Ciphertext)
	if err != nil {
		t.Fatalf("Go failed to decrypt Node ciphertext: %v", err)
	}
	if decrypted != gen.Plaintext {
		t.Errorf("Node->Go mismatch: got %q want %q", decrypted, gen.Plaintext)
	}
	t.Logf("Node->Go OK: %q", decrypted)

	// 2. Go encrypts -> Node decrypts.
	pub, err := crypto.PublicKeyFromBase64(gen.PubB64)
	if err != nil {
		t.Fatal(err)
	}
	goPlaintext := "from-go: ✗ failure on feature/x @ cafe9999"
	goCipher, err := crypto.EncryptString(pub, goPlaintext)
	if err != nil {
		t.Fatal(err)
	}
	nodeOut, err := exec.Command("node", script, "dec", gen.PrivB64, goCipher).Output()
	if err != nil {
		t.Fatalf("node dec: %v", err)
	}
	if got := strings.TrimSpace(string(nodeOut)); got != goPlaintext {
		t.Errorf("Go->Node mismatch: got %q want %q", got, goPlaintext)
	}
	t.Logf("Go->Node OK: %q", goPlaintext)

	// 3. Private-key wrapping interop (cross-client login).
	password := "shared-password-1234"
	kp, _ := crypto.GenerateKeyPair()
	privB64 := crypto.PrivateKeyToBase64(kp.PrivateKey)

	// 3a. Go wraps (CLI register) -> Node unwraps (web login).
	goEnc, goSalt, err := crypto.EncryptPrivateKey(kp.PrivateKey.Bytes(), password)
	if err != nil {
		t.Fatal(err)
	}
	nodeUnwrap, err := exec.Command("node", script, "unwrap", password, goEnc, goSalt).Output()
	if err != nil {
		t.Fatalf("node unwrap: %v", err)
	}
	if got := strings.TrimSpace(string(nodeUnwrap)); got != privB64 {
		t.Errorf("Go-wrap->Node-unwrap mismatch: got %q want %q", got, privB64)
	}
	t.Log("Go-wrap -> Node-unwrap OK (CLI register, web login)")

	// 3b. Node wraps (web register) -> Go unwraps (CLI login).
	wrapOut, err := exec.Command("node", script, "wrap", password, privB64).Output()
	if err != nil {
		t.Fatalf("node wrap: %v", err)
	}
	var wrapped struct {
		EncB64  string `json:"encB64"`
		SaltB64 string `json:"saltB64"`
	}
	if err := json.Unmarshal(wrapOut, &wrapped); err != nil {
		t.Fatal(err)
	}
	goUnwrap, err := crypto.DecryptPrivateKey(wrapped.EncB64, wrapped.SaltB64, password)
	if err != nil {
		t.Fatalf("Go failed to unwrap Node-wrapped key: %v", err)
	}
	if crypto.PrivateKeyBytesToBase64(goUnwrap) != privB64 {
		t.Error("Node-wrap->Go-unwrap mismatch")
	}
	t.Log("Node-wrap -> Go-unwrap OK (web register, CLI login)")
}
