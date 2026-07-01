import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import { decrypt, getEmail, getPrivateKey, getPublicKey } from "../crypto/session";
import { enablePush, disablePush, isPushEnabled, pushSupported } from "../push";
import {
  biometricAvailable,
  hasBiometricUnlock,
  enrollBiometric,
  clearBiometricUnlock,
} from "../crypto/biometric";
import { useAutoRefresh } from "../useAutoRefresh";
import { cachePipelineNames } from "../nameCache";
import { RunFeed, RunItem } from "../components/RunFeed";

interface Summary {
  pass: number;
  fail: number;
  running: number;
}

export function Dashboard() {
  const [recent, setRecent] = useState<RunItem[]>([]);
  const [summary, setSummary] = useState<Summary>({ pass: 0, fail: 0, running: 0 });
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const [pushMsg, setPushMsg] = useState<string | null>(null);
  const [pushOn, setPushOn] = useState(false);
  const [pushBusy, setPushBusy] = useState(false);

  const [bioAvail, setBioAvail] = useState(false);
  const [bioOn, setBioOn] = useState(hasBiometricUnlock(getEmail()));
  const [bioBusy, setBioBusy] = useState(false);
  const [bioMsg, setBioMsg] = useState<string | null>(null);

  useEffect(() => {
    isPushEnabled().then(setPushOn);
    biometricAvailable().then(setBioAvail);
  }, []);

  const load = useCallback(async () => {
    setRefreshing(true);
    const out: RunItem[] = [];
    const nameEntries: { id: string; label: string }[] = [];
    const sum: Summary = { pass: 0, fail: 0, running: 0 };
    try {
      const projects = await api.listProjects();
      for (const p of projects) {
        const projName = decrypt(p.encryptedName);
        const pipelines = await api.listPipelines(p.id);
        for (const pipe of pipelines) {
          const pipeName = decrypt(pipe.encryptedName);
          nameEntries.push({ id: pipe.id, label: `${projName} · ${pipeName}` });
          const runs = await api.listRuns(pipe.id, 5);

          // Newest run of this pipeline decides its current signal.
          let latest: { status: string; when: number } | null = null;
          for (const r of runs) {
            const when = new Date(r.receivedAt).getTime();
            if (!latest || when > latest.when) latest = { status: r.status, when };
            let payload: { branch?: string; commit?: string; message?: string } = {};
            try {
              payload = JSON.parse(decrypt(r.encryptedPayload));
            } catch {
              /* ignore */
            }
            out.push({
              key: r.id,
              status: r.status,
              project: projName,
              pipeline: pipeName,
              branch: payload.branch,
              commit: payload.commit,
              message: payload.message,
              when: r.receivedAt,
              live: r.status === "running",
            });
          }
          if (latest?.status === "success") sum.pass++;
          else if (latest?.status === "failure") sum.fail++;
          else if (latest?.status === "running") sum.running++;
        }
      }
    } finally {
      out.sort((a, b) => b.when.localeCompare(a.when));
      setRecent(out.slice(0, 20));
      setSummary(sum);
      setLoading(false);
      setRefreshing(false);
      // Cache decrypted names so the service worker can label push notifications.
      void cachePipelineNames(nameEntries);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // Auto-refresh on a timer, on incoming Web Push, and on tab re-focus.
  useAutoRefresh(load);

  async function togglePush() {
    setPushBusy(true);
    try {
      if (pushOn) {
        await disablePush();
        setPushOn(false);
        setPushMsg("Push notifications disabled on this device");
      } else {
        await enablePush();
        setPushOn(true);
        setPushMsg("Push notifications enabled on this device");
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
        setBioMsg("Face ID unlock removed from this device");
        return;
      }
      const priv = getPrivateKey();
      const pub = getPublicKey();
      const email = getEmail();
      if (!priv || !pub || !email) throw new Error("Session locked — log in again first");
      await enrollBiometric(priv, pub, email);
      setBioOn(true);
      setBioMsg("Face ID unlock enabled — next time, skip the password on this device");
    } catch (e) {
      // A user cancelling the system prompt shouldn't read as an error.
      const msg = e instanceof Error ? e.message : String(e);
      setBioMsg(/not allowed|cancel|abort/i.test(msg) ? null : msg);
    } finally {
      setBioBusy(false);
    }
  }

  return (
    <div>
      <div className="page-head">
        <div>
          <div className="eyebrow">Signals</div>
          <h1>Dashboard</h1>
        </div>
        <button
          className="btn-icon"
          onClick={load}
          disabled={refreshing}
          title="Refresh"
          aria-label="Refresh runs"
        >
          {refreshing ? "…" : "↻"}
        </button>
      </div>

      <div className="toolbar">
        {pushSupported() && (
          <button className={`chip${pushOn ? " on" : ""}`} onClick={togglePush} disabled={pushBusy}>
            🔔 {pushBusy ? "…" : pushOn ? "Push on" : "Enable push"}
          </button>
        )}
        {bioAvail && (
          <button className={`chip${bioOn ? " on" : ""}`} onClick={toggleBio} disabled={bioBusy}>
            🔐 {bioBusy ? "…" : bioOn ? "Face ID on" : "Enable Face ID"}
          </button>
        )}
      </div>
      {pushMsg && <p className="muted" style={{ fontSize: "0.85rem" }}>{pushMsg}</p>}
      {bioMsg && <p className="muted" style={{ fontSize: "0.85rem" }}>{bioMsg}</p>}

      {recent.length > 0 && (
        <div className="signal-summary">
          <span className="sig sig-pass">
            <span className="dot" />
            <b>{summary.pass}</b> passing
          </span>
          <span className="sig sig-fail">
            <span className="dot" />
            <b>{summary.fail}</b> failing
          </span>
          {summary.running > 0 && (
            <span className="sig sig-run">
              <span className="dot" />
              <b>{summary.running}</b> running
            </span>
          )}
        </div>
      )}

      <h2>Recent runs</h2>
      {loading && <p className="muted">Loading…</p>}
      {!loading && recent.length === 0 && (
        <div className="empty">
          No runs yet. Create a project, add a pipeline, generate a token, and wire it into your
          CI/CD.
        </div>
      )}
      <RunFeed items={recent} />
    </div>
  );
}
