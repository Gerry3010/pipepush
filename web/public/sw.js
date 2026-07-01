// pipepush service worker — receives Web Push events.
//
// The push payload is end-to-end encrypted: the SW cannot read run details
// (it has no private key). It shows a notification and forwards the encrypted
// blob to any open page, which decrypts and can show specifics.
//
// The pipeline/project NAME is also encrypted, so the server only sends the
// non-sensitive pipelineId. We look the name up in an IndexedDB cache the app
// populated with client-decrypted names (see src/nameCache.ts) — the plaintext
// name never travels through the push service.

const NAME_DB = "pipepush-names";
const NAME_STORE = "pipelines";

// idbGetLabel resolves the cached "Project · Pipeline" label for a pipelineId,
// or null. Best-effort — any error resolves null (falls back to a generic body).
function idbGetLabel(pipelineId) {
  return new Promise((resolve) => {
    if (!pipelineId || !("indexedDB" in self)) return resolve(null);
    let req;
    try {
      req = indexedDB.open(NAME_DB, 1);
    } catch {
      return resolve(null);
    }
    req.onupgradeneeded = () => {
      try {
        req.result.createObjectStore(NAME_STORE);
      } catch {
        /* ignore */
      }
    };
    req.onsuccess = () => {
      const db = req.result;
      try {
        const g = db.transaction(NAME_STORE, "readonly").objectStore(NAME_STORE).get(pipelineId);
        g.onsuccess = () => {
          resolve(g.result || null);
          db.close();
        };
        g.onerror = () => {
          resolve(null);
          db.close();
        };
      } catch {
        resolve(null);
      }
    };
    req.onerror = () => resolve(null);
  });
}

self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = {};
  }

  const status = data.status || "update";
  const title = "pipepush";

  event.waitUntil(
    (async () => {
      // Forward the encrypted event to open clients for in-page decryption.
      const clientsList = await self.clients.matchAll({
        type: "window",
        includeUncontrolled: true,
      });
      for (const client of clientsList) {
        client.postMessage({ type: "run_update", data });
      }

      const name = (await idbGetLabel(data.pipelineId)) || "A pipeline";
      const body =
        status === "success"
          ? `✓ ${name} succeeded`
          : status === "failure"
            ? `✗ ${name} failed`
            : `${name}: ${status}`;

      await self.registration.showNotification(title, {
        body,
        icon: "/icon-192.png",
        badge: "/icon-192.png",
        tag: data.runId || "pipepush",
        data,
      });
    })()
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(self.clients.openWindow("/"));
});
