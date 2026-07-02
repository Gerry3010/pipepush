import { useEffect, useState } from "react";
import { getEmail, getPrivateKey, getPublicKey } from "../crypto/session";
import { enablePush, disablePush, isPushEnabled, pushSupported } from "../push";
import {
  biometricSupported,
  biometricAvailable,
  hasBiometricUnlock,
  enrollBiometric,
  clearBiometricUnlock,
} from "../crypto/biometric";

export function Settings({ onLogout }: { onLogout: () => void }) {
  const [pushOn, setPushOn] = useState(false);
  const [pushBusy, setPushBusy] = useState(false);
  const [pushMsg, setPushMsg] = useState<string | null>(null);

  // undefined = still checking; false = no platform authenticator here.
  const [bioAvail, setBioAvail] = useState<boolean | undefined>(undefined);
  const [bioOn, setBioOn] = useState(hasBiometricUnlock(getEmail()));
  const [bioBusy, setBioBusy] = useState(false);
  const [bioMsg, setBioMsg] = useState<string | null>(null);

  useEffect(() => {
    isPushEnabled().then(setPushOn);
    biometricAvailable().then(setBioAvail);
  }, []);

  async function togglePush() {
    setPushBusy(true);
    setPushMsg(null);
    try {
      if (pushOn) {
        await disablePush();
        setPushOn(false);
      } else {
        await enablePush();
        setPushOn(true);
      }
    } catch (e) {
      setPushMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setPushBusy(false);
    }
  }

  async function toggleBio() {
    setBioBusy(true);
    setBioMsg(null);
    try {
      if (bioOn) {
        clearBiometricUnlock();
        setBioOn(false);
        setBioMsg("Removed from this device.");
        return;
      }
      const priv = getPrivateKey();
      const pub = getPublicKey();
      const email = getEmail();
      if (!priv || !pub || !email) throw new Error("Session locked — log in again first.");
      await enrollBiometric(priv, pub, email);
      setBioOn(true);
      setBioMsg("Enabled — next time you can skip the password on this device.");
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      // Cancelling the system prompt isn't an error worth surfacing.
      setBioMsg(/not allowed|cancel|abort/i.test(msg) ? null : msg);
    } finally {
      setBioBusy(false);
    }
  }

  // What to render for the biometric control, given availability.
  function bioControl() {
    if (!biometricSupported()) {
      return <span className="muted" style={{ fontSize: "0.82rem" }}>Not supported here</span>;
    }
    if (bioAvail === undefined) {
      return <span className="muted" style={{ fontSize: "0.82rem" }}>…</span>;
    }
    if (!bioAvail) {
      return <span className="muted" style={{ fontSize: "0.82rem" }}>Unavailable</span>;
    }
    return (
      <button className={`chip${bioOn ? " on" : ""}`} onClick={toggleBio} disabled={bioBusy}>
        {bioBusy ? "…" : bioOn ? "On" : "Enable"}
      </button>
    );
  }

  const bioDesc =
    biometricSupported() && bioAvail === false
      ? "This device has no Face ID / Touch ID / Windows Hello. Open pipepush on your phone to enable it there."
      : "Unlock with Face ID instead of your password on this device. Your key is wrapped behind the biometric locally — the plaintext never leaves the device.";

  return (
    <div>
      <div className="eyebrow">This device &amp; account</div>
      <h1>Settings</h1>

      <div className="eyebrow section-label">This device</div>
      <div className="card">
        <div className="setting">
          <div className="setting-info">
            <div className="setting-title">Push notifications</div>
            <div className="setting-desc muted">
              Get a notification here the moment a pipeline finishes.
            </div>
          </div>
          <div className="setting-control">
            {pushSupported() ? (
              <button className={`chip${pushOn ? " on" : ""}`} onClick={togglePush} disabled={pushBusy}>
                {pushBusy ? "…" : pushOn ? "On" : "Enable"}
              </button>
            ) : (
              <span className="muted" style={{ fontSize: "0.82rem" }}>Unsupported</span>
            )}
          </div>
        </div>

        <div className="setting">
          <div className="setting-info">
            <div className="setting-title">Face ID unlock</div>
            <div className="setting-desc muted">{bioDesc}</div>
          </div>
          <div className="setting-control">{bioControl()}</div>
        </div>
      </div>
      {pushMsg && <p className="error">{pushMsg}</p>}
      {bioMsg && <p className="muted" style={{ fontSize: "0.85rem" }}>{bioMsg}</p>}

      <div className="eyebrow section-label">Account</div>
      <div className="card">
        <div className="setting">
          <div className="setting-info">
            <div className="setting-title">Signed in</div>
            <div className="setting-desc muted mono">{getEmail()}</div>
          </div>
          <div className="setting-control">
            <button className="btn btn-ghost" onClick={onLogout}>
              Log out
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
