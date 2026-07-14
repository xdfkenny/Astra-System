import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { apiClient } from "./state/apiClient";
import "./state/cartService"; // Initialize cart service via import side effects

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Fatal: #root element missing — kiosk boot cannot continue.");
}

// Initialize API client
apiClient.checkHealth().catch((error: unknown) => {
  console.warn("API health check failed, running in offline mode:", error);
});

// Initialize cart service (happens automatically in constructor)

createRoot(rootEl).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

// Register the service worker for offline resilience (Background Sync API).
// Deferred to `load` so it never competes with first-paint on kiosk boot.
// Listens for `controllerchange` to reload the page when a new SW takes over.
if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/service-worker.js").catch((err: unknown) => {
      console.error("Service worker registration failed", err);
    });
  });

  let reloading = false;
  navigator.serviceWorker.addEventListener("controllerchange", () => {
    if (reloading) return;
    reloading = true;
    window.location.reload();
  });
}

