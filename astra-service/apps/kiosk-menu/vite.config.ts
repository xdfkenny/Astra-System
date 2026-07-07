import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";
import federation from "@originjs/vite-plugin-federation";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: "astra_menu",
      filename: "remoteEntry.js",
      exposes: { "./MenuApp": "./src/MenuApp.tsx" },
      shared: ["react", "react-dom", "@tanstack/react-query"],
    }),
  ],
  server: { port: 5171, strictPort: true, cors: true },
  build: { target: "es2022", modulePreload: false, cssCodeSplit: true },
});
