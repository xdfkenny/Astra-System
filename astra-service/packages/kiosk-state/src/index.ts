/**
 * `@astra/kiosk-state` is the ONE shared state package imported by both the
 * kiosk-shell host and every federated micro-frontend (menu/cart/payment).
 *
 * WHY a real workspace package instead of Module Federation's `shared`
 * singleton mechanism for this: Zustand/Valtio store *instances* must be
 * the literal same module instance across federation boundaries, and
 * @originjs/vite-plugin-federation's singleton sharing is version-matched
 * at runtime but still fragile across independently-deployed remotes with
 * independent release cadences. Publishing this as a versioned internal
 * package and having each remote depend on a pinned version means a state
 * shape change is a deliberate, reviewed contract change (a package.json
 * bump), not a runtime surprise when Menu ships before Cart does.
 */
export * from "./cartProxy";
export * from "./sessionStore";
