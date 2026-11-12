# Coverage snapshot

Generated on 2026-11-10 with `make coverage` (Go 1.26, Node 24). To regenerate, run `make coverage` and open `coverage/backend/index.html` and `coverage/frontend/index.html`.

## Backend (Go): 93.3% of statements

```text
$ go test -race -cover ./...
ok   calculator/cmd/server            coverage: 54.8% of statements
ok   calculator/internal/calculator   coverage: 100.0% of statements
ok   calculator/internal/httpapi      coverage: 100.0% of statements
total:                                (statements)  93.3%
```

The only uncovered code is `main()` (process wiring and signal handling) and `run`'s error branch when `Shutdown` itself fails. `run` is otherwise exercised against a real listener, including graceful shutdown.

## Frontend (Vitest, v8): 100% statements, 96.3% branches

```text
File               | % Stmts | % Branch | % Funcs | % Lines
-------------------|---------|----------|---------|--------
All files          |     100 |    96.26 |     100 |     100
 App.tsx           |     100 |    82.35 |     100 |     100
 format.ts         |     100 |    85.71 |     100 |     100
 operations.ts     |     100 |      100 |     100 |     100
 validation.ts     |     100 |      100 |     100 |     100
 api/client.ts     |     100 |      100 |     100 |     100
 CalculatorForm    |     100 |      100 |     100 |     100
 History.tsx       |     100 |      100 |     100 |     100
 ResultPanel.tsx   |     100 |      100 |     100 |     100
 useCalculator.ts  |     100 |      100 |     100 |     100
 useOperations.ts  |     100 |       50 |     100 |     100
```

`src/main.tsx` (React bootstrap), the generated `schema.d.ts`, and test files are excluded. 5 test files, 77 tests.
