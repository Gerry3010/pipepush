// Package cli implements the pipepush command-line interface.
package cli

import "github.com/Gerry3010/pipepush/internal/session"

// Session and loadSession are kept as package-local aliases over internal/session
// so the CLI command handlers read unchanged. The encryption/session logic lives
// in internal/session, shared with the MCP server (cmd/pipepush-mcp).
type Session = session.Session

func loadSession() (*Session, error) { return session.Load() }
