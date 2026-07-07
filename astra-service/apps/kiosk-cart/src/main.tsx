import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import CartApp from "./CartApp";

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Fatal: #root element missing — kiosk cart cannot boot.");
}

createRoot(rootEl).render(
  <StrictMode>
    <CartApp onBackToMenu={() => undefined} onProceedToPayment={() => undefined} />
  </StrictMode>,
);
