// In-memory session holding the decrypted private key and public key.
// The private key is never persisted in the browser — on reload the user
// re-enters their password. The JWT (a bearer cred) is kept in localStorage.

import { eciesDecryptSafe, eciesEncrypt, b64urlEncode } from "./ecies";

interface SessionState {
  privateKey: Uint8Array | null;
  publicKeyB64: string | null;
  email: string | null;
}

const state: SessionState = {
  privateKey: null,
  publicKeyB64: null,
  email: null,
};

export function setSession(privateKey: Uint8Array, publicKeyB64: string, email: string) {
  state.privateKey = privateKey;
  state.publicKeyB64 = publicKeyB64;
  state.email = email;
  // Persist public key + email so we can show context after reload (not secret).
  localStorage.setItem("pp_pub", publicKeyB64);
  localStorage.setItem("pp_email", email);
}

export function clearSession() {
  state.privateKey = null;
  state.publicKeyB64 = null;
  state.email = null;
  localStorage.removeItem("pp_pub");
  localStorage.removeItem("pp_email");
}

export function isUnlocked(): boolean {
  return state.privateKey !== null;
}

// getPrivateKey / getPublicKey expose the in-memory key material for local
// biometric enrollment (wrapping it behind a platform authenticator). The
// private key still never touches disk in the clear.
export function getPrivateKey(): Uint8Array | null {
  return state.privateKey;
}

export function getPublicKey(): string | null {
  return state.publicKeyB64 ?? localStorage.getItem("pp_pub");
}

export function getEmail(): string | null {
  return state.email ?? localStorage.getItem("pp_email");
}

export function encrypt(plaintext: string): string {
  if (!state.publicKeyB64) throw new Error("session not unlocked");
  return eciesEncrypt(state.publicKeyB64, plaintext);
}

export function decrypt(ciphertextB64: string): string {
  if (!state.privateKey) return "<locked>";
  return eciesDecryptSafe(state.privateKey, ciphertextB64);
}

export { b64urlEncode };
