import { useState, type ReactNode } from "react";
import { cn } from "@astra/design-system/utils";
import { Button } from "@astra/design-system";
import { Nav } from "./Nav";
import { ThemeToggle } from "./ThemeToggle";
import { useAuth } from "./AuthProvider";

interface LayoutProps {
  readonly children: ReactNode;
  readonly title: string;
}

export function Layout({ children, title }: LayoutProps): React.JSX.Element {
  const [open, setOpen] = useState(true);
  const { role } = useAuth();

  return (
    <div className="flex h-full flex-1 overflow-hidden bg-background">
      <aside
        className={cn(
          "flex h-full flex-col border-r border-border bg-surface transition-all",
          open ? "w-64" : "w-16",
        )}
      >
        <div className="flex h-16 items-center gap-2 border-b border-border px-4">
          <span className={cn("font-heading text-lg font-bold", !open && "hidden")}>Astra Admin</span>
          <Button variant="ghost" className="ml-auto min-h-8 px-2 py-1" onClick={() => { setOpen((s) => !s); }}>
            {open ? "‹" : "›"}
          </Button>
        </div>
        {open ? <Nav /> : null}
        <div className="mt-auto border-t border-border p-4">
          <div className="flex items-center gap-2 text-xs text-ink-muted">
            <span className="font-semibold uppercase">{role}</span>
          </div>
          {open ? <div className="mt-3"><ThemeToggle /></div> : null}
        </div>
      </aside>
      <main className="flex flex-1 flex-col overflow-hidden">
        <header className="flex h-16 items-center border-b border-border bg-surface px-6">
          <h1 className="font-heading text-xl font-bold">{title}</h1>
        </header>
        <div className="flex-1 overflow-y-auto p-6">{children}</div>
      </main>
    </div>
  );
}
