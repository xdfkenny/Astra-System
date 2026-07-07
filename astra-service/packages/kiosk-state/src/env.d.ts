/**
 * Minimal import-meta types for packages that read `import.meta.env` at
 * compile time (e.g. `VITE_KIOSK_ID`). Kept intentionally small and local so
 * the package does not depend on the full Vite toolchain just for env typing.
 */
interface ImportMetaEnv {
  readonly VITE_KIOSK_ID?: string;
  readonly DEV?: boolean;
  readonly PROD?: boolean;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
