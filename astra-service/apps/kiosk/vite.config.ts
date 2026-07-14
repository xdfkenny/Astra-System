import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";
import federation from "@originjs/vite-plugin-federation";
import { VitePWA } from "vite-plugin-pwa";

/**
 * Kiosk host build config.
 *
 * Native Federation: the unified kiosk is the host. The menu/cart/payment
 * micro-frontends are independently-deployed remotes — each can ship a hotfix
 * without redeploying the whole kiosk process, which matters when a physical
 * restart means a lane goes dark for ~90s. The shell also exposes `./Shell`
 * so it can be embedded by an outer orchestrator (e.g. a drive-thru preview).
 */
export default defineConfig(({ mode }) => ({
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: "astra_kiosk",
      filename: "remoteEntry.js",
      exposes: {
        "./Shell": "./src/App.tsx",
      },
      remotes: {
        astra_menu:
          mode === "production"
            ? "https://cdn.astra-service.internal/menu/remoteEntry.js"
            : "http://localhost:5171/assets/remoteEntry.js",
        astra_cart:
          mode === "production"
            ? "https://cdn.astra-service.internal/cart/remoteEntry.js"
            : "http://localhost:5172/assets/remoteEntry.js",
        astra_payment:
          mode === "production"
            ? "https://cdn.astra-service.internal/payment/remoteEntry.js"
            : "http://localhost:5173/assets/remoteEntry.js",
      },
      shared: [
        "react",
        "react-dom",
        "valtio",
        "zustand",
        "@tanstack/react-query",
      ],
    }),
    VitePWA({
      registerType: "autoUpdate",
      strategies: "injectManifest",
      srcDir: "src/workers",
      filename: "service-worker.ts",
      injectManifest: { swSrc: "src/workers/service-worker.ts" },
      manifest: {
        name: "Astra-Service Unified Kiosk",
        short_name: "Astra Kiosk",
        display: "fullscreen",
        orientation: "portrait",
        background_color: "#f8fafc",
        theme_color: "#0d9488",
        icons: [],
      },
    }),
  ],
  server: {
    port: 5180,
    strictPort: true,
    cors: true,
  },
  preview: {
    port: 4180,
  },
  build: {
    target: "es2022",
    modulePreload: false,
    cssCodeSplit: true,
    rollupOptions: {
      output: {
        manualChunks: {
          "vendor-react": ["react", "react-dom"],
          "vendor-state": ["valtio", "zustand", "@tanstack/react-query"],
          "vendor-motion": ["framer-motion"],
        },
      },
    },
    chunkSizeWarningLimit: 180,
  },
}));
