// @vitest-environment node
// Runs in Node so MSW intercepts the real fetch implementation end to end.
import { delay, http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';
import { ApiError, createClient, isAbortError } from './client';

const BASE = 'http://api.test';
const server = setupServer();
const client = createClient(BASE);

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

async function expectApiError(p: Promise<unknown>, code: string, status: number) {
  const err = await p.then(
    () => {
      throw new Error('expected rejection');
    },
    (e: unknown) => e,
  );
  expect(err).toBeInstanceOf(ApiError);
  expect(err).toMatchObject({ code, status });
  return err as ApiError;
}

describe('calculate', () => {
  it('POSTs JSON and returns the parsed result', async () => {
    let received: unknown;
    let contentType: string | null = null;
    server.use(
      http.post(`${BASE}/api/v1/calculate`, async ({ request }) => {
        received = await request.json();
        contentType = request.headers.get('Content-Type');
        return HttpResponse.json({ operation: 'divide', operands: [10, 4], result: 2.5 });
      }),
    );

    const res = await client.calculate({ operation: 'divide', operands: [10, 4] });

    expect(res.result).toBe(2.5);
    expect(received).toEqual({ operation: 'divide', operands: [10, 4] });
    expect(contentType).toBe('application/json');
  });

  it.each([
    [422, 'DIVISION_BY_ZERO', 'cannot divide by zero'],
    [400, 'INVALID_OPERANDS', 'wrong number of operands'],
    [500, 'INTERNAL', 'internal server error'],
  ])('maps a %i %s error envelope to ApiError', async (status, code, message) => {
    server.use(
      http.post(`${BASE}/api/v1/calculate`, () =>
        HttpResponse.json({ error: { code, message } }, { status }),
      ),
    );
    const err = await expectApiError(
      client.calculate({ operation: 'divide', operands: [1, 0] }),
      code,
      status,
    );
    expect(err.message).toBe(message);
  });

  it('reports BAD_RESPONSE for a non-JSON body', async () => {
    server.use(http.post(`${BASE}/api/v1/calculate`, () => new HttpResponse('oops', { status: 502 })));
    await expectApiError(client.calculate({ operation: 'add', operands: [1, 2] }), 'BAD_RESPONSE', 502);
  });

  it('reports BAD_RESPONSE for an error without an envelope', async () => {
    server.use(http.post(`${BASE}/api/v1/calculate`, () => HttpResponse.json({}, { status: 503 })));
    const err = await expectApiError(
      client.calculate({ operation: 'add', operands: [1, 2] }),
      'BAD_RESPONSE',
      503,
    );
    expect(err.message).toContain('503');
  });

  it('reports NETWORK_ERROR when the server is unreachable', async () => {
    server.use(http.post(`${BASE}/api/v1/calculate`, () => HttpResponse.error()));
    await expectApiError(client.calculate({ operation: 'add', operands: [1, 2] }), 'NETWORK_ERROR', 0);
  });

  it('reports TIMEOUT when the server is too slow', async () => {
    server.use(
      http.post(`${BASE}/api/v1/calculate`, async () => {
        await delay(200);
        return HttpResponse.json({});
      }),
    );
    await expectApiError(
      client.calculate({ operation: 'add', operands: [1, 2] }, { timeoutMs: 20 }),
      'TIMEOUT',
      0,
    );
  });

  it('rejects with an AbortError when the caller aborts', async () => {
    server.use(
      http.post(`${BASE}/api/v1/calculate`, async () => {
        await delay(200);
        return HttpResponse.json({});
      }),
    );
    const controller = new AbortController();
    const p = client.calculate({ operation: 'add', operands: [1, 2] }, { signal: controller.signal });
    controller.abort();
    await expect(p).rejects.toSatisfy(isAbortError);
  });

  it('rejects immediately when given an already-aborted signal', async () => {
    server.use(http.post(`${BASE}/api/v1/calculate`, () => HttpResponse.json({})));
    const controller = new AbortController();
    controller.abort();
    await expect(
      client.calculate({ operation: 'add', operands: [1, 2] }, { signal: controller.signal }),
    ).rejects.toSatisfy(isAbortError);
  });
});

describe('listOperations', () => {
  it('unwraps the operations array', async () => {
    server.use(
      http.get(`${BASE}/api/v1/operations`, () =>
        HttpResponse.json({ operations: [{ name: 'add', symbol: '+', arity: 2, label: 'Add' }] }),
      ),
    );
    await expect(client.listOperations()).resolves.toEqual([
      { name: 'add', symbol: '+', arity: 2, label: 'Add' },
    ]);
  });
});

describe('createClient default base URL', () => {
  it('uses relative URLs when no base is given', async () => {
    const original = globalThis.fetch;
    let url = '';
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      url = String(input);
      return new Response(JSON.stringify({ operations: [] }));
    }) as typeof fetch;
    try {
      await createClient().listOperations();
      expect(url).toBe('/api/v1/operations');
    } finally {
      globalThis.fetch = original;
    }
  });
});

describe('isAbortError', () => {
  it('only matches DOMException AbortError', () => {
    expect(isAbortError(new DOMException('x', 'AbortError'))).toBe(true);
    expect(isAbortError(new Error('AbortError'))).toBe(false);
    expect(isAbortError(null)).toBe(false);
  });
});
