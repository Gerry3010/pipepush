// Web Push subscription via the service worker + VAPID.

import { api } from "./api/client";
import { b64urlEncode } from "./crypto/ecies";

function urlBase64ToArrayBuffer(base64String: string): ArrayBuffer {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(base64);
  const out = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out.buffer;
}

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (!("serviceWorker" in navigator)) return null;
  return navigator.serviceWorker.register("/sw.js", { type: "module" });
}

export async function enablePush(): Promise<void> {
  if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
    throw new Error("Push notifications are not supported in this browser");
  }
  const permission = await Notification.requestPermission();
  if (permission !== "granted") {
    throw new Error("Notification permission denied");
  }

  const reg = await navigator.serviceWorker.ready;
  const { publicKey } = await api.vapidKey();
  if (!publicKey) throw new Error("server has no VAPID key configured");

  const sub = await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToArrayBuffer(publicKey),
  });

  const json = sub.toJSON();
  await api.subscribePush({
    endpoint: sub.endpoint,
    p256dhKey: json.keys?.p256dh ?? "",
    authKey: json.keys?.auth ?? "",
    deviceName: navigator.userAgent.slice(0, 80),
  });
}

// The service worker can't decrypt (no private key). It posts encrypted run
// events to the page via postMessage; the page decrypts and can re-notify.
export { b64urlEncode };
