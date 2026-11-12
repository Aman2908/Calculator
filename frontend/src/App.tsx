import { useRef, useState } from 'react';
import { api, type CalculatorClient } from './api/client';
import type { OperationName } from './api/types';
import { CalculatorForm, type FormValues } from './components/CalculatorForm';
import { History } from './components/History';
import { ResultPanel } from './components/ResultPanel';
import { useCalculator, type HistoryEntry } from './hooks/useCalculator';
import { useOperations } from './hooks/useOperations';
import { findOperation } from './operations';
import { validateInputs, type FieldErrors } from './validation';

interface Props {
  /** Injected in tests; defaults to the real HTTP client. */
  client?: CalculatorClient;
}

export default function App({ client = api }: Props) {
  const operations = useOperations(client);
  const calc = useCalculator(client);
  const [values, setValues] = useState<FormValues>({ operation: 'add', a: '', b: '' });
  const [errors, setErrors] = useState<FieldErrors>({});
  const lastRequest = useRef<{ operation: OperationName; operands: number[] } | null>(null);

  function handleChange(patch: Partial<FormValues>) {
    setValues((v) => ({ ...v, ...patch }));
    // Editing a field clears its error; switching operation clears both.
    const operationChanged = 'operation' in patch;
    setErrors((e) => ({
      a: operationChanged || 'a' in patch ? undefined : e.a,
      b: operationChanged || 'b' in patch ? undefined : e.b,
    }));
  }

  function handleSubmit() {
    const op = findOperation(operations, values.operation);
    const validation = validateInputs(op, values.a, values.b);
    if (!validation.ok) {
      setErrors(validation.errors);
      return;
    }
    setErrors({});
    lastRequest.current = { operation: op.name, operands: validation.operands };
    void calc.calculate(op.name, validation.operands);
  }

  function handleRetry() {
    if (lastRequest.current) {
      void calc.calculate(lastRequest.current.operation, lastRequest.current.operands);
    }
  }

  function handleHistorySelect(entry: HistoryEntry) {
    setErrors({});
    setValues({
      operation: entry.operation,
      a: String(entry.operands[0] ?? ''),
      b: String(entry.operands[1] ?? ''),
    });
  }

  return (
    <main className="app">
      <div className="card">
        <h1>Calculator</h1>
        <ResultPanel status={calc.status} result={calc.result} error={calc.error} onRetry={handleRetry} />
        <CalculatorForm
          operations={operations}
          values={values}
          errors={errors}
          submitting={calc.status === 'submitting'}
          onChange={handleChange}
          onSubmit={handleSubmit}
        />
        <History entries={calc.history} onSelect={handleHistorySelect} />
      </div>
    </main>
  );
}
