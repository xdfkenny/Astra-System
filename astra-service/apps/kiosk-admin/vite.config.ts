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
        manualChunks: {
          "vendor-react": ["react", "react-dom"],
          "vendor-router": ["react-router-dom"],
          "vendor-data": ["@tanstack/react-query", "@apollo/client", "graphql"],
          "vendor-d3": ["d3"],
        },
      },
    },
    chunkSizeWarningLimit: 400,
  },
});
