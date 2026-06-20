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

function buffersEqual(a: ArrayBuffer | null, b: ArrayBuffer): boolean {
  if (!a || a.byteLength !== b.byteLength) return false;
  const x = new Uint8Array(a);
  const y = new Uint8Array(b);
  for (let i = 0; i < x.length; i++) if (x[i] !== y[i]) return false;
  return true;
}

export function pushSupported(): boolean {
  return "serviceWorker" in navigator && "PushManager" in window;
}

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (!("serviceWorker" in navigator)) return null;
  return navigator.serviceWorker.register("/sw.js", { type: "module" });
}

// isPushEnabled reports whether this device already has an active push
// subscription registered with the service worker.
export async function isPushEnabled(): Promise<boolean> {
  if (!pushSupported()) return false;
  const reg = await navigator.serviceWorker.ready;
  return (await reg.pushManager.getSubscription()) !== null;
}

export async function enablePush(): Promise<void> {
  if (!pushSupported()) {
    throw new Error("Push notifications are not supported in this browser");
  }
  const permission = await Notification.requestPermission();
  if (permission !== "granted") {
    throw new Error("Notification permission denied");
  }

  const reg = await navigator.serviceWorker.ready;
  const { publicKey } = await api.vapidKey();
  if (!publicKey) throw new Error("server has no VAPID key configured");
  const appServerKey = urlBase64ToArrayBuffer(publicKey);

  // If a subscription already exists but was created with a different VAPID
  // key (e.g. the server's keys were regenerated), Chrome rejects a fresh
  // subscribe() with "Registration failed - push service error". Drop the
  // stale one first so we can re-subscribe with the current key.
  const existing = await reg.pushManager.getSubscription();
  if (existing) {
    if (buffersEqual(existing.options.applicationServerKey ?? null, appServerKey)) {
      await syncSubscription(existing); // already correct — just resync server
      return;
    }
    await existing.unsubscribe();
  }

  const sub = await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: appServerKey,
  });
  await syncSubscription(sub);
}

// disablePush removes the local subscription and tells the server to forget it.
export async function disablePush(): Promise<void> {
  if (!pushSupported()) return;
  const reg = await navigator.serviceWorker.ready;
  const sub = await reg.pushManager.getSubscription();
  if (!sub) return;
  await api.unsubscribePush({ endpoint: sub.endpoint });
  await sub.unsubscribe();
}

async function syncSubscription(sub: PushSubscription): Promise<void> {
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
