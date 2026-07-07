import mdx from "@mdx-js/rollup";
import react from "@vitejs/plugin-react-swc";
import base from "@astra/config/vite";
import { defineConfig } from "vite";

export default defineConfig({
  ...base,
  plugins: [react(), mdx({ providerImportSource: "@mdx-js/react" })],
  server: {
    port: 5175,
    strictPort: false,
  },
  preview: {
    port: 4175,
  },
});
