import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, Project } from "../api/client";
import { decrypt, encrypt } from "../crypto/session";

export function Projects() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [name, setName] = useState("");
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

  async function create(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    try {
      await api.createProject(encrypt(name.trim()));
      setName("");
      load();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div>
      <div className="eyebrow">Workspace</div>
      <h1>Projects</h1>
      <form onSubmit={create} className="inline-form">
        <div className="row">
          <input
            placeholder="New project name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <button className="btn btn-primary">Create</button>
      </form>
      {err && <p className="error">{err}</p>}
      <div className="grid">
        {projects.map((p) => (
          <Link to={`/projects/${p.id}`} key={p.id} className="card link-card">
            <h3>{decrypt(p.encryptedName)}</h3>
            {p.encryptedDescription && <p className="muted">{decrypt(p.encryptedDescription)}</p>}
            <span className="date">{new Date(p.createdAt).toLocaleDateString()}</span>
          </Link>
        ))}
      </div>
      {projects.length === 0 && <div className="empty">No projects yet.</div>}
    </div>
  );
}
