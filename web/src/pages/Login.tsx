import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, setJWT } from "../api/client";
import { generateKeypair, wrapPrivateKey, unwrapPrivateKey } from "../crypto/ecies";
import { setSession } from "../crypto/session";

export function Login({ onAuth }: { onAuth: () => void }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [register, setRegister] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const navigate = useNavigate();

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
      const priv = await unwrapPrivateKey(
        resp.encryptedPrivateKey,
        resp.kdfSalt,
        password
      );
      setJWT(resp.jwt);
      setSession(priv, resp.publicKey, email);
      onAuth();
      navigate("/");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="card narrow">
      <h1>{register ? "Create account" : "Log in"}</h1>
      <form onSubmit={submit}>
        <label>Email</label>
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoFocus
        />
        <label>Password</label>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={8}
        />
        {err && <p className="error">{err}</p>}
        <button type="submit" disabled={busy} className="primary">
          {busy ? "…" : register ? "Register" : "Log in"}
        </button>
      </form>
      <p className="muted">
        {register ? "Already have an account? " : "New here? "}
        <button className="link-btn" onClick={() => setRegister(!register)}>
          {register ? "Log in" : "Create one"}
        </button>
      </p>
      <p className="hint">
        🔒 Your password never leaves this device unencrypted. It unlocks an
        end-to-end encryption key — the server can't read your pipeline data.
      </p>
    </div>
  );
}
