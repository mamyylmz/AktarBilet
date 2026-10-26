import React from 'react';

export function Table({
  columns = [],
  data = [],
  keyExtractor,
  emptyMessage = 'Kayıt bulunamadı.',
  ariaLabel = 'Tablo verileri',
  className = '',
}) {
  return (
    <div
      tabIndex={0}
      role="region"
      aria-label={ariaLabel}
      className={`table-scroll-container focus-visible:outline-2 ${className}`}
    >
      <table className="data-table">
        <thead>
          <tr>
            {columns.map((col, idx) => (
              <th
                key={col.key || idx}
                style={{ textAlign: col.align || 'left', width: col.width || 'auto' }}
                scope="col"
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="table-empty-cell">
                {emptyMessage}
              </td>
            </tr>
          ) : (
            data.map((row, rowIdx) => (
              <tr key={keyExtractor ? keyExtractor(row, rowIdx) : row.id || rowIdx}>
                {columns.map((col, colIdx) => (
                  <td
                    key={col.key || colIdx}
                    style={{ textAlign: col.align || 'left' }}
                  >
                    {col.render ? col.render(row, rowIdx) : row[col.key]}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}

