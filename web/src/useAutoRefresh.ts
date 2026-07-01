import { useEffect, useRef } from "react";

// useAutoRefresh keeps a runs view live without a manual reload. It calls
// `onRefresh`:
//   - on a timer (default every 20s),
//   - whenever the service worker forwards a Web Push run event (the sw.js
//     "push" handler posts `{ type: "run_update" }` to open clients), and
//   - when the tab becomes visible again (e.g. reopening the PWA after a push).
//
// The callback is held in a ref so listeners never go stale even if `onRefresh`
// is redefined each render.
export function useAutoRefresh(onRefresh: () => void, intervalMs = 20000): void {
  const cb = useRef(onRefresh);
  cb.current = onRefresh;

  useEffect(() => {
    const fire = () => cb.current();

    const id = window.setInterval(fire, intervalMs);

    const onMessage = (e: MessageEvent) => {
      if (e.data && e.data.type === "run_update") fire();
    };
    navigator.serviceWorker?.addEventListener("message", onMessage);

    const onVisible = () => {
      if (document.visibilityState === "visible") fire();
    };
    document.addEventListener("visibilitychange", onVisible);

    return () => {
      window.clearInterval(id);
      navigator.serviceWorker?.removeEventListener("message", onMessage);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [intervalMs]);
}
