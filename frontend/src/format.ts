import type { OperationName } from './api/types';
import { FALLBACK_OPERATIONS } from './operations';

const SIGNIFICANT_DIGITS = 12;

/**
 * Formats a float64 for display: at most 12 significant digits, so
 * 0.1 + 0.2 shows as 0.3 rather than 0.30000000000000004.
 */
export function formatNumber(n: number): string {
  return String(Number(n.toPrecision(SIGNIFICANT_DIGITS)));
}

function symbolFor(op: OperationName): string {
  return FALLBACK_OPERATIONS.find((o) => o.name === op)?.symbol ?? op;
}

/** Human-readable expression, e.g. "12 ÷ 4", "√16", "15% of 200". */
export function formatExpression(op: OperationName, operands: readonly number[]): string {
  const [a = 0, b = 0] = operands;
  switch (op) {
    case 'sqrt':
      return `√${formatNumber(a)}`;
    case 'percent':
      return `${formatNumber(b)}% of ${formatNumber(a)}`;
    default:
      return `${formatNumber(a)} ${symbolFor(op)} ${formatNumber(b)}`;
  }
}
