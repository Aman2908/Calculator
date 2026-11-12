import { describe, expect, it } from 'vitest';
import { FALLBACK_OPERATIONS, findOperation } from './operations';

describe('findOperation', () => {
  it('finds by name', () => {
    expect(findOperation(FALLBACK_OPERATIONS, 'sqrt').arity).toBe(1);
  });

  it('falls back to the first operation, then to add', () => {
    expect(findOperation(FALLBACK_OPERATIONS.slice(3), 'nope').name).toBe('divide');
    expect(findOperation([], 'nope').name).toBe('add');
  });
});
