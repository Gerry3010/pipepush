// Typed API client for the pipepush server.

export interface LoginResponse {
  jwt: string;
  publicKey: string;
  encryptedPrivateKey: string;
  kdfSalt: string;
}

export interface Project {
  id: string;
  encryptedName: string;
  encryptedDescription?: string;
  createdAt: string;
}

export interface Pipeline {
  id: string;
  projectId: string;
  encryptedName: string;
  createdAt: string;
}

export interface NotificationToken {
  id: string;
  projectId: string;
  pipelineId?: string;
  encryptedName: string;
  active: boolean;
  createdAt: string;
  lastUsedAt?: string | null;
}

export interface Run {
  id: string;
  pipelineId: string;
  status: string;
  encryptedPayload: string;
  receivedAt: string;
}

export interface Settings {
  // null = keep runs forever; otherwise prune runs older than this many hours.
  retentionHours: number | null;
}

let jwt: string | null = localStorage.getItem("pp_jwt");

export function setJWT(token: string | null) {
  jwt = token;
  if (token) localStorage.setItem("pp_jwt", token);
  else localStorage.removeItem("pp_jwt");
}

export function getJWT(): string | null {
  return jwt;
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (jwt) headers["Authorization"] = `Bearer ${jwt}`;

  const res = await fetch(`/api${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let msg = `request failed (${res.status})`;
    try {
      const j = await res.json();
      if (j.error) msg = j.error;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }

  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return text ? (JSON.parse(text) as T) : (undefined as T);
}

export const api = {
  register: (b: {
    email: string;
    password: string;
    publicKey: string;
    encryptedPrivateKey: string;
    kdfSalt: string;
  }) => req<LoginResponse>("POST", "/auth/register", b),

  login: (email: string, password: string) =>
    req<LoginResponse>("POST", "/auth/login", { email, password }),

  listProjects: () => req<Project[]>("GET", "/projects"),
  createProject: (encryptedName: string, encryptedDescription = "") =>
    req<Project>("POST", "/projects", { encryptedName, encryptedDescription }),
  deleteProject: (id: string) => req<void>("DELETE", `/projects/${id}`),

  listPipelines: (projectId: string) =>
    req<Pipeline[]>("GET", `/projects/${projectId}/pipelines`),
  createPipeline: (projectId: string, encryptedName: string, routingKey = "") =>
    req<Pipeline>("POST", `/projects/${projectId}/pipelines`, { encryptedName, routingKey }),

  listTokens: (projectId: string) =>
    req<NotificationToken[]>("GET", `/projects/${projectId}/tokens`),
  createToken: (encryptedName: string, projectId: string, pipelineId = "") =>
    req<{ token: NotificationToken; plaintextToken: string }>("POST", "/tokens", {
      encryptedName,
      projectId,
      pipelineId,
    }),
  revokeToken: (id: string) => req<void>("DELETE", `/tokens/${id}`),
  deleteToken: (id: string) => req<void>("DELETE", `/tokens/${id}/permanent`),

  listRuns: (pipelineId: string, limit = 50) =>
    req<Run[]>("GET", `/pipelines/${pipelineId}/runs?limit=${limit}`),
  getRun: (id: string) => req<Run>("GET", `/runs/${id}`),

  getSettings: () => req<Settings>("GET", "/settings"),
  updateSettings: (s: Settings) => req<Settings>("POST", "/settings", s),

  vapidKey: () => req<{ publicKey: string }>("GET", "/push/vapid-key"),
  subscribePush: (b: {
    endpoint: string;
    p256dhKey: string;
    authKey: string;
    deviceName: string;
  }) => req<void>("POST", "/push/subscribe", b),
  unsubscribePush: (b: { endpoint: string }) =>
    req<void>("DELETE", "/push/subscribe", b),
};
