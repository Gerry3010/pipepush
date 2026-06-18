// Cross-language interop harness for the ECIES scheme.
// Mirrors src/crypto/ecies.ts exactly, runnable in Node for testing against Go.
//
// Usage:
//   node interop.mjs gen                       -> {privB64, pubB64, plaintext, ciphertext}
//   node interop.mjs dec <privB64> <ciphertext> -> prints decrypted plaintext

import { x25519 } from "@noble/curves/ed25519";
import { xchacha20poly1305 } from "@noble/ciphers/chacha";
import { hkdf } from "@noble/hashes/hkdf";
import { sha256 } from "@noble/hashes/sha256";
import { argon2id } from "@noble/hashes/argon2";
import { randomBytes } from "@noble/hashes/utils";
import { webcrypto } from "node:crypto";

const HKDF_SALT = new TextEncoder().encode("pipepush-ecies-v1");

function b64urlEncode(b) {
  return Buffer.from(b).toString("base64url");
}
function b64urlDecode(s) {
  return new Uint8Array(Buffer.from(s, "base64url"));
}

function deriveSymKey(shared, ephPub, recipientPub) {
  const info = new Uint8Array(ephPub.length + recipientPub.length);
  info.set(ephPub, 0);
  info.set(recipientPub, ephPub.length);
  return hkdf(sha256, shared, HKDF_SALT, info, 32);
}

function encrypt(recipientPubB64, plaintext) {
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

function decrypt(recipientPriv, ciphertextB64) {
  const wire = b64urlDecode(ciphertextB64);
  const ephPub = wire.slice(0, 32);
  const nonce = wire.slice(32, 56);
  const ct = wire.slice(56);
  const shared = x25519.getSharedSecret(recipientPriv, ephPub);
  const recipientPub = x25519.getPublicKey(recipientPriv);
  const symKey = deriveSymKey(shared, ephPub, recipientPub);
  const aead = xchacha20poly1305(symKey, nonce);
  return new TextDecoder().decode(aead.decrypt(ct));
}

function deriveKDFKey(password, salt) {
  return argon2id(new TextEncoder().encode(password), salt, {
    t: 3,
    m: 64 * 1024,
    p: 4,
    dkLen: 32,
  });
}

async function wrapPrivateKey(privateKey, password) {
  const salt = randomBytes(32);
  const keyBytes = deriveKDFKey(password, salt);
  const key = await webcrypto.subtle.importKey("raw", keyBytes, "AES-GCM", false, ["encrypt"]);
  const nonce = randomBytes(12);
  const ct = new Uint8Array(
    await webcrypto.subtle.encrypt({ name: "AES-GCM", iv: nonce }, key, privateKey)
  );
  const combined = new Uint8Array(nonce.length + ct.length);
  combined.set(nonce, 0);
  combined.set(ct, nonce.length);
  return { encB64: b64urlEncode(combined), saltB64: b64urlEncode(salt) };
}

async function unwrapPrivateKey(encB64, saltB64, password) {
  const combined = b64urlDecode(encB64);
  const salt = b64urlDecode(saltB64);
  const nonce = combined.slice(0, 12);
  const ct = combined.slice(12);
  const keyBytes = deriveKDFKey(password, salt);
  const key = await webcrypto.subtle.importKey("raw", keyBytes, "AES-GCM", false, ["decrypt"]);
  return new Uint8Array(await webcrypto.subtle.decrypt({ name: "AES-GCM", iv: nonce }, key, ct));
}

const cmd = process.argv[2];
if (cmd === "gen") {
  const priv = x25519.utils.randomPrivateKey();
  const pub = x25519.getPublicKey(priv);
  const plaintext = "interop-test: ✓ success on main @ abc123";
  const ciphertext = encrypt(b64urlEncode(pub), plaintext);
  process.stdout.write(
    JSON.stringify({
      privB64: b64urlEncode(priv),
      pubB64: b64urlEncode(pub),
      plaintext,
      ciphertext,
    })
  );
} else if (cmd === "dec") {
  const priv = b64urlDecode(process.argv[3]);
  process.stdout.write(decrypt(priv, process.argv[4]));
} else if (cmd === "wrap") {
  // wrap <password> <privB64> -> {encB64, saltB64}
  const password = process.argv[3];
  const priv = b64urlDecode(process.argv[4]);
  const wrapped = await wrapPrivateKey(priv, password);
  process.stdout.write(JSON.stringify(wrapped));
} else if (cmd === "unwrap") {
  // unwrap <password> <encB64> <saltB64> -> privB64
  const priv = await unwrapPrivateKey(process.argv[4], process.argv[5], process.argv[3]);
  process.stdout.write(b64urlEncode(priv));
} else {
  console.error("unknown command");
  process.exit(1);
}
