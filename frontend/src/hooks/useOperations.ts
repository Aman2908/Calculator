import { useEffect, useState } from 'react';
import type { CalculatorClient } from '../api/client';
import type { Operation } from '../api/types';
import { FALLBACK_OPERATIONS } from '../operations';

/**
 * Loads the operation list from the server. Starts with, and falls back to,
 * the static list, so the calculator works even if this request fails.
 */
export function useOperations(client: CalculatorClient): readonly Operation[] {
  const [operations, setOperations] = useState<readonly Operation[]>(FALLBACK_OPERATIONS);

  useEffect(() => {
    const controller = new AbortController();
    client
      .listOperations({ signal: controller.signal })
      .then((ops) => {
        if (ops.length > 0) setOperations(ops);
      })
      .catch(() => {
        // Keep the fallback list; the calculate call will surface real errors.
      });
    return () => controller.abort();
  }, [client]);

  return operations;
}
