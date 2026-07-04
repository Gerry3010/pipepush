import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { api, Project } from "../api/client";
import { decrypt, encrypt } from "../crypto/session";
import { IconPlus, IconSearch } from "../components/icons";

export function Projects() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [query, setQuery] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      setProjects(await api.listProjects());
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    load();
  }, []);

  // Names are decrypted client-side, so search filters in memory.
  const decorated = useMemo(
    () => projects.map((p) => ({ p, name: decrypt(p.encryptedName) })),
    [projects],
  );
  const q = query.trim().toLowerCase();
  const visible = q ? decorated.filter((d) => d.name.toLowerCase().includes(q)) : decorated;

  function openCreate() {
    setErr(null);
    setName("");
    setShowCreate(true);
  }

  async function create(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    setBusy(true);
    setErr(null);
    try {
      await api.createProject(encrypt(trimmed));
      setName("");
      setShowCreate(false);
      load();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <div className="eyebrow">Workspace</div>
      <h1>Projects</h1>

      {projects.length > 0 && (
        <div className="search-row">
          <IconSearch className="search-ico" />
          <input
            className="search-input"
            type="search"
            placeholder="Search projects"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            aria-label="Search projects"
          />
        </div>
      )}

      {err && !showCreate && <p className="error">{err}</p>}

      <div className="grid">
        {visible.map((d) => (
          <Link to={`/projects/${d.p.id}`} key={d.p.id} className="card link-card">
            <h3>{d.name}</h3>
            {d.p.encryptedDescription && (
              <p className="muted">{decrypt(d.p.encryptedDescription)}</p>
            )}
            <span className="date">{new Date(d.p.createdAt).toLocaleDateString()}</span>
          </Link>
        ))}
      </div>

      {projects.length === 0 && (
        <div className="empty">No projects yet. Tap + to create your first one.</div>
      )}
      {projects.length > 0 && visible.length === 0 && (
        <div className="empty">No projects match “{query.trim()}”.</div>
      )}

      <button className="fab" onClick={openCreate} aria-label="New project" title="New project">
        <IconPlus />
      </button>

      {showCreate && (
        <div className="modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>New project</h3>
              <button className="link-btn" onClick={() => setShowCreate(false)}>
                close
              </button>
            </div>
            <form onSubmit={create}>
              <label>Project name</label>
              <input
                autoFocus
                placeholder="e.g. Mobile app"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              {err && <p className="error">{err}</p>}
              <button
                className="btn btn-primary btn-block"
                disabled={busy}
                style={{ marginTop: "1rem" }}
              >
                {busy ? "Creating…" : "Create project"}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
