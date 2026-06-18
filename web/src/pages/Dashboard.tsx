import { useEffect, useState } from "react";
import { api } from "../api/client";
import { decrypt } from "../crypto/session";
import { enablePush } from "../push";

interface RecentRun {
  project: string;
  pipeline: string;
  status: string;
  branch?: string;
  when: string;
}

const statusGlyph: Record<string, string> = {
  success: "✓",
  failure: "✗",
  cancelled: "⊘",
  running: "●",
  skipped: "○",
};

export function Dashboard() {
  const [recent, setRecent] = useState<RecentRun[]>([]);
  const [pushMsg, setPushMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      const out: RecentRun[] = [];
      try {
        const projects = await api.listProjects();
        for (const p of projects) {
          const projName = decrypt(p.encryptedName);
          const pipelines = await api.listPipelines(p.id);
          for (const pipe of pipelines) {
            const pipeName = decrypt(pipe.encryptedName);
            const runs = await api.listRuns(pipe.id, 5);
            for (const r of runs) {
              let branch: string | undefined;
              try {
                branch = JSON.parse(decrypt(r.encryptedPayload)).branch;
              } catch {
                /* ignore */
              }
              out.push({
                project: projName,
                pipeline: pipeName,
                status: r.status,
                branch,
                when: r.receivedAt,
              });
            }
          }
        }
      } finally {
        out.sort((a, b) => b.when.localeCompare(a.when));
        setRecent(out.slice(0, 20));
        setLoading(false);
      }
    })();
  }, []);

  async function turnOnPush() {
    try {
      await enablePush();
      setPushMsg("✓ Push notifications enabled on this device");
    } catch (e) {
      setPushMsg(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div>
      <div className="dash-head">
        <h1>Dashboard</h1>
        <button className="primary" onClick={turnOnPush}>
          🔔 Enable push notifications
        </button>
      </div>
      {pushMsg && <p className="muted">{pushMsg}</p>}

      <h2>Recent runs</h2>
      {loading && <p className="muted">Loading…</p>}
      {!loading && recent.length === 0 && (
        <p className="muted">
          No runs yet. Create a project, add a pipeline, generate a token, and
          wire it into your CI/CD.
        </p>
      )}
      <table className="runs">
        <tbody>
          {recent.map((r, i) => (
            <tr key={i} className={`run run-${r.status}`}>
              <td className="glyph">{statusGlyph[r.status] ?? "•"}</td>
              <td>{r.status}</td>
              <td>{r.project}</td>
              <td>{r.pipeline}</td>
              <td>{r.branch}</td>
              <td className="date">{new Date(r.when).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
