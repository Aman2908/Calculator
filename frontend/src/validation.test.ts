import { describe, expect, it } from 'vitest';
import type { Operation } from './api/types';
import { FALLBACK_OPERATIONS, findOperation } from './operations';
import { MAX_INPUT_LENGTH, parseOperand, validateInputs } from './validation';

const op = (name: Operation['name']) => findOperation(FALLBACK_OPERATIONS, name);

describe('parseOperand', () => {
  it.each([
    ['42', 42],
    ['-3.5', -3.5],
    ['+7', 7],
    ['.5', 0.5],
    ['5.', 5],
    ['1e6', 1e6],
    ['2.5E-3', 0.0025],
    ['  12  ', 12],
    ['0', 0],
  ])('accepts %j', (raw, value) => {
    expect(parseOperand(raw)).toEqual({ ok: true, value });
  });

  it.each([
    ['', 'Enter a number.'],
    ['   ', 'Enter a number.'],
    ['abc', 'valid number'],
    ['1,000', 'valid number'],
    ['0x10', 'valid number'],
    ['Infinity', 'valid number'],
    ['NaN', 'valid number'],
    ['1.2.3', 'valid number'],
    ['-', 'valid number'],
    ['1e400', 'too large'],
    ['1'.repeat(MAX_INPUT_LENGTH + 1), 'at most'],
  ])('rejects %j', (raw, message) => {
    const r = parseOperand(raw);
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error).toContain(message);
  });
});

describe('validateInputs', () => {
  it('returns both operands for a binary operation', () => {
    expect(validateInputs(op('add'), '2', '3')).toEqual({ ok: true, operands: [2, 3] });
  });

  it('ignores operand B for a unary operation', () => {
    expect(validateInputs(op('sqrt'), '16', 'garbage')).toEqual({ ok: true, operands: [16] });
  });

  it('reports errors for both fields at once', () => {
    expect(validateInputs(op('multiply'), '', 'x')).toEqual({
      ok: false,
      errors: { a: 'Enter a number.', b: expect.stringContaining('valid number') },
    });
  });

  it('catches division by zero before calling the server', () => {
    expect(validateInputs(op('divide'), '5', '0')).toEqual({
      ok: false,
      errors: { b: 'Cannot divide by zero.' },
    });
  });

  it('catches the square root of a negative number', () => {
    expect(validateInputs(op('sqrt'), '-4', '')).toEqual({
      ok: false,
      errors: { a: expect.stringContaining('0 or more') },
    });
  });

  it('allows zero as a numerator and in other operations', () => {
    expect(validateInputs(op('divide'), '0', '5')).toEqual({ ok: true, operands: [0, 5] });
    expect(validateInputs(op('multiply'), '5', '0')).toEqual({ ok: true, operands: [5, 0] });
  });
});
