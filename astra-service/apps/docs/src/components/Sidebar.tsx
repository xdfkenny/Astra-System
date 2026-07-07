import { NavLink } from "react-router-dom";

const routes = [
  { path: "/", label: "Introduction" },
  { path: "/architecture", label: "Architecture" },
  { path: "/kiosk-flow", label: "Kiosk Flow" },
  { path: "/offline-mode", label: "Offline Mode" },
  { path: "/p2p-sync", label: "P2P Sync" },
  { path: "/payment-flow", label: "Payment Flow" },
  { path: "/security", label: "Security" },
  { path: "/api-reference", label: "API Reference" },
  { path: "/runbooks", label: "Runbooks" },
];

export function Sidebar() {
  return (
    <aside className="docs-sidebar">
      <h1 className="docs-sidebar__title">Astra-Service Docs</h1>
      <nav>
        <ul className="docs-sidebar__nav">
          {routes.map((route) => (
            <li key={route.path}>
              <NavLink
                to={route.path}
                end={route.path === "/"}
                className={({ isActive }) =>
                  isActive
                    ? "docs-sidebar__link docs-sidebar__link--active"
                    : "docs-sidebar__link"
                }
              >
                {route.label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  );
}
