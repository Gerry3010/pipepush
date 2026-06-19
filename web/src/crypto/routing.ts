// Mirrors the Go internal/routing.Key helper: a deterministic key that lets a
// project-wide token resolve a pipeline by its (plaintext) name. The hashing
// MUST stay identical to the server side (SHA-256 of the trimmed, lowercased
// name) so a pipeline created here matches one auto-created from a webhook.
export async function routingKey(name: string): Promise<string> {
  const norm = name.trim().toLowerCase();
  if (!norm) return "";
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(norm));
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
