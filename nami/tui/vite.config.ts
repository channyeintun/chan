import { defineConfig } from "vite-plus";

export default defineConfig({
  pack: {
    entry: ["src/index.tsx", "src/standalone.tsx"],
    dts: false,
    sourcemap: true,
  },
});
