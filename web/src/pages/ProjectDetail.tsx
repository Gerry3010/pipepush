import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api, Pipeline, NotificationToken, Run } from "../api/client";
import { decrypt, encrypt } from "../crypto/session";

interface RunPayload {
  status: string;
  pipeline?: string;
  runId?: string;
  commit?: string;
  branch?: string;
  duration?: string;
  message?: string;
}

const statusGlyph: Record<string, string> = {
  success: "✓",
  failure: "✗",
  cancelled: "⊘",
  running: "●",
  skipped: "○",
};

export function ProjectDetail() {
  const { id } = useParams<{ id: string }>();
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [tokens, setTokens] = useState<NotificationToken[]>([]);
  const [runs, setRuns] = useState<Record<string, Run[]>>({});
  const [newPipeline, setNewPipeline] = useState("");
  const [newToken, setNewToken] = useState("");
  const [tokenPipeline, setTokenPipeline] = useState("");
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    if (!id) return;
    try {
      const ps = await api.listPipelines(id);
      setPipelines(ps);
      setTokens(await api.listTokens(id));
      const runMap: Record<string, Run[]> = {};
      for (const p of ps) {
        runMap[p.id] = await api.listRuns(p.id, 10);
      }
      setRuns(runMap);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  async function createPipeline(e: React.FormEvent) {
    e.preventDefault();
    if (!id || !newPipeline.trim()) return;
    await api.createPipeline(id, encrypt(newPipeline.trim()));
    setNewPipeline("");
    load();
  }

  async function createToken(e: React.FormEvent) {
    e.preventDefault();
    if (!id || !newToken.trim()) return;
    const res = await api.createToken(encrypt(newToken.trim()), id, tokenPipeline);
    setCreatedToken(res.plaintextToken);
    setNewToken("");
    load();
  }

  function decodeRun(r: Run): RunPayload {
    try {
      return JSON.parse(decrypt(r.encryptedPayload));
    } catch {
      return { status: r.status };
    }
  }

  return (
    <div>
      <h1>Project</h1>
      {err && <p className="error">{err}</p>}

      <section>
        <h2>Pipelines</h2>
        <form onSubmit={createPipeline} className="inline-form">
          <input
            placeholder="New pipeline name"
            value={newPipeline}
            onChange={(e) => setNewPipeline(e.target.value)}
          />
          <button className="primary">Add</button>
        </form>
        {pipelines.map((p) => (
          <div key={p.id} className="card">
            <h3>{decrypt(p.encryptedName)}</h3>
            <table className="runs">
              <tbody>
                {(runs[p.id] ?? []).map((r) => {
                  const d = decodeRun(r);
                  return (
                    <tr key={r.id} className={`run run-${r.status}`}>
                      <td className="glyph">{statusGlyph[r.status] ?? "•"}</td>
                      <td>{r.status}</td>
                      <td>{d.branch}</td>
                      <td className="mono">{d.commit?.slice(0, 8)}</td>
                      <td>{d.message}</td>
                      <td className="date">
                        {new Date(r.receivedAt).toLocaleString()}
                      </td>
                    </tr>
                  );
                })}
                {(runs[p.id] ?? []).length === 0 && (
                  <tr>
                    <td colSpan={6} className="muted">
                      No runs yet
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        ))}
      </section>

      <section>
        <h2>Notification Tokens</h2>
        <form onSubmit={createToken} className="inline-form">
          <input
            placeholder="Token name (e.g. GitHub Actions)"
            value={newToken}
            onChange={(e) => setNewToken(e.target.value)}
          />
          <select value={tokenPipeline} onChange={(e) => setTokenPipeline(e.target.value)}>
            <option value="">— bind to pipeline —</option>
            {pipelines.map((p) => (
              <option key={p.id} value={p.id}>
                {decrypt(p.encryptedName)}
              </option>
            ))}
          </select>
          <button className="primary">Create token</button>
        </form>

        {createdToken && (
          <div className="card token-reveal">
            <p>Copy this token now — shown only once:</p>
            <code>{createdToken}</code>
            <button className="link-btn" onClick={() => setCreatedToken(null)}>
              dismiss
            </button>
          </div>
        )}

        <ul className="token-list">
          {tokens.map((t) => (
            <li key={t.id} className={t.active ? "" : "revoked"}>
              <span>
                {t.active ? "●" : "○"} {decrypt(t.encryptedName)}
              </span>
              {t.active && (
                <button
                  className="link-btn"
                  onClick={async () => {
                    await api.revokeToken(t.id);
                    load();
                  }}
                >
                  revoke
                </button>
              )}
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
