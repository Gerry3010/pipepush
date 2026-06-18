package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Gerry3010/pipepush/internal/client"
	"github.com/Gerry3010/pipepush/internal/config"
	"github.com/Gerry3010/pipepush/internal/crypto"
	"github.com/Gerry3010/pipepush/internal/models"
)

func newLoginCmd() *cobra.Command {
	var serverFlag, emailFlag string
	var registerFlag bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in (or register) and store a local session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadClientConfig()
			if err != nil {
				return err
			}
			if serverFlag != "" {
				cfg.ServerURL = serverFlag
			}

			email := emailFlag
			if email == "" {
				email, err = promptLine("Email: ")
				if err != nil {
					return err
				}
			}
			password, err := promptPassword("Password: ")
			if err != nil {
				return err
			}

			api := client.New(cfg.ServerURL, "")
			ctx := context.Background()

			var resp *models.LoginResponse
			if registerFlag {
				resp, err = doRegister(ctx, api, email, password)
			} else {
				resp, err = api.Login(ctx, email, password)
			}
			if err != nil {
				return err
			}

			// Decrypt the private key locally with the password and cache it.
			privBytes, err := crypto.DecryptPrivateKey(resp.EncryptedPrivateKey, resp.KDFSalt, password)
			if err != nil {
				return fmt.Errorf("could not unlock private key: %w", err)
			}
			priv, err := crypto.PrivateKeyFromBytes(privBytes)
			if err != nil {
				return err
			}

			cfg.JWT = resp.JWT
			cfg.Email = email
			cfg.PublicKey = resp.PublicKey
			cfg.PrivateKey = crypto.PrivateKeyToBase64(priv)
			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("✓ Logged in as %s (server: %s)\n", email, cfg.ServerURL)
			return nil
		},
	}

	cmd.Flags().StringVar(&serverFlag, "server", "", "Server URL (overrides config)")
	cmd.Flags().StringVar(&emailFlag, "email", "", "Email address")
	cmd.Flags().BoolVar(&registerFlag, "register", false, "Register a new account instead of logging in")
	return cmd
}

// doRegister generates a keypair, encrypts the private key with the password,
// and registers the account.
func doRegister(ctx context.Context, api *client.Client, email, password string) (*models.LoginResponse, error) {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	encPriv, salt, err := crypto.EncryptPrivateKey(kp.PrivateKey.Bytes(), password)
	if err != nil {
		return nil, err
	}

	return api.Register(ctx, models.RegisterRequest{
		Email:               email,
		Password:            password,
		PublicKey:           crypto.PublicKeyToBase64(kp.PublicKey),
		EncryptedPrivateKey: encPriv,
		KDFSalt:             salt,
	})
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the local session and private key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadClientConfig()
			if err != nil {
				return err
			}
			cfg.Logout()
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Println("✓ Logged out")
			return nil
		},
	}
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadClientConfig()
			if err != nil {
				return err
			}
			if !cfg.IsLoggedIn() {
				fmt.Println("Not logged in")
				return nil
			}
			fmt.Printf("Email:  %s\nServer: %s\n", cfg.Email, cfg.ServerURL)
			return nil
		},
	}
}

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the server URL",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "set <url>",
		Short: "Set the server URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadClientConfig()
			if err != nil {
				return err
			}
			cfg.ServerURL = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("✓ Server set to %s\n", args[0])
			return nil
		},
	})
	return cmd
}
