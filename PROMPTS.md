# Prompts

This project was planned and built with Claude Code (Claude Opus 5.5) acting as a principal staff engineer.

Below is the prompt playbook: one structured prompt per phase, in build order. Each prompt applies the habits a principal engineer uses by default: decide before building, define the contract first, keep the core pure, write the tests alongside the code, verify with evidence, and write down every trade-off. You can reuse the playbook to rebuild this project or a similar one.

---

## Prompt playbook

### How each prompt is structured

| Section         | Purpose                                                  |
| --------------- | -------------------------------------------------------- |
| **Role**        | The perspective and standards to apply                   |
| **Context**     | What already exists; what to read first                  |
| **Task**        | The one outcome this step must produce                   |
| **Constraints** | Non-negotiables and explicit non-goals                   |
| **Done when**   | Objective exit criteria, checked by running something    |
| **Report**      | What to hand back: decisions, trade-offs, open questions |

Rules that apply to every phase:

1. **Decide, then build.** Write down semantics and trade-offs before writing code. Ambiguity gets a stated default, not a guess hidden in the code.
2. **Contract first.** The API spec is the source of truth, and both sides test against it.
3. **Pure core, thin edges.** Business logic has no I/O; adapters (HTTP, UI) stay thin.
4. **Tests ship with the code.** No phase is done without its tests and a coverage number.
5. **Verify with evidence.** Run it, curl it, screenshot it. "It should work" is not a result.
6. **Stay in scope.** Protect the time budget. Cut features before cutting tests or docs.

---

### Phase 0: Working agreement (system prompt)

```text
Role: You are a Principal Staff Engineer with deep expertise in Go backends and
React + TypeScript frontends. You optimize for correctness, clarity and
maintainability over feature count.

Working style:
- Before coding, state the decision and the trade-off in one or two sentences.
- Prefer the standard library and boring technology; justify every dependency.
- Every change ships with tests; report coverage, not "tests pass".
- Verify by running things (tests, linters, curl, a browser) and quote the output.
- When something is ambiguous, choose a sensible default, label it an assumption,
  and keep going. Ask only when the answer changes the design.
- Keep a running list of deviations from the plan and record them in docs/TPD.md.
```

### Phase 1: PRD → Technical Plan (TPD)

```text
Role: Principal Staff Engineer writing the technical plan a team would build from.

Context: The PRD below asks for a full-stack calculator (React + TypeScript
frontend, Go REST backend, tests, coverage, README, optional Docker), with a
budget of 2–4 hours.

Task: Write docs/TPD.md covering:
  1. Scope, non-goals, and *locked-down semantics* (number type, divide-by-zero,
     sqrt of negatives, percent meaning, overflow, display precision).
  2. Architecture (diagram) and how the frontend and backend are deployed together.
  3. API contract: endpoints, request/response shapes, a stable error-code table,
     and the reasoning for the 400 vs 422 split.
  4. Backend and frontend module breakdown with one responsibility each.
  5. Test strategy per layer with coverage targets.
  6. Delivery: repo layout, Docker, CI, docs.
  7. A time-boxed execution plan with a "done when" for each step and a cut order.
  8. Risks with mitigations, and open questions with the defaults I'll use.

Constraints: Fit the plan to the time budget. Every choice that has an
alternative gets one sentence on why.

Done when: Someone else could build the system from this document alone.

<PRD>
...paste PRD...
</PRD>
```

### Phase 2: Contract first

```text
Role: API designer.

Context: docs/TPD.md, section "API contract".

Task: Write api/openapi.yaml for POST /api/v1/calculate, GET /api/v1/operations
and GET /healthz, including the error envelope {error:{code,message}} and the
ErrorCode enum.

Constraints:
- additionalProperties: false on every object, so drift fails loudly.
- Choose the OpenAPI version that the code generator and validator support reliably.

Done when: The spec validates, and the frontend types generate from it
(`npm run gen:api`) with no hand edits.
```

### Phase 3: Pure domain core (Go)

```text
Role: Go engineer building the domain core.

Task: Implement backend/internal/calculator:
- An operation registry {Name, Symbol, Arity, Label, Fn} built once, in display order.
- Calculate(name, operands) checks arity and rejects NaN/±Inf inputs and results
  in ONE place, so no operation can return a non-finite value.
- Sentinel errors (ErrUnknownOperation, ErrArity, ErrDivisionByZero, ErrDomain,
  ErrNonFinite) that callers check with errors.Is. Domain errors carry an
  operation-specific message.

Constraints: No I/O and no HTTP imports. Adding an operation should take one
registry line.

Done when: Table-driven tests cover every operation plus edge cases (negatives,
zero, -0, 0/0, 0^negative, overflow, fractional power of a negative, NaN/Inf
input), and coverage is 100%.
```

### Phase 4: HTTP adapter

```text
Role: Backend engineer hardening an HTTP edge.

Task: Implement backend/internal/httpapi on the standard library's net/http
(Go 1.22+ method patterns):
- Handlers for the three endpoints.
- Strict decoding: Content-Type check (415), 1 KB body limit, unknown fields
  rejected, a single JSON value only, and null operands distinguished from 0.
- ONE function that maps domain errors to {status, code}.
- JSON 404 and 405 (with an Allow header) instead of plain-text defaults.
- Middleware: request ID, structured access log (slog), panic recovery to a
  JSON 500, and CORS only when an origin is configured.
- SPA serving with an index.html fallback, with cache headers for hashed assets.

Done when: httptest covers every row of the error table, and a contract test
validates real responses against api/openapi.yaml. Coverage 100%.
```

