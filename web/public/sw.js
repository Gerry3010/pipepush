// pipepush service worker — receives Web Push events.
//
// The push payload is end-to-end encrypted: the SW cannot read run details
// (it has no private key). It shows a generic notification and forwards the
// encrypted blob to any open page, which decrypts and can show specifics.

self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = {};
  }

  const status = data.status || "update";
  const title = "pipepush";
  const body =
    status === "success"
      ? "✓ A pipeline succeeded"
      : status === "failure"
        ? "✗ A pipeline failed"
        : `Pipeline ${status}`;

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
