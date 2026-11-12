import { vi } from 'vitest';
import type { CalculatorClient } from '../api/client';
import type { CalculateRequest, CalculateResponse } from '../api/types';
import { FALLBACK_OPERATIONS } from '../operations';

/** A fake client whose calculate() evaluates locally, like the real server. */
export function fakeClient(overrides: Partial<CalculatorClient> = {}): CalculatorClient {
  return {
    listOperations: vi.fn(async () => [...FALLBACK_OPERATIONS]),
    calculate: vi.fn(async ({ operation, operands }: CalculateRequest) => {
      const [a = 0, b = 0] = operands;
      const results: Record<CalculateRequest['operation'], number> = {
        add: a + b,
        subtract: a - b,
        multiply: a * b,
        divide: a / b,
        power: a ** b,
        sqrt: Math.sqrt(a),
        percent: (a * b) / 100,
      };
      return { operation, operands, result: results[operation] } satisfies CalculateResponse;
    }),
    ...overrides,
  };
}

/** A promise you can settle from the outside, for in-flight assertions. */
export function deferred<T>() {
  let resolve!: (v: T) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}
