import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import { decrypt } from "../crypto/session";
import { useAutoRefresh } from "../useAutoRefresh";
import { cachePipelineNames } from "../nameCache";
import { RunFeed, RunItem } from "../components/RunFeed";
import { IconRefresh, IconCheck, IconWarning, IconCross } from "../components/icons";

interface Summary {
  pass: number;
  fail: number;
  running: number;
}

type Filter = "all" | "success" | "warning" | "error";

// "Warning" groups the amber/grey non-critical outcomes (a real `warning`
// status plus skipped and cancelled runs).
const WARN_STATUSES = new Set(["warning", "skipped", "cancelled"]);

function matchesFilter(status: string, f: Filter): boolean {
  switch (f) {
    case "success":
      return status === "success";
    case "error":
      return status === "failure";
    case "warning":
      return WARN_STATUSES.has(status);
    default:
      return true;
  }
}

export function Dashboard() {
  const [recent, setRecent] = useState<RunItem[]>([]);
  const [summary, setSummary] = useState<Summary>({ pass: 0, fail: 0, running: 0 });
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [filter, setFilter] = useState<Filter>("all");

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

  const counts = useMemo(
    () => ({
      all: recent.length,
      success: recent.filter((r) => r.status === "success").length,
      warning: recent.filter((r) => WARN_STATUSES.has(r.status)).length,
      error: recent.filter((r) => r.status === "failure").length,
    }),
    [recent],
  );
  const filtered = recent.filter((r) => matchesFilter(r.status, filter));

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
          <IconRefresh className={refreshing ? "spin" : undefined} />
        </button>
      </div>

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
      {recent.length > 0 && (
        <div className="filter-bar" role="tablist" aria-label="Filter runs by status">
          <button
            role="tab"
            aria-selected={filter === "all"}
            className={`fchip${filter === "all" ? " on" : ""}`}
            onClick={() => setFilter("all")}
          >
            All <b>{counts.all}</b>
          </button>
          <button
            role="tab"
            aria-selected={filter === "success"}
            className={`fchip f-pass${filter === "success" ? " on" : ""}`}
            onClick={() => setFilter("success")}
          >
            <IconCheck /> Success <b>{counts.success}</b>
          </button>
          <button
            role="tab"
            aria-selected={filter === "warning"}
            className={`fchip f-warn${filter === "warning" ? " on" : ""}`}
            onClick={() => setFilter("warning")}
          >
            <IconWarning /> Warning <b>{counts.warning}</b>
          </button>
          <button
            role="tab"
            aria-selected={filter === "error"}
            className={`fchip f-fail${filter === "error" ? " on" : ""}`}
            onClick={() => setFilter("error")}
          >
            <IconCross /> Error <b>{counts.error}</b>
          </button>
        </div>
      )}
      {loading && <p className="muted">Loading…</p>}
      {!loading && recent.length === 0 && (
        <div className="empty">
          No runs yet. Create a project, add a pipeline, generate a token, and wire it into your
          CI/CD.
        </div>
      )}
      {!loading && recent.length > 0 && filtered.length === 0 && (
        <div className="empty">No {filter} runs right now.</div>
      )}
      <RunFeed items={filtered} />
    </div>
  );
}
