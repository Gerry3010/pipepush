// Browser-side crypto that is wire-compatible with the Go implementation in
// internal/crypto. ECIES = ephemeral X25519 + HKDF-SHA256 + XChaCha20-Poly1305.
//
//   wire = ephemeralPub(32) || nonce(24) || ciphertext+tag
//   symKey = HKDF-SHA256(secret=shared, salt="pipepush-ecies-v1",
//                        info=ephemeralPub||recipientPub, len=32)

import { x25519 } from "@noble/curves/ed25519";
import { xchacha20poly1305 } from "@noble/ciphers/chacha";
import { hkdf } from "@noble/hashes/hkdf";
import { sha256 } from "@noble/hashes/sha256";
import { argon2id } from "@noble/hashes/argon2";
import { randomBytes } from "@noble/hashes/utils";

const HKDF_SALT = new TextEncoder().encode("pipepush-ecies-v1");

// Coerce a Uint8Array (possibly backed by ArrayBufferLike/SharedArrayBuffer) to
// a plain ArrayBuffer, which WebCrypto/PushManager require under TS 5.7+.
function toArrayBuffer(u: Uint8Array): ArrayBuffer {
  return u.buffer.slice(u.byteOffset, u.byteOffset + u.byteLength) as ArrayBuffer;
}

// --- base64url (no padding), matching Go's base64.RawURLEncoding ---

export function b64urlEncode(bytes: Uint8Array): string {
  let s = btoa(String.fromCharCode(...bytes));
  return s.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function b64urlDecode(str: string): Uint8Array {
  const s = str.replace(/-/g, "+").replace(/_/g, "/");
  const pad = s.length % 4 === 0 ? "" : "=".repeat(4 - (s.length % 4));
  const bin = atob(s + pad);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

function deriveSymKey(
  shared: Uint8Array,
  ephemeralPub: Uint8Array,
  recipientPub: Uint8Array
): Uint8Array {
  const info = new Uint8Array(ephemeralPub.length + recipientPub.length);
  info.set(ephemeralPub, 0);
  info.set(recipientPub, ephemeralPub.length);
  return hkdf(sha256, shared, HKDF_SALT, info, 32);
}

// Encrypt plaintext for a recipient public key (base64url). Returns base64url.
export function eciesEncrypt(recipientPubB64: string, plaintext: string): string {
  const recipientPub = b64urlDecode(recipientPubB64);
  const ephPriv = x25519.utils.randomPrivateKey();
  const ephPub = x25519.getPublicKey(ephPriv);
  const shared = x25519.getSharedSecret(ephPriv, recipientPub);

  const symKey = deriveSymKey(shared, ephPub, recipientPub);
  const nonce = randomBytes(24);
  const aead = xchacha20poly1305(symKey, nonce);
  const ct = aead.encrypt(new TextEncoder().encode(plaintext));

  const wire = new Uint8Array(32 + 24 + ct.length);
  wire.set(ephPub, 0);
  wire.set(nonce, 32);
  wire.set(ct, 56);
  return b64urlEncode(wire);
}

// Decrypt a base64url ciphertext with the recipient private key (raw bytes).
export function eciesDecrypt(recipientPriv: Uint8Array, ciphertextB64: string): string {
  const wire = b64urlDecode(ciphertextB64);
  const ephPub = wire.slice(0, 32);
  const nonce = wire.slice(32, 56);
  const ct = wire.slice(56);

  const shared = x25519.getSharedSecret(recipientPriv, ephPub);
  const recipientPub = x25519.getPublicKey(recipientPriv);
  const symKey = deriveSymKey(shared, ephPub, recipientPub);

  const aead = xchacha20poly1305(symKey, nonce);
  const pt = aead.decrypt(ct);
  return new TextDecoder().decode(pt);
}

// Try to decrypt; return a placeholder on failure so listings stay usable.
export function eciesDecryptSafe(recipientPriv: Uint8Array, ciphertextB64: string): string {
  try {
    return eciesDecrypt(recipientPriv, ciphertextB64);
  } catch {
    return "<decryption failed>";
  }
}

// --- Key generation + password-based private key wrapping (matches Go) ---

export interface Keypair {
  publicKeyB64: string;
  privateKey: Uint8Array;
}

export function generateKeypair(): Keypair {
  const priv = x25519.utils.randomPrivateKey();
  const pub = x25519.getPublicKey(priv);
  return { publicKeyB64: b64urlEncode(pub), privateKey: priv };
}

// Argon2id params must match Go: time=3, memory=64MiB, threads=4, keyLen=32.
function deriveKDFKey(password: string, salt: Uint8Array): Uint8Array {
  return argon2id(new TextEncoder().encode(password), salt, {
    t: 3,
    m: 64 * 1024,
    p: 4,
    dkLen: 32,
  });
}

// Encrypt the private key with AES-256-GCM (12-byte nonce prepended), matching
// Go's EncryptPrivateKey. Returns { encryptedPrivateKey, kdfSalt } as base64url.
export async function wrapPrivateKey(
  privateKey: Uint8Array,
  password: string
): Promise<{ encryptedPrivateKey: string; kdfSalt: string }> {
  const salt = randomBytes(32);
  const keyBytes = deriveKDFKey(password, salt);
  const key = await crypto.subtle.importKey("raw", toArrayBuffer(keyBytes), "AES-GCM", false, [
    "encrypt",
  ]);
  const nonce = randomBytes(12);
  const ct = new Uint8Array(
    await crypto.subtle.encrypt(
      { name: "AES-GCM", iv: toArrayBuffer(nonce) },
      key,
      toArrayBuffer(privateKey)
    )
  );
  const combined = new Uint8Array(nonce.length + ct.length);
  combined.set(nonce, 0);
  combined.set(ct, nonce.length);
  return { encryptedPrivateKey: b64urlEncode(combined), kdfSalt: b64urlEncode(salt) };
}

// Decrypt the wrapped private key with the password (matches Go's DecryptPrivateKey).
export async function unwrapPrivateKey(
  encryptedPrivateKeyB64: string,
  kdfSaltB64: string,
  password: string
): Promise<Uint8Array> {
  const combined = b64urlDecode(encryptedPrivateKeyB64);
  const salt = b64urlDecode(kdfSaltB64);
  const nonce = combined.slice(0, 12);
  const ct = combined.slice(12);
  const keyBytes = deriveKDFKey(password, salt);
  const key = await crypto.subtle.importKey("raw", toArrayBuffer(keyBytes), "AES-GCM", false, [
    "decrypt",
  ]);
  const pt = new Uint8Array(
    await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: toArrayBuffer(nonce) },
      key,
      toArrayBuffer(ct)
    )
  );
  return pt;
}
