import React from 'react';

export function Input({
  id,
  label,
  type = 'text',
  value,
  onChange,
  placeholder,
  error,
  helpText,
  required = false,
  disabled = false,
  className = '',
  autoFocus = false,
  ...props
}) {
  const inputId = id || `input-${Math.random().toString(36).slice(2, 9)}`;
  const errorId = `${inputId}-error`;
  const helpId = `${inputId}-help`;

  let describedBy = '';
  if (error) describedBy += errorId;
  if (helpText) describedBy += (describedBy ? ' ' : '') + helpId;

  return (
    <div className={`form-group ${className}`}>
      {label && (
        <label htmlFor={inputId} className="form-label">
          {label}
          {required && <span className="text-danger ml-1" aria-hidden="true">*</span>}
        </label>
      )}
      <input
        id={inputId}
        type={type}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        required={required}
        disabled={disabled}
        autoFocus={autoFocus}
        aria-invalid={!!error}
        aria-describedby={describedBy || undefined}
        className={`form-input ${error ? 'border-danger' : ''}`}
        {...props}
      />
      {error && (
        <p id={errorId} className="form-error" role="alert">
          {error}
        </p>
      )}
      {helpText && !error && (
        <p id={helpId} className="form-help">
          {helpText}
        </p>
      )}
    </div>
  );
}

