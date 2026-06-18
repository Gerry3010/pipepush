import { useState } from "react";
import { Routes, Route, Navigate, Link, useNavigate } from "react-router-dom";
import { getJWT, setJWT } from "./api/client";
import { isUnlocked, clearSession, getEmail } from "./crypto/session";
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
    force((n) => n + 1);
    navigate("/login");
  }

  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">
          ⚡ pipepush
        </Link>
        {authed && (
          <nav>
            <Link to="/">Dashboard</Link>
            <Link to="/projects">Projects</Link>
            <span className="email">{getEmail()}</span>
            <button onClick={logout} className="link-btn">
              Logout
            </button>
          </nav>
        )}
      </header>

      <main className="content">
        <Routes>
          <Route
            path="/login"
            element={<Login onAuth={() => force((n) => n + 1)} />}
          />
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
    </div>
  );
}
