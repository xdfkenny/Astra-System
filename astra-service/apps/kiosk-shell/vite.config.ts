import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";
import federation from "@originjs/vite-plugin-federation";
import { VitePWA } from "vite-plugin-pwa";

/**
 * Kiosk shell build config.
 *
 * Module Federation: the shell is the "host". Menu/Cart/Payment/Admin are
 * independently deployable "remotes" — each can ship a hotfix (e.g. a menu
 * pricing bug) without redeploying or restarting the whole kiosk process,
 * which matters when a physical restart means a lane goes dark for 90s.
 *
 * Bundle budget enforced by rollup-plugin-visualizer in CI (see build.rollupOptions
 * and .github/workflows/ci.yml `bundle-budget` job): main chunk < 150KB gz.
 */
export default defineConfig(({ mode }) => ({
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: "astra_kiosk_shell",
      remotes: {
        astra_menu: mode === "production"
          ? "https://cdn.astra-service.internal/menu/remoteEntry.js"
          : "http://localhost:5171/assets/remoteEntry.js",
        astra_cart: mode === "production"
          ? "https://cdn.astra-service.internal/cart/remoteEntry.js"
          : "http://localhost:5172/assets/remoteEntry.js",
        astra_payment: mode === "production"
          ? "https://cdn.astra-service.internal/payment/remoteEntry.js"
          : "http://localhost:5173/assets/remoteEntry.js",
        astra_admin: mode === "production"
          ? "https://cdn.astra-service.internal/admin/remoteEntry.js"
          : "http://localhost:5174/assets/remoteEntry.js",
      },
      shared: ["react", "react-dom", "zustand", "@tanstack/react-query"],
    }),
    VitePWA({
      registerType: "autoUpdate",
      strategies: "injectManifest",
      srcDir: "src/workers",
      filename: "service-worker.ts",
      injectManifest: { swSrc: "src/workers/service-worker.ts" },
      manifest: {
        name: "Astra-Service Kiosk",
        short_name: "Astra",
        display: "fullscreen",
        orientation: "portrait",
        background_color: "#F8F9FC",
        theme_color: "#4F6D7A",
        icons: [],
      },
    }),
  ],
  server: {
    port: 5170,
    strictPort: true,
    cors: true,
  },
  preview: {
    port: 4173,
  },
  build: {
    target: "es2022",
    modulePreload: false,
    cssCodeSplit: true,
    rollupOptions: {
      output: {
        // Keep vendor CRDT/crypto WASM glue in its own chunk so it can be
        // cached long-term independently of frequently-changing app code.
        manualChunks: {
          "vendor-react": ["react", "react-dom", "react-router-dom"],
          "vendor-state": ["zustand", "valtio", "@tanstack/react-query"],
          "vendor-motion": ["framer-motion"],
        },
      },
    },
    chunkSizeWarningLimit: 150,
  },
}));
