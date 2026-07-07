import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import MenuApp from "./MenuApp";

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Fatal: #root element missing — kiosk menu cannot boot.");
}

const queryClient = new QueryClient();

createRoot(rootEl).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <MenuApp laneMode="full" silentAssistArmed={false} onSelectItem={() => undefined} />
    </QueryClientProvider>
  </StrictMode>,
);
