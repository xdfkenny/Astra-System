import type { ReactElement, ReactNode } from "react";

export interface TailwindProviderProps {
  children: ReactNode;
  className?: string;
}

// Scopes the Living Weave base surface to a subtree. Global Tailwind utilities
// are provided by the design-system stylesheet; this provider establishes the
// root theming element and forwards an optional className for layout control.
export default function TailwindProvider({
  children,
  className,
}: TailwindProviderProps): ReactElement {
  const rootClass = className ? `astra-root ${className}` : "astra-root";
  return <div className={rootClass}>{children}</div>;
}
