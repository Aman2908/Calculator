import { useCallback, useEffect, useReducer, useRef } from 'react';
import { ApiError, isAbortError, type CalculatorClient } from '../api/client';
import type { CalculateResponse, OperationName } from '../api/types';

export const HISTORY_LIMIT = 10;

export interface HistoryEntry extends CalculateResponse {
  id: number;
}

export type CalcStatus = 'idle' | 'submitting' | 'success' | 'error';

export interface CalcState {
  status: CalcStatus;
  result: CalculateResponse | null;
  error: ApiError | null;
  history: HistoryEntry[];
  nextId: number;
}

type Action =
  | { type: 'submit' }
  | { type: 'success'; result: CalculateResponse }
  | { type: 'failure'; error: ApiError }
  | { type: 'reset' };

export const initialState: CalcState = {
  status: 'idle',
  result: null,
  error: null,
  history: [],
  nextId: 1,
};

export function reducer(state: CalcState, action: Action): CalcState {
  switch (action.type) {
    case 'submit':
      return { ...state, status: 'submitting', error: null };
    case 'success':
      return {
        ...state,
        status: 'success',
        result: action.result,
        error: null,
        history: [{ ...action.result, id: state.nextId }, ...state.history].slice(0, HISTORY_LIMIT),
        nextId: state.nextId + 1,
      };
    case 'failure':
      return { ...state, status: 'error', result: null, error: action.error };
    case 'reset':
      return { ...state, status: 'idle', result: null, error: null };
  }
}

/**
 * Owns the request lifecycle: idle → submitting → success | error.
 * A new calculation cancels the one in flight, and unmounting cancels too,
 * so a stale response can never overwrite a newer one.
 */
export function useCalculator(client: CalculatorClient) {
  const [state, dispatch] = useReducer(reducer, initialState);
  const inFlight = useRef<AbortController | null>(null);

  useEffect(() => () => inFlight.current?.abort(), []);

  const calculate = useCallback(
    async (operation: OperationName, operands: number[]) => {
      inFlight.current?.abort();
      const controller = new AbortController();
      inFlight.current = controller;
      dispatch({ type: 'submit' });
      try {
        const result = await client.calculate({ operation, operands }, { signal: controller.signal });
        if (!controller.signal.aborted) dispatch({ type: 'success', result });
      } catch (err) {
        if (isAbortError(err) || controller.signal.aborted) return;
        const error =
          err instanceof ApiError
            ? err
            : new ApiError('BAD_RESPONSE', 'Something went wrong. Please try again.', 0);
        dispatch({ type: 'failure', error });
      }
    },
    [client],
  );

  const reset = useCallback(() => dispatch({ type: 'reset' }), []);

  return { ...state, calculate, reset };
}
