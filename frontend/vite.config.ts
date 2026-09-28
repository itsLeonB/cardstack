/// <reference types="vitest/config" />
import { defineConfig } from "vite"
import { devtools } from "@tanstack/devtools-vite"
import { tanstackStart } from "@tanstack/react-start/plugin/vite"
import viteReact from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  plugins: [
    devtools(),
    tailwindcss(),
    tanstackStart({ spa: { enabled: true } }),
    viteReact(),
  ],
  test: {
    environment: "jsdom",
    // Playwright e2e specs live under e2e/ and run via `playwright test`,
    // not vitest — exclude them so vitest's default *.spec.ts glob doesn't
    // also try (and fail) to run them as unit tests.
    exclude: ["node_modules/**", "e2e/**"],
  },
})

export default config
