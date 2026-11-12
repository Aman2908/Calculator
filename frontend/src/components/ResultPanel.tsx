import type { ApiError } from '../api/client';
import type { CalculateResponse } from '../api/types';
import { formatExpression, formatNumber } from '../format';
import type { CalcStatus } from '../hooks/useCalculator';

interface Props {
  status: CalcStatus;
  result: CalculateResponse | null;
  error: ApiError | null;
  onRetry: () => void;
}

const RETRYABLE = new Set(['NETWORK_ERROR', 'TIMEOUT', 'BAD_RESPONSE', 'INTERNAL']);

function errorTitle(error: ApiError): string {
  if (error.status === 422) return "Can't calculate that";
  if (error.status === 400 || error.status === 415) return 'Invalid input';
  if (error.code === 'NETWORK_ERROR' || error.code === 'TIMEOUT') return 'Connection problem';
  return 'Something went wrong';
}

export function ResultPanel({ status, result, error, onRetry }: Props) {
  return (
    <section className={`display display-${status}`} aria-live="polite" aria-label="Result">
      {status === 'idle' && <p className="display-hint">Enter numbers and choose an operation.</p>}

      {status === 'submitting' && <p className="display-hint">Calculating…</p>}

      {status === 'success' && result && (
        <>
          <p className="display-expr">{formatExpression(result.operation, result.operands)} =</p>
          <output className="display-result" data-testid="result">
            {formatNumber(result.result)}
          </output>
        </>
      )}

      {status === 'error' && error && (
        <div role="alert">
          <p className="display-error-title">{errorTitle(error)}</p>
          <p className="display-error-msg">{error.message}</p>
          {RETRYABLE.has(error.code) && (
            <button type="button" className="retry" onClick={onRetry}>
              Try again
            </button>
          )}
        </div>
      )}
    </section>
  );
}
