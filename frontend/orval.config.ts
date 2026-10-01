import { defineConfig } from "orval"

// ponytail: assumes backend is checked out one level up (monorepo sibling
// layout); update this path if that ever changes.
// Huma emits repeatable query params as type ["array","null"], which orval's
// fetch client doesn't recognise as arrays: it would join values with commas
// instead of repeating the param (?rarityId=a&rarityId=b). Collapse to plain
// array so the generated URL builder explodes them.
const input = {
  target: "../backend/openapi.json",
  override: {
    transformer: (spec: any) => {
      for (const path of Object.values<any>(spec.paths ?? {})) {
        for (const op of Object.values<any>(path)) {
          for (const param of op?.parameters ?? []) {
            if (
              param.in === "query" &&
              Array.isArray(param.schema?.type) &&
              param.schema.type.includes("array")
            ) {
              param.schema.type = "array"
            }
          }
        }
      }
      return spec
    },
  },
}

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
      override: {
        mutator: {
          path: "./src/lib/http.ts",
          name: "customFetch",
        },
      },
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
