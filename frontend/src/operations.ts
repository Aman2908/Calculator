import type { Operation } from './api/types';

const DEFAULT_OPERATION: Operation = { name: 'add', symbol: '+', arity: 2, label: 'Add' };

/**
 * Used until GET /api/v1/operations answers, and if it fails, so the UI is
 * never empty. Kept in sync with the server registry by the OperationName type.
 */
export const FALLBACK_OPERATIONS: readonly Operation[] = [
  DEFAULT_OPERATION,
  { name: 'subtract', symbol: '−', arity: 2, label: 'Subtract' },
  { name: 'multiply', symbol: '×', arity: 2, label: 'Multiply' },
  { name: 'divide', symbol: '÷', arity: 2, label: 'Divide' },
  { name: 'power', symbol: '^', arity: 2, label: 'Power' },
  { name: 'sqrt', symbol: '√', arity: 1, label: 'Square root' },
  { name: 'percent', symbol: '%', arity: 2, label: 'Percent (b% of a)' },
];

/** Finds an operation by name, never returning undefined. */
export function findOperation(operations: readonly Operation[], name: string): Operation {
  return operations.find((o) => o.name === name) ?? operations[0] ?? DEFAULT_OPERATION;
}
