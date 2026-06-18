package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const defaultServerURL = "https://pipepush.io"

// ClientConfig holds CLI configuration persisted locally.
//
// SECURITY NOTE: After login the decrypted X25519 private key is stored here in
// the config file with 0600 permissions (like an SSH key or a cached bearer
// token). This is the trade-off that lets every CLI command decrypt run details
// without re-prompting for the password. The *server* still never sees the
// plaintext key — E2E holds. Treat ~/.config/pipepush/config.json as a secret.
type ClientConfig struct {
	ServerURL string `json:"serverURL"`
	JWT       string `json:"jwt"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`

	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"` // base64url X25519 private key (sensitive)
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding config dir: %w", err)
	}
	return filepath.Join(dir, "pipepush", "config.json"), nil
}

// LoadClientConfig loads the CLI config from disk.
func LoadClientConfig() (*ClientConfig, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &ClientConfig{ServerURL: defaultServerURL}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = defaultServerURL
	}
	return &cfg, nil
}

// Save persists the config to disk.
func (c *ClientConfig) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// IsLoggedIn returns true if the config has a JWT token.
func (c *ClientConfig) IsLoggedIn() bool {
	return c.JWT != ""
}

// Logout clears authentication data and the local private key.
func (c *ClientConfig) Logout() {
	c.JWT = ""
	c.UserID = ""
	c.Email = ""
	c.PublicKey = ""
	c.PrivateKey = ""
}
