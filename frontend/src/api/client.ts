import type { CalculateRequest, CalculateResponse, ErrorCode, Operation, OperationsResponse } from './types';

export const DEFAULT_TIMEOUT_MS = 5000;

/** Every failure from the client surfaces as an ApiError (except caller aborts). */
export class ApiError extends Error {
  readonly code: ErrorCode;
  /** HTTP status, or 0 when no response was received. */
  readonly status: number;

  constructor(code: ErrorCode, message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

export function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === 'AbortError';
}

export interface RequestOptions {
  signal?: AbortSignal;
  timeoutMs?: number;
}

export interface CalculatorClient {
  calculate(req: CalculateRequest, opts?: RequestOptions): Promise<CalculateResponse>;
  listOperations(opts?: RequestOptions): Promise<Operation[]>;
}

export function createClient(baseUrl = ''): CalculatorClient {
  async function request<T>(path: string, init: RequestInit, opts: RequestOptions = {}): Promise<T> {
    const { signal: callerSignal, timeoutMs = DEFAULT_TIMEOUT_MS } = opts;
    const deadline = withDeadline(callerSignal, timeoutMs);
    try {
      let res: Response;
      try {
        res = await fetch(baseUrl + path, { ...init, signal: deadline.signal });
      } catch {
        throw transportError(deadline.timedOut(), callerSignal);
      }
      return await readJson<T>(res);
    } finally {
      deadline.dispose();
    }
  }

  return {
    calculate: (req, opts) =>
      request<CalculateResponse>(
        '/api/v1/calculate',
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        },
        opts,
      ),
    listOperations: async (opts) =>
      (await request<OperationsResponse>('/api/v1/operations', { method: 'GET' }, opts)).operations,
  };
}

/**
 * Combines the caller's signal with a timeout into one signal for fetch, and
 * remembers which one fired so a timeout can be told apart from a caller abort.
 */
function withDeadline(callerSignal: AbortSignal | undefined, timeoutMs: number) {
  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);
  const forwardAbort = () => controller.abort();
  if (callerSignal?.aborted) controller.abort();
  callerSignal?.addEventListener('abort', forwardAbort, { once: true });

  return {
    signal: controller.signal,
    timedOut: () => timedOut,
    dispose() {
      clearTimeout(timer);
      callerSignal?.removeEventListener('abort', forwardAbort);
    },
  };
}

/** Explains why fetch rejected: timeout, caller abort, or no network. */
function transportError(timedOut: boolean, callerSignal: AbortSignal | undefined): Error {
  if (timedOut) return new ApiError('TIMEOUT', 'The server took too long to respond.', 0);
  if (callerSignal?.aborted) return new DOMException('Request aborted', 'AbortError');
  return new ApiError('NETWORK_ERROR', 'Could not reach the server. Check your connection.', 0);
}

/** Parses the JSON body. A non-2xx status becomes an ApiError from the error envelope. */
async function readJson<T>(res: Response): Promise<T> {
  let body: unknown;
  try {
    body = await res.json();
  } catch {
    throw new ApiError('BAD_RESPONSE', `Unexpected response from server (${res.status}).`, res.status);
  }

  if (!res.ok) {
    const err = (body as { error?: { code?: ErrorCode; message?: string } } | null)?.error;
    throw new ApiError(
      err?.code ?? 'BAD_RESPONSE',
      err?.message ?? `Request failed (${res.status}).`,
      res.status,
    );
  }
  return body as T;
}

/** Default client: same origin in production, proxied by Vite in development. */
export const api = createClient(import.meta.env.VITE_API_BASE_URL ?? '');
