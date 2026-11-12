import { formatExpression, formatNumber } from '../format';
import type { HistoryEntry } from '../hooks/useCalculator';

interface Props {
  entries: readonly HistoryEntry[];
  onSelect: (entry: HistoryEntry) => void;
}

export function History({ entries, onSelect }: Props) {
  if (entries.length === 0) return null;
  return (
    <section className="history" aria-labelledby="history-title">
      <h2 id="history-title">History</h2>
      <ol>
        {entries.map((e) => (
          <li key={e.id}>
            <button
              type="button"
              onClick={() => onSelect(e)}
              aria-label={`Reuse ${formatExpression(e.operation, e.operands)}`}
            >
              <span>{formatExpression(e.operation, e.operands)}</span>
              <span className="history-result">= {formatNumber(e.result)}</span>
            </button>
          </li>
        ))}
      </ol>
    </section>
  );
}
