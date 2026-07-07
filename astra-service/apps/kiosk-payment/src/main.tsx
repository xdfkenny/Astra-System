import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import PaymentApp from "./PaymentApp";

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Fatal: #root element missing — kiosk payment cannot boot.");
}

createRoot(rootEl).render(
  <StrictMode>
    <PaymentApp onResult={() => undefined} onCancel={() => undefined} />
  </StrictMode>,
);
