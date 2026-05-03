/**
 * api-paths.ts
 *
 * Type helpers for the OpenAPI-derived `paths` interface in
 * `src/shared/types/api.d.ts`. Used by `apiClient<T>(path)` call sites
 * to derive the response type from the OpenAPI schema instead of
 * hand-writing it. Keeps the call site short:
 *
 *     // before — duplicated, drift-prone
 *     apiClient<Get<"/api/v1/auth/me"> & MeResponse>("/api/v1/auth/me")
 *
 *     // after — derived from the contract
 *     apiClient<Get<"/api/v1/auth/me">>("/api/v1/auth/me")
 *
 * The helpers are pure type-level — they emit nothing at runtime. If
 * the OpenAPI document declares no 200 application/json response for
 * a given (path, method) pair the helper resolves to `unknown` so
 * the call site fails type-check on the consumer side; this is the
 * "silent drift" we want to surface.
 */
import type { paths } from "../types/api"

// ---------------------------------------------------------------------------
// Internal: response extractor
// ---------------------------------------------------------------------------

type JSONResponse<T> = T extends {
  responses: {
    200: { content: { "application/json": infer R } }
  }
}
  ? R
  : T extends {
      responses: {
        201: { content: { "application/json": infer R } }
      }
    }
  ? R
  : T extends {
      responses: {
        202: { content: { "application/json": infer R } }
      }
    }
  ? R
  : never

// ---------------------------------------------------------------------------
// Public: per-method extractors
// ---------------------------------------------------------------------------

/**
 * Get<P> resolves to the application/json 200/201/202 response shape for
 * `paths[P]["get"]`. Use it on every `apiClient<T>(path)` call site
 * issuing a GET so the type stays in lock-step with the OpenAPI contract.
 */
export type Get<P extends keyof paths> = paths[P] extends { get: infer Op }
  ? JSONResponse<Op>
  : never

/**
 * Post<P> mirrors Get for POST endpoints. Resolves to the response
 * shape; the request body is typed via `PostBody<P>` when needed.
 */
export type Post<P extends keyof paths> = paths[P] extends { post: infer Op }
  ? JSONResponse<Op>
  : never

/** Put<P> — response shape for the PUT method on `paths[P]`. */
export type Put<P extends keyof paths> = paths[P] extends { put: infer Op }
  ? JSONResponse<Op>
  : never

/** Patch<P> — response shape for the PATCH method on `paths[P]`. */
export type Patch<P extends keyof paths> = paths[P] extends { patch: infer Op }
  ? JSONResponse<Op>
  : never

/**
 * Delete<P> — response shape for the DELETE method. Most DELETE
 * endpoints respond with 204 No Content; this helper resolves to
 * `void` in that case so the call site can write `apiClient<DeleteOf<...>>(...)`
 * without forcing a `never`.
 */
export type Delete<P extends keyof paths> = paths[P] extends {
  delete: infer Op
}
  ? JSONResponse<Op> extends never
    ? void
    : JSONResponse<Op>
  : never

// ---------------------------------------------------------------------------
// Path-only validation (no response shape inferred)
// ---------------------------------------------------------------------------

/**
 * Void<P> validates the path against `keyof paths` while resolving to
 * `void` — for call sites whose caller does not consume the response
 * body. Equivalent to:
 *
 *     apiClient<Void<"/api/v1/foo">>(path)
 *
 * The path string in the type generic is checked against the OpenAPI
 * contract via the `P extends keyof paths` constraint; the runtime
 * path can still be a templated string.
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export type Void<P extends keyof paths> = void

// ---------------------------------------------------------------------------
// Helpers for request bodies (less commonly needed but offered for
// symmetry — `apiClient` accepts a typed body via `options.body`).
// ---------------------------------------------------------------------------

type RequestJSONBody<T> = T extends {
  requestBody: { content: { "application/json": infer B } }
}
  ? B
  : never

/** PostBody<P> — request body type for `paths[P]["post"]`. */
export type PostBody<P extends keyof paths> = paths[P] extends { post: infer Op }
  ? RequestJSONBody<Op>
  : never

/** PutBody<P> — request body type for `paths[P]["put"]`. */
export type PutBody<P extends keyof paths> = paths[P] extends { put: infer Op }
  ? RequestJSONBody<Op>
  : never

/** PatchBody<P> — request body type for `paths[P]["patch"]`. */
export type PatchBody<P extends keyof paths> = paths[P] extends {
  patch: infer Op
}
  ? RequestJSONBody<Op>
  : never
