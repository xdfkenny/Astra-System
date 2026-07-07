import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";
import federation from "@originjs/vite-plugin-federation";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: "astra_payment",
      filename: "remoteEntry.js",
      exposes: { "./PaymentApp": "./src/PaymentApp.tsx" },
      shared: ["react", "react-dom"],
    }),
  ],
  server: { port: 5173, strictPort: true, cors: true },
  build: { target: "es2022", modulePreload: false, cssCodeSplit: true },
});
