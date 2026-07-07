/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_KIOSK_ID?: string;
  readonly VITE_API_GATEWAY_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
