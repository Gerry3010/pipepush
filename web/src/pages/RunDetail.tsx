import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { api } from "../api/client";
import { decrypt } from "../crypto/session";
import { statusGlyph, relTime } from "../components/RunFeed";

interface RunPayload {
  status: string;
  pipeline?: string;
  runId?: string;
  commit?: string;
  branch?: string;
  duration?: string;
  message?: string;
  logs?: string;
}

export function RunDetail() {
  const { runId } = useParams<{ runId: string }>();
  const [status, setStatus] = useState<string>("");
  const [when, setWhen] = useState<string>("");
  const [d, setD] = useState<RunPayload | null>(null);
  const [state, setState] = useState<"loading" | "ok" | "notfound" | "error">("loading");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!runId) return;
    let cancelled = false;
    (async () => {
      try {
        const run = await api.getRun(runId);
        if (cancelled) return;
        setStatus(run.status);
        setWhen(run.receivedAt);
        try {
          setD(JSON.parse(decrypt(run.encryptedPayload)));
        } catch {
          setD({ status: run.status });
        }
        setState("ok");
      } catch (e) {
        if (cancelled) return;
        setState(e instanceof Error && /404|not found/i.test(e.message) ? "notfound" : "error");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [runId]);

  async function copyLogs() {
    if (!d?.logs) return;
    try {
      await navigator.clipboard.writeText(d.logs);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard blocked */
    }
  }

  if (state === "loading") return <p className="muted">Loading…</p>;
  if (state === "notfound")
    return (
      <div className="empty">
        Run not found. <Link to="/">Back to dashboard</Link>
      </div>
    );
  if (state === "error")
    return (
      <div className="empty">
        Couldn’t load this run. <Link to="/">Back to dashboard</Link>
      </div>
    );

  const rows: [string, string | undefined][] = [
    ["Branch", d?.branch],
    ["Commit", d?.commit],
    ["Duration", d?.duration],
    ["CI run", d?.runId],
  ];

  return (
    <div className={`run-detail run-${status}`}>
      <div className="eyebrow">Run</div>
      <div className="rd-hero">
        <span className="node" aria-hidden="true">
          {statusGlyph[status] ?? "•"}
        </span>
        <div>
          <h1>{d?.pipeline || "Pipeline run"}</h1>
          <div className="muted mono" style={{ fontSize: "0.85rem" }}>
            {status}
            {when && (
              <>
                {" · "}
                <time dateTime={when} title={new Date(when).toLocaleString()}>
                  {relTime(when)}
                </time>
              </>
            )}
          </div>
        </div>
      </div>

      {d?.message && <p className="rd-message">{d.message}</p>}

      <dl className="rd-meta">
        {rows
          .filter(([, v]) => v)
          .map(([k, v]) => (
            <div key={k} className="rd-row">
              <dt>{k}</dt>
              <dd className="mono">{v}</dd>
            </div>
          ))}
      </dl>

      <h2>Logs</h2>
      {d?.logs ? (
        <div className="snippet-wrap">
          <button className="copy-snippet" onClick={copyLogs}>
            {copied ? "copied ✓" : "copy"}
          </button>
          <pre className="snippet">{d.logs}</pre>
        </div>
      ) : (
        <div className="empty">
          No logs were sent with this run. Add a <code className="mono">logs</code> field to your
          CI webhook to see build output here.
        </div>
      )}

      <p style={{ marginTop: "1.5rem" }}>
        <Link className="link-btn" to="/">
          ← Back to runs
        </Link>
      </p>
    </div>
  );
}
