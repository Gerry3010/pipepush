import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { api, setJWT, getJWT } from "../api/client";
import { generateKeypair, wrapPrivateKey, unwrapPrivateKey } from "../crypto/ecies";
import { setSession, getEmail } from "../crypto/session";
import {
  biometricSupported,
  hasBiometricUnlock,
  unlockWithBiometric,
  biometricEmail,
  biometricErrorMessage,
} from "../crypto/biometric";

export function Login({ onAuth }: { onAuth: () => void }) {
  const [email, setEmail] = useState(getEmail() ?? "");
  const [password, setPassword] = useState("");
  const [register, setRegister] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const navigate = useNavigate();
  const loc = useLocation();
  // Where a deep link (e.g. a run from a push notification) wanted to go.
  const dest = (loc.state as { from?: string } | null)?.from ?? "/";

  // Offer biometric unlock only when this device is enrolled AND we still hold a
  // valid session token — Face ID restores the key locally but can't re-auth.
  const canBio = biometricSupported() && hasBiometricUnlock() && !!getJWT();
  const [bioBusy, setBioBusy] = useState(false);

  async function unlockBio() {
    setErr(null);
    setBioBusy(true);
    try {
      const { privateKey, pub, email: e } = await unlockWithBiometric();
      setSession(privateKey, pub, e);
      onAuth();
      navigate(dest);
    } catch (e) {
      setErr(biometricErrorMessage(e));
    } finally {
      setBioBusy(false);
    }
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      let resp;
      if (register) {
        const kp = generateKeypair();
        const wrapped = await wrapPrivateKey(kp.privateKey, password);
        resp = await api.register({
          email,
          password,
          publicKey: kp.publicKeyB64,
          encryptedPrivateKey: wrapped.encryptedPrivateKey,
          kdfSalt: wrapped.kdfSalt,
        });
      } else {
        resp = await api.login(email, password);
      }

      // Unlock the private key locally.
      const priv = await unwrapPrivateKey(resp.encryptedPrivateKey, resp.kdfSalt, password);
      setJWT(resp.jwt);
      setSession(priv, resp.publicKey, email);
      onAuth();
      navigate(dest);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="card narrow">
      <div className="eyebrow">{register ? "Get started" : canBio ? "Welcome back" : "Sign in"}</div>
      <h1>{register ? "Create account" : "Log in"}</h1>

      {canBio && !register && (
        <>
          <button className="btn btn-primary btn-block" onClick={unlockBio} disabled={bioBusy}>
            🔐 {bioBusy ? "Unlocking…" : `Unlock with Face ID`}
          </button>
          {biometricEmail() && <p className="muted" style={{ fontSize: "0.8rem", marginTop: "0.5rem" }}>{biometricEmail()}</p>}
          <div className="divider">or use password</div>
        </>
      )}

      <form onSubmit={submit}>
        <label>Email</label>
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoFocus={!canBio}
          autoComplete="email"
        />
        <label>Password</label>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={8}
          autoComplete={register ? "new-password" : "current-password"}
        />
        {err && <p className="error">{err}</p>}
        <button type="submit" disabled={busy} className="btn btn-primary btn-block" style={{ marginTop: "1rem" }}>
          {busy ? "…" : register ? "Register" : "Log in"}
        </button>
      </form>

      <p className="muted" style={{ marginTop: "1rem" }}>
        {register ? "Already have an account? " : "New here? "}
        <button className="link-btn" onClick={() => setRegister(!register)}>
          {register ? "Log in" : "Create one"}
        </button>
      </p>
      <p className="hint">
        🔒 Your password never leaves this device unencrypted. It unlocks an end-to-end encryption
        key — the server can't read your pipeline data.
      </p>
    </div>
  );
}
