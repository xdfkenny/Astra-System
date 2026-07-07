import type { ReactNode } from "react";
import { Sidebar } from "./Sidebar";

export function Layout({ children }: { readonly children: ReactNode }) {
  return (
    <div className="docs-layout">
      <Sidebar />
      <main className="docs-main">{children}</main>
    </div>
  );
}
