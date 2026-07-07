import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Fatal: #root element missing — kiosk boot cannot continue.");
}

createRoot(rootEl).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

// Register the service worker for offline resilience (Background Sync API).
// Deferred to `load` so it never competes with first-paint on kiosk boot.
if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/service-worker.js").catch((err: unknown) => {
      console.error("Service worker registration failed", err);
    });
  });
}
