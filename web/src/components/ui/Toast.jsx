import React, { useEffect } from 'react';

export function Toast({
  message,
  type = 'info',
  onClose,
  duration = 4000,
}) {
  useEffect(() => {
    if (!message) return;
    const timer = setTimeout(() => {
      if (onClose) onClose();
    }, duration);
    return () => clearTimeout(timer);
  }, [message, duration, onClose]);

  if (!message) return null;

  let typeClass = 'toast-info';
  if (type === 'success') typeClass = 'toast-success';
  if (type === 'error') typeClass = 'toast-error';
  if (type === 'warning') typeClass = 'toast-warning';

  return (
    <div className="toast-container">
      <div role={type === 'error' ? 'alert' : 'status'} aria-live={type === 'error' ? 'assertive' : 'polite'} className={`toast ${typeClass}`}>
        <div className="toast-content">
          <span>{message}</span>
          {onClose && (
            <button
              type="button"
              onClick={onClose}
              aria-label="Bildirimi kapat"
              className="toast-close-btn"
            >
              ✕
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

