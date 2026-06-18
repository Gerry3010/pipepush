# Architecture & Security

## Overview

```
┌─────────────┐   POST /api/webhook        ┌──────────────────────┐
│  CI/CD step │ ─────(token + status)────▶ │   pipepush-server    │
└─────────────┘                            │                      │
                                           │  1. SHA-256(token) →  │
                                           │     look up user      │
┌─────────────┐   SSE / Web Push           │  2. ECIES-encrypt the │
│ CLI / Web /  │ ◀───(encrypted blob)────── │     payload with the  │
│  phone       │                            │     user's PUBLIC key │
└─────────────┘                            │  3. store + dispatch  │
      │ decrypt with private key            └──────────┬───────────┘
      ▼ (only on device)                               │
  plaintext run details                          PostgreSQL (ciphertext only)
```

## End-to-end encryption

Every user has an **X25519 key pair**, generated on the client at registration.

- **Public key** → stored on the server.
- **Private key** → wrapped with **AES-256-GCM** under a key derived from the
  user's password via **Argon2id** (t=3, m=64 MiB, p=4, 32-byte key), then stored
  on the server as an opaque blob. The server cannot unwrap it.

### Encrypting a run (server side, webhook)

The server never sees plaintext run details for storage. It encrypts the incoming
payload **to the user's public key** using ECIES:

1. Generate an ephemeral X25519 key pair.
2. `shared = ECDH(ephemeral_priv, user_pub)`.
3. `symKey = HKDF-SHA256(secret=shared, salt="pipepush-ecies-v1", info=ephemeral_pub‖user_pub, len=32)`.
4. `ciphertext = XChaCha20-Poly1305(symKey, nonce, payload)` with a random 24-byte nonce.
5. Wire format (base64url): `ephemeral_pub(32) ‖ nonce(24) ‖ ciphertext+tag`.

Only the holder of the private key can derive the same `symKey` and decrypt.

### User-authored data (project / pipeline / token names)

Encrypted client-side **to the user's own public key** before upload, using the
same ECIES scheme, and decrypted on display. The server stores ciphertext.

### Cross-client compatibility

The Go (`internal/crypto`) and browser (`web/src/crypto`, `@noble/*`)
implementations are byte-for-byte wire compatible — both the ECIES scheme and the
Argon2id+AES-GCM key wrapping. This is enforced by an interop test
(`PIPEPUSH_INTEROP=1 go test ./internal/crypto -run Interop`), so you can register
on the CLI and log in on the web (or vice versa).

## What the server / operator can and cannot see

| Can see                                   | Cannot see                                  |
| ----------------------------------------- | ------------------------------------------- |
| Your email                                | Your password (only a bcrypt hash is stored) |
| Public key, wrapped private key blob      | Your private key                            |
| Token **hashes** (SHA-256)                | Plaintext tokens                            |
| Run **status** (success/failure/…) + time | Pipeline names, branches, commits, messages |
| FCM/Web-Push endpoints (delivery routing) | Decrypted notification contents             |

> Status and timing are stored in plaintext so the server can route and the UI
> can color runs without a round-trip. If you need those hidden too, that's a
> future enhancement (store status inside the encrypted blob only).

## Tokens

- Created client-side request; the server generates a 32-byte random token,
  returns it **once**, and stores only `SHA-256(token)`.
- A token is bound to a project and (to receive runs) a pipeline.
- Revoking flips `active=false`; the hash lookup ignores inactive tokens.

## Trust boundaries / caveats

- **Local key storage.** CLI caches the decrypted private key in
  `~/.config/pipepush/config.json` (0600); the browser keeps it in memory only
  (re-enter password after reload). Protect the CLI config like an SSH key.
- **Web Push metadata.** The push service (browser vendor) sees endpoint + timing;
  the payload it carries is the same E2E ciphertext.
- **Password = data.** Lose the password and the wrapped private key can't be
  unwrapped → historical encrypted runs become unreadable. (A recovery-key escrow
  is a possible future feature.)
