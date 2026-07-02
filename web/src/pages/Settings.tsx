import { useEffect, useState } from "react";
import { api } from "../api/client";
import { getEmail, getPrivateKey, getPublicKey } from "../crypto/session";
import { enablePush, disablePush, isPushEnabled, pushSupported } from "../push";
import {
  biometricSupported,
  biometricAvailable,
  hasBiometricUnlock,
  enrollBiometric,
  clearBiometricUnlock,
  biometricErrorMessage,
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

  // Retention: null = keep forever. "" in the <select> maps to null.
  const [retention, setRetention] = useState<number | null>(null);
  const [retentionMsg, setRetentionMsg] = useState<string | null>(null);

  useEffect(() => {
    isPushEnabled().then(setPushOn);
    biometricAvailable().then(setBioAvail);
    api
      .getSettings()
      .then((s) => setRetention(s.retentionHours))
      .catch(() => {
        /* leave default */
      });
  }, []);

  async function changeRetention(value: string) {
    const hours = value === "" ? null : parseInt(value, 10);
    const prev = retention;
    setRetention(hours);
    setRetentionMsg(null);
    try {
      await api.updateSettings({ retentionHours: hours });
      setRetentionMsg("Saved.");
    } catch (e) {
      setRetention(prev);
      setRetentionMsg(e instanceof Error ? e.message : String(e));
    }
  }

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
      setBioMsg(biometricErrorMessage(e));
    } finally {
      setBioBusy(false);
    }
  }

  // What to render for the biometric control. We DON'T hard-gate on isUVPAA:
  // inside an installed iOS PWA it often reports false even when Face ID exists.
  // So we offer Enable whenever the WebAuthn APIs are present and let the enroll
  // attempt surface the real reason if it can't proceed.
  function bioControl() {
    if (!biometricSupported()) {
      return <span className="muted" style={{ fontSize: "0.82rem" }}>Not supported here</span>;
    }
    return (
      <button className={`chip${bioOn ? " on" : ""}`} onClick={toggleBio} disabled={bioBusy}>
        {bioBusy ? "…" : bioOn ? "On" : "Enable"}
      </button>
    );
  }

  const bioDesc =
    biometricSupported() && bioAvail === false
      ? "Unlock with Face ID / Touch ID instead of your password on this device. Your device didn’t report a biometric sensor — tap Enable to try anyway; if it can’t, the reason shows below."
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

      <div className="eyebrow section-label">Data</div>
      <div className="card">
        <div className="setting">
          <div className="setting-info">
            <div className="setting-title">Data retention</div>
            <div className="setting-desc muted">
              Runs and their logs older than this are deleted automatically. Applies to all your
              devices.
            </div>
          </div>
          <div className="setting-control">
            <select
              value={retention === null ? "" : String(retention)}
              onChange={(e) => changeRetention(e.target.value)}
              style={{ width: "auto" }}
            >
              <option value="">Keep forever</option>
              <option value="6">6 hours</option>
              <option value="24">24 hours</option>
              <option value="72">3 days</option>
              <option value="168">1 week</option>
              <option value="336">2 weeks</option>
            </select>
          </div>
        </div>
      </div>
      {retentionMsg && <p className="muted" style={{ fontSize: "0.85rem" }}>{retentionMsg}</p>}

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
