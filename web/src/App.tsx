import { useState } from "react";
import { Routes, Route, Navigate, Link, NavLink, useNavigate } from "react-router-dom";
import { getJWT, setJWT } from "./api/client";
import { isUnlocked, clearSession, getEmail } from "./crypto/session";
import { clearBiometricUnlock } from "./crypto/biometric";
import { Login } from "./pages/Login";
import { Dashboard } from "./pages/Dashboard";
import { Projects } from "./pages/Projects";
import { ProjectDetail } from "./pages/ProjectDetail";

function RequireAuth({ children }: { children: React.ReactNode }) {
  // Must have a JWT *and* an unlocked private key (else re-login to unlock).
  if (!getJWT() || !isUnlocked()) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
}

export function App() {
  const [, force] = useState(0);
  const navigate = useNavigate();
  const authed = !!getJWT() && isUnlocked();

  function logout() {
    setJWT(null);
    clearSession();
    // Also drop this device's biometric enrollment: without the JWT it can't
    // reach the API anyway, and a fresh login should re-enroll deliberately.
    clearBiometricUnlock();
    force((n) => n + 1);
    navigate("/login");
  }

  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">
          <span className="bolt" aria-hidden="true">
            ⚡
          </span>
          pipepush
        </Link>
        {authed && (
          <div className="topbar-actions">
            <nav className="topbar-nav">
              <NavLink to="/" end>
                Dashboard
              </NavLink>
              <NavLink to="/projects">Projects</NavLink>
            </nav>
            <span className="acct-email">{getEmail()}</span>
            <button onClick={logout} className="btn-icon" title="Lock &amp; log out" aria-label="Lock and log out">
              ⏻
            </button>
          </div>
        )}
      </header>

      <main className="content">
        <Routes>
          <Route path="/login" element={<Login onAuth={() => force((n) => n + 1)} />} />
          <Route
            path="/"
            element={
              <RequireAuth>
                <Dashboard />
              </RequireAuth>
            }
          />
          <Route
            path="/projects"
            element={
              <RequireAuth>
                <Projects />
              </RequireAuth>
            }
          />
          <Route
            path="/projects/:id"
            element={
              <RequireAuth>
                <ProjectDetail />
              </RequireAuth>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>

      {authed && (
        <nav className="tabbar" aria-label="Primary">
          <NavLink to="/" end className="tab-item">
            <span className="ic" aria-hidden="true">
              ◉
            </span>
            Runs
          </NavLink>
          <NavLink to="/projects" className="tab-item">
            <span className="ic" aria-hidden="true">
              ▦
            </span>
            Projects
          </NavLink>
          <button className="tab-item" onClick={logout}>
            <span className="ic" aria-hidden="true">
              ⏻
            </span>
            Lock
          </button>
        </nav>
      )}
    </div>
  );
}
