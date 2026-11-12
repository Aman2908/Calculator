import type { Operation } from './api/types';

export const MAX_INPUT_LENGTH = 32;

// Plain decimal or scientific notation: 42, -3.5, .5, 1e6, 2.5E-3.
const NUMBER_PATTERN = /^[+-]?(\d+\.?\d*|\.\d+)(e[+-]?\d+)?$/i;

export type ParseResult = { ok: true; value: number } | { ok: false; error: string };

/** Parses one raw input string into a finite number or a user-facing error. */
export function parseOperand(raw: string): ParseResult {
  const s = raw.trim();
  if (s === '') return { ok: false, error: 'Enter a number.' };
  if (s.length > MAX_INPUT_LENGTH) {
    return { ok: false, error: `Use at most ${MAX_INPUT_LENGTH} characters.` };
  }
  if (!NUMBER_PATTERN.test(s)) {
    return { ok: false, error: 'Enter a valid number, like 42, -3.5 or 1e6.' };
  }
  const value = Number(s);
  if (!Number.isFinite(value)) return { ok: false, error: 'That number is too large.' };
  return { ok: true, value };
}

export interface FieldErrors {
  a?: string;
  b?: string;
}

export type ValidationResult = { ok: true; operands: number[] } | { ok: false; errors: FieldErrors };

/**
 * Validates the form for the chosen operation. Obvious mathematical errors
 * are caught here for instant feedback; the server remains authoritative.
 */
export function validateInputs(op: Operation, rawA: string, rawB: string): ValidationResult {
  const errors: FieldErrors = {};
  const a = parseOperand(rawA);
  if (!a.ok) errors.a = a.error;

  let b: ParseResult | undefined;
  if (op.arity === 2) {
    b = parseOperand(rawB);
    if (!b.ok) errors.b = b.error;
  }

  if (a.ok && op.name === 'sqrt' && a.value < 0) {
    errors.a = 'Square root needs a number of 0 or more.';
  }
  if (b?.ok && op.name === 'divide' && b.value === 0) {
    errors.b = 'Cannot divide by zero.';
  }

  if (errors.a || errors.b || !a.ok) return { ok: false, errors };
  return { ok: true, operands: b?.ok ? [a.value, b.value] : [a.value] };
}
