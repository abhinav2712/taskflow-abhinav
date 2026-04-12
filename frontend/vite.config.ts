import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      api: path.resolve(__dirname, "src/api"),
      components: path.resolve(__dirname, "src/components"),
      pages: path.resolve(__dirname, "src/pages"),
      store: path.resolve(__dirname, "src/store"),
      types: path.resolve(__dirname, "src/types"),
    },
  },
});
