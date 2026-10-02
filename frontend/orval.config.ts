import { defineConfig } from "orval"

// ponytail: assumes backend is checked out one level up (monorepo sibling
// layout); update this path if that ever changes.
// Huma emits repeatable query params as type ["array","null"], which orval's
// fetch client doesn't recognise as arrays: it would join values with commas
// instead of repeating the param (?rarityId=a&rarityId=b). Collapse to plain
// array so the generated URL builder explodes them.
interface OpParam {
  in: string
  name: string
  schema?: { type?: string | string[] }
}
interface OpenApiSpec {
  paths?: Record<
    string,
    Record<string, { parameters?: OpParam[] } | undefined> | undefined
  >
}

const repeatable = ["expansionSetId", "rarityId", "category", "tag"]

const input = {
  target: "../backend/openapi.json",
  override: {
    transformer: (spec: OpenApiSpec) => {
      const rewritten = new Set<string>()
      for (const path of Object.values(spec.paths ?? {})) {
        for (const op of Object.values(path ?? {})) {
          for (const param of op?.parameters ?? []) {
            if (
              param.in === "query" &&
              Array.isArray(param.schema?.type) &&
              param.schema.type.includes("array")
            ) {
              param.schema.type = "array"
              rewritten.add(param.name)
            }
          }
        }
      }
      // Fail loudly if Huma's output changes, else arrays silently go back to comma-joined.
      const missing = repeatable.filter((name) => !rewritten.has(name))
      if (missing.length > 0) {
        throw new Error(
          `orval transformer rewrote no array type for: ${missing.join(", ")}`
        )
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
