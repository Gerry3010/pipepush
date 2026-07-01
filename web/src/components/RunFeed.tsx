// Shared run feed — the signature element. Each run is a card with a colored
// "pipe spine" and a status node; the newest run of a running pipeline pulses.

export const statusGlyph: Record<string, string> = {
  success: "✓",
  failure: "✗",
  cancelled: "⊘",
  running: "●",
  skipped: "○",
};

export interface RunItem {
  key: string;
  status: string;
  project?: string;
  pipeline: string;
  branch?: string;
  commit?: string;
  message?: string;
  when: string;
  live?: boolean;
}

// relTime renders a compact, glanceable age ("just now", "4m", "2h", "3d")
// and falls back to a date for anything older than a week.
export function relTime(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const s = Math.max(0, Math.round((Date.now() - then) / 1000));
  if (s < 45) return "just now";
  const m = Math.round(s / 60);
  if (m < 60) return `${m}m`;
  const h = Math.round(m / 60);
  if (h < 24) return `${h}h`;
  const d = Math.round(h / 24);
  if (d < 7) return `${d}d`;
  return new Date(iso).toLocaleDateString();
}

function Run({ r }: { r: RunItem }) {
  const sub = [r.branch, r.commit ? r.commit.slice(0, 8) : "", r.message]
    .filter(Boolean)
    .join(" · ");
  return (
    <li className={`run run-${r.status}`}>
      <span className="node" aria-hidden="true">
        {statusGlyph[r.status] ?? "•"}
      </span>
      <div className="run-body">
        <div className="run-title">
          {r.project && (
            <>
              <span className="proj">{r.project}</span>
              <span className="sep">·</span>
            </>
          )}
          {r.pipeline}
        </div>
        {sub && <div className="run-sub">{sub}</div>}
      </div>
      <div className="run-meta">
        {r.live && <span className="run-live">live</span>}
        <time className="run-when" dateTime={r.when} title={new Date(r.when).toLocaleString()}>
          {relTime(r.when)}
        </time>
      </div>
    </li>
  );
}

export function RunFeed({ items }: { items: RunItem[] }) {
  return (
    <ul className="feed">
      {items.map((r) => (
        <Run key={r.key} r={r} />
      ))}
    </ul>
  );
}
