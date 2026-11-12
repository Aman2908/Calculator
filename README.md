# Calculator

A full-stack calculator: a **Go** REST API and a **React + TypeScript** single-page app, packaged as one ~15 MB Docker image.

- Operations: add, subtract, multiply, divide, power, square root, percent
- Validation and typed errors on both sides (division by zero, √ of a negative, overflow, bad input)
- One OpenAPI contract ([`api/openapi.yaml`](api/openapi.yaml)) drives the generated TypeScript types and the Go contract tests
- Responsive, keyboard-accessible UI with light and dark themes

The full technical plan is in [`docs/TPD.md`](docs/TPD.md).

---

## Quick start (Docker only)

```bash
docker build -t calculator .
docker run --rm -p 8080:8080 calculator
# open http://localhost:8080
```

The image serves the SPA and the API from the same origin, so no CORS setup is needed.

## Local development

Prerequisites: **Go 1.26+**, **Node 24+**, and `make`. On macOS: `brew install go node`.

```bash
make install   # npm ci + go mod download
make dev       # Go API on :8080, Vite on :5173 (proxies /api to :8080)
```

Open http://localhost:5173. To run the two halves separately:

```bash
cd backend  && go run ./cmd/server     # API on :8080
cd frontend && npm run dev             # UI  on :5173
```

| Command | What it does |
| --- | --- |
| `make test` | Go tests (with `-race`) and Vitest |
| `make coverage` | Coverage reports in `coverage/backend/index.html` and `coverage/frontend/index.html` |
| `make lint` | gofmt, go vet, golangci-lint, tsc, ESLint, Prettier |
| `make gen` | Regenerate `frontend/src/api/schema.d.ts` from the OpenAPI spec |
| `make docker-build docker-run` | Build and run the production image |

Server configuration is read from the environment:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | Listen port |
| `STATIC_DIR` | _(unset)_ | Directory with the built SPA; when set, served at `/` with an `index.html` fallback |
| `CORS_ORIGIN` | _(unset)_ | Single origin allowed to call the API cross-origin. Not needed when using the Vite proxy |

---

## API

Base URL: `http://localhost:8080`. The full schema is in [`api/openapi.yaml`](api/openapi.yaml).

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/v1/calculate` | Run one operation |
| `GET` | `/api/v1/operations` | List supported operations (name, symbol, arity, label) |
| `GET` | `/healthz` | Liveness probe |

### Request and response

```bash
curl -s -X POST localhost:8080/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"divide","operands":[10,4]}'
# 200 {"operation":"divide","operands":[10,4],"result":2.5}
```

| Operation | Operands | Meaning | Example → result |
| --- | --- | --- | --- |
| `add` | `[a, b]` | a + b | `[2, 3]` → `5` |
| `subtract` | `[a, b]` | a − b | `[2, 3]` → `-1` |
| `multiply` | `[a, b]` | a × b | `[6, 7]` → `42` |
| `divide` | `[a, b]` | a ÷ b | `[10, 4]` → `2.5` |
| `power` | `[a, b]` | a ^ b | `[2, 10]` → `1024` |
| `sqrt` | `[a]` | √a | `[16]` → `4` |
| `percent` | `[a, b]` | b% of a = a × b / 100 | `[200, 15]` → `30` |

More examples:

```bash
# Square root (unary)
curl -s -X POST localhost:8080/api/v1/calculate -H 'Content-Type: application/json' \
  -d '{"operation":"sqrt","operands":[16]}'
# 200 {"operation":"sqrt","operands":[16],"result":4}

# Division by zero
curl -s -X POST localhost:8080/api/v1/calculate -H 'Content-Type: application/json' \
  -d '{"operation":"divide","operands":[1,0]}'
# 422 {"error":{"code":"DIVISION_BY_ZERO","message":"cannot divide by zero"}}

# Square root of a negative number
curl -s -X POST localhost:8080/api/v1/calculate -H 'Content-Type: application/json' \
  -d '{"operation":"sqrt","operands":[-1]}'
# 422 {"error":{"code":"DOMAIN_ERROR","message":"cannot take the square root of a negative number"}}

# Wrong number of operands
curl -s -X POST localhost:8080/api/v1/calculate -H 'Content-Type: application/json' \
  -d '{"operation":"add","operands":[1]}'
# 400 {"error":{"code":"INVALID_OPERANDS","message":"wrong number of operands: add takes 2, got 1"}}

# Unknown operation
curl -s -X POST localhost:8080/api/v1/calculate -H 'Content-Type: application/json' \
  -d '{"operation":"modulo","operands":[5,2]}'
# 400 {"error":{"code":"UNSUPPORTED_OPERATION","message":"unsupported operation: \"modulo\""}}

