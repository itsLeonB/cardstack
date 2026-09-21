import { defineConfig } from "orval"

// ponytail: assumes backend is checked out one level up (monorepo sibling
// layout); update this path if that ever changes.
const input = { target: "../backend/openapi.json" }

export default defineConfig({
  cardstack: {
    input,
    output: {
      mode: "tags-split",
      target: "src/generated/endpoints",
      schemas: "src/generated/models",
      client: "react-query",
      httpClient: "fetch",
      baseUrl: {
        runtime: "import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'",
      },
      clean: true,
    },
  },
  cardstackZod: {
    input,
    output: {
      mode: "tags-split",
      target: "src/generated/endpoints",
      client: "zod",
      fileExtension: ".zod.ts",
    },
  },
})
