// Biometric (Face ID / Touch ID / Windows Hello) unlock for this device.
//
// The E2E private key normally lives only in memory and is re-derived from the
// password on every app start. That is safe but tedious on a phone. Here we let
// a platform authenticator stand in for the password on THIS device:
//
//   • Enrollment (once, right after a password login): create a platform
//     passkey with the WebAuthn PRF extension. PRF gives us a stable 32-byte
//     secret bound to the credential + a salt — only retrievable after a
//     successful biometric check. We derive an AES key from it and wrap the
//     private key, storing the wrapped blob locally.
//   • Unlock: a biometric prompt returns the same PRF secret, we unwrap the
//     private key and restore the session. No password, no server round-trip.
//
// The private key is never stored in the clear, and the PRF secret never leaves
// the authenticator's control — a stolen device without the biometric can't
// unwrap it. Purely local: no backend, no DB. Falls back to password anywhere
// PRF isn't supported (older Safari/Chrome, no platform authenticator).

import { hkdf } from "@noble/hashes/hkdf";
import { sha256 } from "@noble/hashes/sha256";
import { randomBytes } from "@noble/hashes/utils";
import { b64urlEncode, b64urlDecode } from "./ecies";

const STORE_KEY = "pp_bio";
const PRF_INFO = new TextEncoder().encode("pipepush-prf-wrap-v1");

interface BioRecord {
  credentialId: string; // base64url
  prfSalt: string; // base64url
  wrapped: string; // base64url: nonce(12) || AES-GCM ciphertext of the private key
  pub: string; // base64url public key (not secret; needed to restore the session)
  email: string;
}

function toArrayBuffer(u: Uint8Array): ArrayBuffer {
  return u.buffer.slice(u.byteOffset, u.byteOffset + u.byteLength) as ArrayBuffer;
}

function loadRecord(): BioRecord | null {
  try {
    const raw = localStorage.getItem(STORE_KEY);
    return raw ? (JSON.parse(raw) as BioRecord) : null;
  } catch {
    return null;
  }
}

// biometricSupported reports the static prerequisites (secure context + API).
export function biometricSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    window.isSecureContext &&
    typeof window.PublicKeyCredential !== "undefined" &&
    !!navigator.credentials
  );
}

// biometricAvailable additionally confirms a platform authenticator exists
// (Face ID / Touch ID / Hello) so we only offer enrollment when it can work.
export async function biometricAvailable(): Promise<boolean> {
  if (!biometricSupported()) return false;
  try {
    return await window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
  } catch {
    return false;
  }
}

// hasBiometricUnlock reports whether THIS device is enrolled for the given user.
export function hasBiometricUnlock(email?: string | null): boolean {
  const rec = loadRecord();
  if (!rec) return false;
  return !email || rec.email === email;
}

export function clearBiometricUnlock(): void {
  localStorage.removeItem(STORE_KEY);
}

// Pull the PRF secret out of a WebAuthn credential's extension results.
function prfSecret(cred: PublicKeyCredential): Uint8Array | null {
  const ext = cred.getClientExtensionResults() as {
    prf?: { results?: { first?: ArrayBuffer } };
  };
  const first = ext.prf?.results?.first;
  return first ? new Uint8Array(first) : null;
}

async function aesKeyFromPRF(secret: Uint8Array, usage: KeyUsage): Promise<CryptoKey> {
  const keyBytes = hkdf(sha256, secret, undefined, PRF_INFO, 32);
  return crypto.subtle.importKey("raw", toArrayBuffer(keyBytes), "AES-GCM", false, [usage]);
}

