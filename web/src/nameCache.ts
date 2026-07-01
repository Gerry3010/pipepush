// A tiny IndexedDB cache mapping pipelineId -> a human label ("Project · Pipeline").
//
// The names are decrypted client-side (E2E) and stored locally so the service
// worker — which cannot decrypt the push payload — can still show the pipeline
// name in a notification by looking it up by the non-sensitive pipelineId that
// the server includes in the push. The plaintext name never leaves the device.
//
// Must stay in sync with the reader inlined in public/sw.js (same DB/store names).

const DB_NAME = "pipepush-names";
const STORE = "pipelines";

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

// cachePipelineNames upserts {pipelineId -> label} entries. Best-effort: any
// failure is swallowed so it never breaks rendering.
export async function cachePipelineNames(
  entries: { id: string; label: string }[],
): Promise<void> {
  if (!("indexedDB" in window) || entries.length === 0) return;
  try {
    const db = await openDB();
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(STORE, "readwrite");
      const store = tx.objectStore(STORE);
      for (const e of entries) store.put(e.label, e.id);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
    db.close();
  } catch {
    /* best-effort */
  }
}
