import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";
import federation from "@originjs/vite-plugin-federation";

/**
 * Admin dashboard build config.
 *
 * Exposes the admin app as a federated remote so the unified kiosk shell can
 * mount it in supervisor-override mode, while also building a standalone
 * index.html entry for direct admin deployments and E2E tests.
 */
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: "astra_admin",
      filename: "remoteEntry.js",
      exposes: { "./AdminApp": "./src/App.tsx" },
      shared: ["react", "react-dom", "@tanstack/react-query", "@apollo/client"],
    }),
  ],
  server: { port: 5174, strictPort: true, cors: true },
  build: {
    target: "es2022",
    modulePreload: false,
    cssCodeSplit: true,
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes("node_modules/react-dom") || id.includes("node_modules/react/")) return "vendor-react";
          if (id.includes("node_modules/react-router-dom")) return "vendor-router";
          if (id.includes("node_modules/@tanstack") || id.includes("node_modules/@apollo") || id.includes("node_modules/graphql")) return "vendor-data";
          if (id.includes("node_modules/d3")) return "vendor-d3";
          return undefined;
        },
      },
    },
    chunkSizeWarningLimit: 400,
  },
});
