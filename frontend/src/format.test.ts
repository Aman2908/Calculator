import { describe, expect, it } from 'vitest';
import { formatExpression, formatNumber } from './format';

describe('formatNumber', () => {
  it.each([
    [0.1 + 0.2, '0.3'],
    [2.5, '2.5'],
    [42, '42'],
    [-0, '0'],
    [1 / 3, '0.333333333333'],
    [1e21, '1e+21'],
    [123456789012345, '123456789012000'],
    [1e-7, '1e-7'],
  ])('formats %d as %s', (n, s) => {
    expect(formatNumber(n)).toBe(s);
  });
});

describe('formatExpression', () => {
  it.each([
    ['add', [2, 3], '2 + 3'],
    ['subtract', [2, 3], '2 − 3'],
    ['multiply', [2, 3], '2 × 3'],
    ['divide', [12, 4], '12 ÷ 4'],
    ['power', [2, 8], '2 ^ 8'],
    ['sqrt', [16], '√16'],
    ['percent', [200, 15], '15% of 200'],
  ] as const)('%s %j → %s', (op, operands, text) => {
    expect(formatExpression(op, operands)).toBe(text);
  });
});
