import { defineConfig } from "orval"

// The Admin app generates only what it calls: the admin routes and the two
// catalog lists its pickers read. A new call from the Admin app means adding
// its operationId here, and the transformer below fails loudly if one is
// misspelt, because orval would otherwise generate nothing for it.
const operations = {
  "admin-check": {},
  "get-admin-card": {},
  "create-admin-card": {},
  "update-admin-card": {},
  "delete-admin-card": {},
  "request-admin-card-image-upload": {},
  "confirm-admin-card-image": {},
  "create-expansion-set": {},
  "update-expansion-set": {},
  "list-catalog-rarities": {},
  "list-catalog-series": {},
}

interface OpParam {
  in: string
  name: string
  schema?: { type?: string | string[] }
}
interface Operation {
  operationId?: string
  parameters?: OpParam[]
}
type PathItem = Record<string, Operation | undefined>
// Path items are typed `object` here: orval's own spec type fails a stricter
// shape on `$ref`, and the loop below reads them only through METHODS.
interface OpenApiSpec {
  // oxlint-disable-next-line anti-slop/no-unsafe-dictionary-type
  paths?: Record<string, object>
}

const METHODS = ["get", "put", "post", "delete", "patch"]

// orval's input filters select by tag only, so the operations are cut here.
const input = {
  target: "../backend/openapi.json",
  override: {
    transformer: <T extends OpenApiSpec>(spec: T): T => {
      const seen = new Set<string>()
      for (const [route, value] of Object.entries(spec.paths ?? {})) {
        // SAFETY: an OpenAPI path item maps method names to operations, plus
        // `parameters` and `$ref`; the loop reads it only through METHODS.
        const item = value as PathItem
        for (const [method, op] of Object.entries(item)) {
          if (!METHODS.includes(method) || !op) continue
          if (op.operationId) seen.add(op.operationId)
          if (!op.operationId || !(op.operationId in operations)) {
            delete item?.[method]
            continue
          }
          for (const param of op.parameters ?? []) {
            // Huma emits repeatable query params as ["array","null"], which
            // orval's fetch client would comma-join instead of repeating.
            if (
              param.in === "query" &&
              Array.isArray(param.schema?.type) &&
              param.schema.type.includes("array")
            ) {
              param.schema.type = "array"
            }
          }
        }
        if (item && !METHODS.some((method) => method in item)) {
          delete spec.paths?.[route]
        }
      }
      const unknown = Object.keys(operations).filter((id) => !seen.has(id))
      if (unknown.length > 0) {
        throw new Error(
          `orval config names operations the spec does not have: ${unknown.join(", ")}`
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
        operations,
      },
    },
  },
})