// enrollBiometric wraps the (already unlocked) private key behind a platform
// passkey. Requires the private key currently in memory. Throws if PRF is
// unsupported by the authenticator (caller should keep the password path).
export async function enrollBiometric(
  privateKey: Uint8Array,
  pub: string,
  email: string,
): Promise<void> {
  if (!biometricSupported()) throw new Error("Biometric unlock isn't supported on this device");

  const prfSalt = randomBytes(32);
  const cred = (await navigator.credentials.create({
    publicKey: {
      challenge: toArrayBuffer(randomBytes(32)),
      rp: { name: "pipepush", id: location.hostname },
      user: {
        id: toArrayBuffer(randomBytes(16)),
        name: email,
        displayName: email,
      },
      pubKeyCredParams: [
        { type: "public-key", alg: -7 },
        { type: "public-key", alg: -257 },
      ],
      authenticatorSelection: {
        authenticatorAttachment: "platform",
        userVerification: "required",
        residentKey: "required",
      },
      timeout: 60000,
      extensions: { prf: { eval: { first: toArrayBuffer(prfSalt) } } } as AuthenticationExtensionsClientInputs,
    },
  })) as PublicKeyCredential | null;

  if (!cred) throw new Error("Biometric setup was cancelled");
  const credentialId = new Uint8Array(cred.rawId);

  // Some platforms (Safari 18+) return the PRF result straight from create();
  // others (Chrome) only report support and need a follow-up get() to yield it.
  let secret = prfSecret(cred);
  if (!secret) {
    const enabled =
      (cred.getClientExtensionResults() as { prf?: { enabled?: boolean } }).prf?.enabled;
    if (enabled === false) {
      clearBiometricUnlock();
      throw new Error("This device's authenticator doesn't support PRF-based unlock");
    }
    const asserted = (await navigator.credentials.get({
      publicKey: {
        challenge: toArrayBuffer(randomBytes(32)),
        rpId: location.hostname,
        allowCredentials: [{ type: "public-key", id: toArrayBuffer(credentialId) }],
        userVerification: "required",
        timeout: 60000,
        extensions: { prf: { eval: { first: toArrayBuffer(prfSalt) } } } as AuthenticationExtensionsClientInputs,
      },
    })) as PublicKeyCredential | null;
    secret = asserted ? prfSecret(asserted) : null;
  }
  if (!secret) throw new Error("This device's authenticator doesn't support PRF-based unlock");

  const key = await aesKeyFromPRF(secret, "encrypt");
  const nonce = randomBytes(12);
  const ct = new Uint8Array(
    await crypto.subtle.encrypt(
      { name: "AES-GCM", iv: toArrayBuffer(nonce) },
      key,
      toArrayBuffer(privateKey),
    ),
  );
  const blob = new Uint8Array(nonce.length + ct.length);
  blob.set(nonce, 0);
  blob.set(ct, nonce.length);

  const rec: BioRecord = {
    credentialId: b64urlEncode(credentialId),
    prfSalt: b64urlEncode(prfSalt),
    wrapped: b64urlEncode(blob),
    pub,
    email,
  };
  localStorage.setItem(STORE_KEY, JSON.stringify(rec));
}

// unlockWithBiometric prompts for the biometric and returns the restored key
// material. Throws (and leaves enrollment intact) if the user cancels or fails.
export async function unlockWithBiometric(): Promise<{
  privateKey: Uint8Array;
  pub: string;
  email: string;
}> {
  const rec = loadRecord();
  if (!rec) throw new Error("No biometric unlock is set up on this device");
  if (!biometricSupported()) throw new Error("Biometric unlock isn't supported on this device");

  const credentialId = b64urlDecode(rec.credentialId);
  const prfSalt = b64urlDecode(rec.prfSalt);

  const asserted = (await navigator.credentials.get({
    publicKey: {
      challenge: toArrayBuffer(randomBytes(32)),
      rpId: location.hostname,
      allowCredentials: [{ type: "public-key", id: toArrayBuffer(credentialId) }],
      userVerification: "required",
      timeout: 60000,
      extensions: { prf: { eval: { first: toArrayBuffer(prfSalt) } } } as AuthenticationExtensionsClientInputs,
    },
  })) as PublicKeyCredential | null;

  const secret = asserted ? prfSecret(asserted) : null;
  if (!secret) throw new Error("Biometric unlock failed — use your password");

  const key = await aesKeyFromPRF(secret, "decrypt");
  const blob = b64urlDecode(rec.wrapped);
  const nonce = blob.slice(0, 12);
  const ct = blob.slice(12);
  const privateKey = new Uint8Array(
    await crypto.subtle.decrypt({ name: "AES-GCM", iv: toArrayBuffer(nonce) }, key, toArrayBuffer(ct)),
  );
  return { privateKey, pub: rec.pub, email: rec.email };
}

export function biometricEmail(): string | null {
  return loadRecord()?.email ?? null;
}
