import type { FormEvent } from 'react';
import type { Operation, OperationName } from '../api/types';
import { findOperation } from '../operations';
import type { FieldErrors } from '../validation';

export interface FormValues {
  operation: OperationName;
  a: string;
  b: string;
}

interface Props {
  operations: readonly Operation[];
  values: FormValues;
  errors: FieldErrors;
  submitting: boolean;
  onChange: (patch: Partial<FormValues>) => void;
  onSubmit: () => void;
}

function operandLabels(op: OperationName): [string, string] {
  switch (op) {
    case 'sqrt':
      return ['Number', ''];
    case 'power':
      return ['Base', 'Exponent'];
    case 'percent':
      return ['Amount', 'Percent'];
    default:
      return ['First number', 'Second number'];
  }
}

export function CalculatorForm({ operations, values, errors, submitting, onChange, onSubmit }: Props) {
  const selected = findOperation(operations, values.operation);
  const [labelA, labelB] = operandLabels(selected.name);
  const binary = selected.arity === 2;

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit();
  }

  return (
    <form className="calc-form" onSubmit={handleSubmit} noValidate>
      <fieldset className="ops">
        <legend>Operation</legend>
        <div className="ops-grid">
          {operations.map((op) => (
            <label key={op.name} className="op" title={op.label}>
              <input
                type="radio"
                name="operation"
                value={op.name}
                checked={op.name === selected.name}
                onChange={() => onChange({ operation: op.name })}
              />
              <span aria-hidden="true">{op.symbol}</span>
              <span className="visually-hidden">{op.label}</span>
            </label>
          ))}
        </div>
        <p className="op-label" aria-hidden="true">
          {selected.label}
        </p>
      </fieldset>

      <div className={binary ? 'operands' : 'operands single'}>
        <NumberField
          id="operand-a"
          label={labelA}
          value={values.a}
          error={errors.a}
          onChange={(a) => onChange({ a })}
        />
        {binary && (
          <NumberField
            id="operand-b"
            label={labelB}
            value={values.b}
            error={errors.b}
            onChange={(b) => onChange({ b })}
          />
        )}
      </div>

      <button type="submit" className="submit" disabled={submitting}>
        {submitting ? 'Calculating…' : 'Calculate'}
      </button>
    </form>
  );
}

interface FieldProps {
  id: string;
  label: string;
  value: string;
  error?: string;
  onChange: (value: string) => void;
}

function NumberField({ id, label, value, error, onChange }: FieldProps) {
  const errorId = `${id}-error`;
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        type="text"
        inputMode="decimal"
        autoComplete="off"
        spellCheck={false}
        value={value}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorId : undefined}
        onChange={(e) => onChange(e.target.value)}
      />
      {error && (
        <p id={errorId} className="field-error">
          {error}
        </p>
      )}
    </div>
  );
}
