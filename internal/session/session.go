// Package session bundles a loaded CLI config, an authenticated API client, and
// the in-memory X25519 key pair used for client-side encryption/decryption.
//
// It is shared by the interactive CLI (internal/cli) and the MCP server
// (cmd/pipepush-mcp) so both encrypt/decrypt project, pipeline, token and run
// details identically. The server never sees plaintext — E2E holds.
package session

import (
	"crypto/ecdh"
	"fmt"

	"github.com/Gerry3010/pipepush/internal/client"
	"github.com/Gerry3010/pipepush/internal/config"
	"github.com/Gerry3010/pipepush/internal/crypto"
)

// Session bundles the loaded config, an authenticated API client, and the
// in-memory private key for client-side encryption/decryption.
type Session struct {
	Cfg  *config.ClientConfig
	API  *client.Client
	priv *ecdh.PrivateKey
	pub  *ecdh.PublicKey
}

// Load loads config and constructs an authenticated session.
// Returns an error if the user is not logged in.
func Load() (*Session, error) {
	cfg, err := config.LoadClientConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.IsLoggedIn() {
		return nil, fmt.Errorf("not logged in — run 'pipepush login' first")
	}

	priv, err := crypto.PrivateKeyFromBase64(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid local key — try logging in again: %w", err)
	}
	pub, err := crypto.PublicKeyFromBase64(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid local public key — try logging in again: %w", err)
	}

	return &Session{
		Cfg:  cfg,
		API:  client.New(cfg.ServerURL, cfg.JWT),
		priv: priv,
		pub:  pub,
	}, nil
}

// Encrypt encrypts plaintext with the user's own public key (self-addressed E2E).
func (s *Session) Encrypt(plaintext string) (string, error) {
	return crypto.EncryptString(s.pub, plaintext)
}

// Decrypt decrypts a ciphertext with the user's private key.
// On failure it returns a placeholder so listings stay usable.
func (s *Session) Decrypt(ciphertext string) string {
	plain, err := crypto.DecryptString(s.priv, ciphertext)
	if err != nil {
		return "<decryption failed>"
	}
	return plain
}

// DecryptPayload decrypts a ciphertext and returns the error, so callers that
// unmarshal (e.g. run payloads) can distinguish a genuine failure from the
// placeholder text Decrypt returns.
func (s *Session) DecryptPayload(ciphertext string) (string, error) {
	return crypto.DecryptString(s.priv, ciphertext)
}
