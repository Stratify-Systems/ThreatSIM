import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
        configure: (proxy) => {
          proxy.on("error", (err) => {
            if (err.message && err.message.includes("EPIPE")) {
              // Ignore broken pipe errors which happen frequently during reconnects
              return;
            }
            console.log("proxy error", err.message);
          });
        },
      },
    },
  },
});
