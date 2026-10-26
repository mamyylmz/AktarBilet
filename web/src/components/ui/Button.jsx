import React from 'react';

export function Button({
  children,
  variant = 'primary',
  size = 'md',
  type = 'button',
  disabled = false,
  loading = false,
  className = '',
  onClick,
  ...props
}) {
  const baseClasses = 'btn inline-flex items-center justify-center font-medium transition-all focus-visible:outline-2 focus-visible:outline-offset-2';
  
  let variantClass = 'btn-primary';
  if (variant === 'secondary') variantClass = 'btn-secondary';
  if (variant === 'outline') variantClass = 'btn-outline';
  if (variant === 'danger') variantClass = 'btn-danger';
  if (variant === 'ghost') variantClass = 'btn-ghost';

  return (
    <button
      type={type}
      disabled={disabled || loading}
      onClick={onClick}
      className={`${baseClasses} ${variantClass} btn-${size} ${className}`}
      {...props}
    >
      {loading && (
        <span className="spinner mr-2" aria-hidden="true" />
      )}
      {children}
    </button>
  );
}

