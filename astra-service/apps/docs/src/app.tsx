import { Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import ApiReferencePage from "./pages/api-reference.mdx";
import ArchitecturePage from "./pages/architecture.mdx";
import IndexPage from "./pages/index.mdx";
import KioskFlowPage from "./pages/kiosk-flow.mdx";
import OfflineModePage from "./pages/offline-mode.mdx";
import PaymentFlowPage from "./pages/payment-flow.mdx";
import P2pSyncPage from "./pages/p2p-sync.mdx";
import RunbooksIndexPage from "./pages/runbooks/index.mdx";
import SecurityPage from "./pages/security.mdx";

export function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<IndexPage />} />
        <Route path="/architecture" element={<ArchitecturePage />} />
        <Route path="/kiosk-flow" element={<KioskFlowPage />} />
        <Route path="/offline-mode" element={<OfflineModePage />} />
        <Route path="/p2p-sync" element={<P2pSyncPage />} />
        <Route path="/payment-flow" element={<PaymentFlowPage />} />
        <Route path="/security" element={<SecurityPage />} />
        <Route path="/api-reference" element={<ApiReferencePage />} />
        <Route path="/runbooks" element={<RunbooksIndexPage />} />
      </Routes>
    </Layout>
  );
}
