// vite.config.ts
import { defineConfig } from "file:///C:/Users/xdfke/Desktop/DEV/Astra-System/astra-service/node_modules/.pnpm/vite@6.4.3_@types+node@22.20.0_jiti@2.7.0_lightningcss@1.32.0_terser@5.48.0/node_modules/vite/dist/node/index.js";
import react from "file:///C:/Users/xdfke/Desktop/DEV/Astra-System/astra-service/node_modules/.pnpm/@vitejs+plugin-react-swc@3.11.0_vite@6.4.3_@types+node@22.20.0_jiti@2.7.0_lightningcss@1.32.0_terser@5.48.0_/node_modules/@vitejs/plugin-react-swc/index.js";
import tailwindcss from "file:///C:/Users/xdfke/Desktop/DEV/Astra-System/astra-service/node_modules/.pnpm/@tailwindcss+vite@4.3.2_vite@6.4.3_@types+node@22.20.0_jiti@2.7.0_lightningcss@1.32.0_terser@5.48.0_/node_modules/@tailwindcss/vite/dist/index.mjs";
import federation from "file:///C:/Users/xdfke/Desktop/DEV/Astra-System/astra-service/node_modules/.pnpm/@originjs+vite-plugin-federation@1.4.1/node_modules/@originjs/vite-plugin-federation/dist/index.mjs";
var vite_config_default = defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: "astra_menu",
      filename: "remoteEntry.js",
      exposes: { "./MenuApp": "./src/MenuApp.tsx" },
      shared: ["react", "react-dom", "@tanstack/react-query"]
    })
  ],
  server: { port: 5171, strictPort: true, cors: true },
  build: { target: "es2022", modulePreload: false, cssCodeSplit: true }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCJDOlxcXFxVc2Vyc1xcXFx4ZGZrZVxcXFxEZXNrdG9wXFxcXERFVlxcXFxBc3RyYS1TeXN0ZW1cXFxcYXN0cmEtc2VydmljZVxcXFxhcHBzXFxcXGtpb3NrLW1lbnVcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIkM6XFxcXFVzZXJzXFxcXHhkZmtlXFxcXERlc2t0b3BcXFxcREVWXFxcXEFzdHJhLVN5c3RlbVxcXFxhc3RyYS1zZXJ2aWNlXFxcXGFwcHNcXFxca2lvc2stbWVudVxcXFx2aXRlLmNvbmZpZy50c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vQzovVXNlcnMveGRma2UvRGVza3RvcC9ERVYvQXN0cmEtU3lzdGVtL2FzdHJhLXNlcnZpY2UvYXBwcy9raW9zay1tZW51L3ZpdGUuY29uZmlnLnRzXCI7aW1wb3J0IHsgZGVmaW5lQ29uZmlnIH0gZnJvbSBcInZpdGVcIjtcclxuaW1wb3J0IHJlYWN0IGZyb20gXCJAdml0ZWpzL3BsdWdpbi1yZWFjdC1zd2NcIjtcclxuaW1wb3J0IHRhaWx3aW5kY3NzIGZyb20gXCJAdGFpbHdpbmRjc3Mvdml0ZVwiO1xyXG5pbXBvcnQgZmVkZXJhdGlvbiBmcm9tIFwiQG9yaWdpbmpzL3ZpdGUtcGx1Z2luLWZlZGVyYXRpb25cIjtcclxuXHJcbmV4cG9ydCBkZWZhdWx0IGRlZmluZUNvbmZpZyh7XHJcbiAgcGx1Z2luczogW1xyXG4gICAgcmVhY3QoKSxcclxuICAgIHRhaWx3aW5kY3NzKCksXHJcbiAgICBmZWRlcmF0aW9uKHtcclxuICAgICAgbmFtZTogXCJhc3RyYV9tZW51XCIsXHJcbiAgICAgIGZpbGVuYW1lOiBcInJlbW90ZUVudHJ5LmpzXCIsXHJcbiAgICAgIGV4cG9zZXM6IHsgXCIuL01lbnVBcHBcIjogXCIuL3NyYy9NZW51QXBwLnRzeFwiIH0sXHJcbiAgICAgIHNoYXJlZDogW1wicmVhY3RcIiwgXCJyZWFjdC1kb21cIiwgXCJAdGFuc3RhY2svcmVhY3QtcXVlcnlcIl0sXHJcbiAgICB9KSxcclxuICBdLFxyXG4gIHNlcnZlcjogeyBwb3J0OiA1MTcxLCBzdHJpY3RQb3J0OiB0cnVlLCBjb3JzOiB0cnVlIH0sXHJcbiAgYnVpbGQ6IHsgdGFyZ2V0OiBcImVzMjAyMlwiLCBtb2R1bGVQcmVsb2FkOiBmYWxzZSwgY3NzQ29kZVNwbGl0OiB0cnVlIH0sXHJcbn0pO1xyXG4iXSwKICAibWFwcGluZ3MiOiAiO0FBQW1aLFNBQVMsb0JBQW9CO0FBQ2hiLE9BQU8sV0FBVztBQUNsQixPQUFPLGlCQUFpQjtBQUN4QixPQUFPLGdCQUFnQjtBQUV2QixJQUFPLHNCQUFRLGFBQWE7QUFBQSxFQUMxQixTQUFTO0FBQUEsSUFDUCxNQUFNO0FBQUEsSUFDTixZQUFZO0FBQUEsSUFDWixXQUFXO0FBQUEsTUFDVCxNQUFNO0FBQUEsTUFDTixVQUFVO0FBQUEsTUFDVixTQUFTLEVBQUUsYUFBYSxvQkFBb0I7QUFBQSxNQUM1QyxRQUFRLENBQUMsU0FBUyxhQUFhLHVCQUF1QjtBQUFBLElBQ3hELENBQUM7QUFBQSxFQUNIO0FBQUEsRUFDQSxRQUFRLEVBQUUsTUFBTSxNQUFNLFlBQVksTUFBTSxNQUFNLEtBQUs7QUFBQSxFQUNuRCxPQUFPLEVBQUUsUUFBUSxVQUFVLGVBQWUsT0FBTyxjQUFjLEtBQUs7QUFDdEUsQ0FBQzsiLAogICJuYW1lcyI6IFtdCn0K
