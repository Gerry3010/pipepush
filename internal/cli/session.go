// Package cli implements the pipepush command-line interface.
package cli

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

// loadSession loads config and constructs an authenticated session.
// Returns an error if the user is not logged in.
func loadSession() (*Session, error) {
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

// encrypt encrypts plaintext with the user's own public key (self-addressed E2E).
func (s *Session) encrypt(plaintext string) (string, error) {
	return crypto.EncryptString(s.pub, plaintext)
}

// decrypt decrypts a ciphertext with the user's private key.
// On failure it returns a placeholder so listings stay usable.
func (s *Session) decrypt(ciphertext string) string {
	plain, err := crypto.DecryptString(s.priv, ciphertext)
	if err != nil {
		return "<decryption failed>"
	}
	return plain
}