# List operations
curl -s localhost:8080/api/v1/operations
```

### Errors

Every error has the same shape: `{"error": {"code": "...", "message": "..."}}`. `code` is stable and meant for programs; `message` is for people.

| HTTP | `code` | When |
| --- | --- | --- |
| 400 | `INVALID_JSON` | Malformed JSON, unknown fields, empty body, body over 1 KB, or more than one JSON value |
| 400 | `UNSUPPORTED_OPERATION` | Missing or unknown `operation` |
| 400 | `INVALID_OPERANDS` | Missing `operands`, wrong count, `null`, non-numeric, or out-of-range numbers |
| 404 | `NOT_FOUND` | Unknown API path |
| 405 | `METHOD_NOT_ALLOWED` | Wrong HTTP method (the `Allow` header lists the right one) |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | `Content-Type` is not `application/json` |
| 422 | `DIVISION_BY_ZERO` | Divisor is 0, or 0 raised to a negative power |
| 422 | `DOMAIN_ERROR` | √ of a negative, or a negative base with a fractional exponent |
| 422 | `NON_FINITE_RESULT` | The result overflows float64 (for example `10 ^ 400`) |
| 500 | `INTERNAL` | Unexpected server error. No internal details are exposed |

**400 vs 422:** 400 means the request is malformed. 422 means it is well formed but has no mathematical answer.

---

## Design decisions

- **One generic endpoint.** `POST /api/v1/calculate` takes `{operation, operands}`, so adding an operation means adding one registry entry, with no new route, handler, or client function. Per-operation routes (`/add`, `/divide`) would look more RESTful but repeat the same code seven times.
- **Pure calculator core.** `internal/calculator` has no HTTP or I/O. It holds a registry of `{name, symbol, arity, fn}` and one `Calculate` function that checks arity and rejects NaN and ±Inf for every operation in one place. Errors are sentinel values, and `internal/httpapi` maps them to status codes in a single function (`toAPIError`).
- **Standard library only on the server.** Go 1.22+ `net/http` routing handles methods and path patterns, and `log/slog` gives structured logs. The only third-party Go module, `kin-openapi`, is used in tests.
- **Defensive HTTP.** 1 KB body limit, unknown fields rejected, a single JSON value required, `null` operands told apart from 0, server timeouts, graceful shutdown on SIGTERM, panic recovery that returns a JSON 500, and an `X-Request-ID` on every response.
- **The contract is the source of truth.** `api/openapi.yaml` generates the frontend types (`npm run gen:api`). Go tests validate real responses against it, and CI fails if the generated types are stale.
- **Form UI, not a keypad.** Operand A, an operation picker, and operand B map one-to-one onto the API, need no expression parser, and are easy to validate and test. The picker uses native radio inputs, so keyboard and screen-reader support come built in.
- **Validate twice, trust the server.** The client catches empty or invalid numbers, ÷ 0, and √ of a negative for instant feedback. The server stays authoritative and re-validates everything.
- **Predictable request state.** `useCalculator` is a small `useReducer` state machine (`idle → submitting → success | error`). A new request aborts the one in flight, so a late response can never overwrite a newer result. The client turns every failure into an `ApiError` with a code (including `NETWORK_ERROR` and `TIMEOUT`).
- **Single image, single origin.** A three-stage Dockerfile (Node build, Go build, distroless runtime) produces a ~15 MB non-root image in which Go serves the SPA. In development, Vite proxies `/api`, so CORS is never needed.

### Assumptions

- Numbers are IEEE-754 float64 end to end, the same as JSON and JavaScript. The API returns the exact float; the UI rounds to 12 significant digits, so 0.1 + 0.2 shows as `0.3`.
- `percent(a, b)` means "b percent of a" (a × b / 100).
- `0 ^ 0 = 1`, following `math.Pow` and IEEE-754.
- Each request is one operation. Expression parsing, memory, and persistence are out of scope. History is kept in memory on the client and holds the last 10 results.

---

## Testing and coverage

```bash
make test       # all unit tests
make coverage   # HTML reports under ./coverage
```

| Layer | Tests | Coverage |
| --- | --- | --- |
| `internal/calculator` | Table-driven: every operation, negatives, zero, overflow, NaN and Inf input, ÷ 0, domain errors | 100% |
| `internal/httpapi` | `httptest`: every error code, 405, 404, CORS, panic recovery, SPA serving, and an OpenAPI contract test | 100% |
| `cmd/server` | Config, timeouts, real listener, graceful shutdown | 93.3% backend total (only `main()` is uncovered) |
| Frontend | Vitest + Testing Library + MSW: client (real `fetch` against MSW), validation, formatting, hook state machine, full UI flows | 100% statements and lines, 96% branches |

A snapshot of the reports is in [`docs/COVERAGE.md`](docs/COVERAGE.md). CI enforces at least 85% for Go and the thresholds in `frontend/vite.config.ts`.

## Project layout

```text
api/openapi.yaml                 API contract (source of truth)
backend/
  cmd/server/                    entrypoint: config, timeouts, graceful shutdown
  internal/calculator/           pure arithmetic core + registry
  internal/httpapi/              routes, handlers, error mapping, middleware, SPA serving
frontend/src/
  api/                           typed client + generated schema types
  hooks/                         useCalculator (state machine), useOperations
  components/                    CalculatorForm, ResultPanel, History
  validation.ts, format.ts       pure helpers
docs/                            TPD, coverage snapshot
Dockerfile, Makefile, .github/workflows/ci.yml
PROMPTS.md                       AI prompts used to build this
```
