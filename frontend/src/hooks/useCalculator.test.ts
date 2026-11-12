import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ApiError, type RequestOptions } from '../api/client';
import type { CalculateRequest, CalculateResponse } from '../api/types';
import { deferred, fakeClient } from '../test/fakeClient';
import { HISTORY_LIMIT, initialState, reducer, useCalculator } from './useCalculator';

describe('useCalculator', () => {
  it('moves idle → submitting → success and records history', async () => {
    const pending = deferred<CalculateResponse>();
    const client = fakeClient({ calculate: vi.fn(() => pending.promise) });
    const { result } = renderHook(() => useCalculator(client));
    expect(result.current.status).toBe('idle');

    act(() => void result.current.calculate('add', [2, 3]));
    expect(result.current.status).toBe('submitting');

    await act(async () => pending.resolve({ operation: 'add', operands: [2, 3], result: 5 }));
    expect(result.current.status).toBe('success');
    expect(result.current.result?.result).toBe(5);
    expect(result.current.history).toHaveLength(1);
    expect(result.current.history[0]).toMatchObject({ id: 1, result: 5 });
  });

  it('stores ApiError on failure and keeps history', async () => {
    const err = new ApiError('DIVISION_BY_ZERO', 'cannot divide by zero', 422);
    const client = fakeClient({ calculate: vi.fn().mockRejectedValue(err) });
    const { result } = renderHook(() => useCalculator(client));

    await act(() => result.current.calculate('divide', [1, 0]));

    expect(result.current.status).toBe('error');
    expect(result.current.error).toBe(err);
    expect(result.current.result).toBeNull();
  });

  it('wraps unexpected errors in a generic ApiError', async () => {
    const client = fakeClient({ calculate: vi.fn().mockRejectedValue(new TypeError('boom')) });
    const { result } = renderHook(() => useCalculator(client));

    await act(() => result.current.calculate('add', [1, 2]));

    expect(result.current.error).toBeInstanceOf(ApiError);
    expect(result.current.error?.code).toBe('BAD_RESPONSE');
  });

  it('aborts the in-flight request when a new one starts, and ignores its result', async () => {
    const first = deferred<CalculateResponse>();
    const signals: AbortSignal[] = [];
    const calculate = vi
      .fn()
      .mockImplementationOnce((_req: CalculateRequest, opts?: RequestOptions) => {
        if (opts?.signal) signals.push(opts.signal);
        return first.promise;
      })
      .mockImplementationOnce(async (_req: CalculateRequest, opts?: RequestOptions) => {
        if (opts?.signal) signals.push(opts.signal);
        return { operation: 'add', operands: [1, 1], result: 2 };
      });
    const { result } = renderHook(() => useCalculator(fakeClient({ calculate })));

    act(() => void result.current.calculate('add', [5, 5]));
    await act(() => result.current.calculate('add', [1, 1]));
    expect(signals[0]?.aborted).toBe(true);

    // The stale response arrives late and must not overwrite the newer result.
    await act(async () => first.resolve({ operation: 'add', operands: [5, 5], result: 10 }));
    expect(result.current.result?.result).toBe(2);
    expect(result.current.history).toHaveLength(1);
  });

  it('ignores AbortError from a cancelled request', async () => {
    const calculate = vi.fn().mockRejectedValue(new DOMException('aborted', 'AbortError'));
    const { result } = renderHook(() => useCalculator(fakeClient({ calculate })));

    await act(() => result.current.calculate('add', [1, 2]));

    expect(result.current.status).toBe('submitting');
    expect(result.current.error).toBeNull();
  });

  it('aborts on unmount', async () => {
    let signal: AbortSignal | undefined;
    const calculate = vi.fn((_req: CalculateRequest, opts?: RequestOptions) => {
      signal = opts?.signal;
      return new Promise<CalculateResponse>(() => {});
    });
    const { result, unmount } = renderHook(() => useCalculator(fakeClient({ calculate })));
    act(() => void result.current.calculate('add', [1, 2]));
    unmount();
    await waitFor(() => expect(signal?.aborted).toBe(true));
  });

  it('reset returns to idle without clearing history', async () => {
    const { result } = renderHook(() => useCalculator(fakeClient()));
    await act(() => result.current.calculate('add', [1, 2]));
    act(() => result.current.reset());
    expect(result.current.status).toBe('idle');
    expect(result.current.history).toHaveLength(1);
  });
});

describe('reducer', () => {
  it(`caps history at ${HISTORY_LIMIT} entries, newest first`, () => {
    let state = initialState;
    for (let i = 0; i < HISTORY_LIMIT + 3; i++) {
      state = reducer(state, { type: 'success', result: { operation: 'add', operands: [i, 0], result: i } });
    }
    expect(state.history).toHaveLength(HISTORY_LIMIT);
    expect(state.history[0]?.result).toBe(HISTORY_LIMIT + 2);
  });
});
