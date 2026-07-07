import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Fatal: #root element missing — admin app cannot continue.");
}

createRoot(rootEl).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
