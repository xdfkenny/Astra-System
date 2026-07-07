import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";
import federation from "@originjs/vite-plugin-federation";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: "astra_cart",
      filename: "remoteEntry.js",
      exposes: { "./CartApp": "./src/CartApp.tsx" },
      shared: ["react", "react-dom"],
    }),
  ],
  server: { port: 5172, strictPort: true, cors: true },
  build: { target: "es2022", modulePreload: false, cssCodeSplit: true },
});
