/// <reference types="vitest/config" />
import { defineConfig } from "vite"
import { tanstackStart } from "@tanstack/react-start/plugin/vite"
import viteReact from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  plugins: [
    tailwindcss(),
    tanstackStart({ spa: { enabled: true } }),
    viteReact(),
  ],
  test: {
    environment: "jsdom",
    // Cold route-module imports in a test can pass 5s when the suite runs in parallel.
    testTimeout: 15_000,
    setupFiles: ["./src/test-setup.ts"],
  },
})

export default config