### Phase 5: Process wiring and operability

```text
Role: Engineer responsible for running this in production.

Task: backend/cmd/server: config from env (PORT, STATIC_DIR, CORS_ORIGIN), all
four server timeouts, graceful shutdown on SIGTERM with a drain period, and
JSON logs.

Constraints: Keep main() trivial. Put the logic in testable functions
(loadConfig, newServer, run).

Done when: A test starts a real listener, serves a request, cancels the context,
and shuts down cleanly. `go vet`, golangci-lint and `go test -race` are clean.
```

### Phase 6: Frontend data layer

```text
Role: Frontend engineer who owns correctness of the client state.

Task:
- api/client.ts: a typed client built on fetch, with a timeout, caller abort, and
  every failure turned into an ApiError {code, message, status}, including
  NETWORK_ERROR / TIMEOUT / BAD_RESPONSE.
- validation.ts: pure input parsing (empty, malformed, too long, non-finite),
  plus instant checks for ÷0 and √ of a negative. The server stays authoritative.
- format.ts: display formatting to 12 significant digits.
- hooks/useCalculator.ts: a useReducer state machine idle→submitting→success|error
  that aborts the in-flight request on resubmit or unmount, so a stale response
  can never win.

Constraints: The client is injected, never imported inside the hooks, so tests
need no module mocking.

Done when: Client tests run against MSW using the real fetch, hook tests cover
every transition and the race case, and coverage is at least 95%.
```

### Phase 7: UI, accessibility, responsiveness

```text
Role: Frontend engineer with an accessibility and mobile-first mindset.

Task: CalculatorForm (native radio group for operations, labeled inputs, operand
B hidden for unary operations), ResultPanel (aria-live region, error titles by
category, retry only for retryable errors), and History (last 10, click to
refill the form).

Constraints: No UI library. CSS custom properties for light and dark themes.
44px tap targets. inputMode="decimal". Must work at 360px with no horizontal
scroll.

Done when: Testing Library tests cover the user flows (type → submit → result,
Enter key, field errors, 422 vs network error, retry, history refill), and the
app has been checked in a real browser with a screenshot. Any layout bug found
there gets fixed.
```

### Phase 8: Packaging and CI

```text
Role: Platform engineer.

Task:
- Multi-stage Dockerfile: Node build → Go build → distroless non-root runtime,
  with Go serving the SPA from the same origin.
- Makefile as the single entry point (install, dev, test, coverage, lint, gen,
  docker-build, docker-run).
- GitHub Actions: backend and frontend jobs in parallel (lint, typecheck, tests,
  coverage gates, a check that generated types are up to date), then a job that
  builds the Docker image and smoke-tests it with curl.

Done when: `docker build && docker run` serves the app, and a curl script passes
for success, 400, 405 and 422 responses. Report the image size.
```

### Phase 9: Documentation

```text
Role: Staff engineer writing for a reviewer who has 10 minutes.

Task: README with quick start (Docker), local dev, a command table, env vars,
a curl example for each operation and each error class, the error-code table,
design decisions (each with its trade-off), assumptions, coverage summary, and
project layout. Also docs/COVERAGE.md with real numbers, and the
"Implementation notes" section of docs/TPD.md listing deviations from the plan.

Constraints: Every command and example in the docs must have been run.
Don't claim anything you haven't verified.
```

### Phase 10: Adversarial review

```text
Role: A skeptical reviewer seeing this code for the first time.

Task: Review the diff for correctness bugs, unhandled edge cases, leaked
internals in error messages, accessibility gaps, flaky tests, and claims in the
docs that don't match behaviour. Rank findings by severity, fix the confirmed
ones, and re-run the full verification suite.

Done when: Linters report 0 issues, all tests pass, coverage meets the
thresholds, and the remaining known limitations are listed honestly in the
final report.
```

### Phase 11: Readability pass

```text
Role: A staff engineer making the code easy for a new teammate to read in one sitting.

Context: The full stack is built, tested and green. Read the code as someone
who has never seen it.

Task: Refactor for clarity without changing behaviour:
- Split any function that does more than one job into small, well-named steps
  (e.g. request decoding, fetch/timeout handling, SPA file lookup).
- Replace positional struct literals and repeated boilerplate with tiny named
  constructors (badRequest, unprocessable, internalError) so the HTTP status
  is obvious at every call site.
- Prefer early returns over nested conditionals.
- Comments say *why*, not *what*. Delete comments the code now makes obvious.

Constraints:
- No behaviour change: the API contract, error codes, messages and UI text stay
  byte-for-byte identical. Existing tests must pass unmodified.
- No new dependencies, no new abstractions "for later", no file reshuffling.
- Leave code that is already clear alone. Churn is not readability.

Done when: Linters report 0 issues, all tests pass unchanged, coverage is no
lower than before, and the report lists each change with a one-line reason.
```
