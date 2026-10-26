import React from 'react';

export function Card({
  title,
  subtitle,
  action,
  children,
  className = '',
  headerClassName = '',
  bodyClassName = '',
  ...props
}) {
  return (
    <div className={`card ${className}`} {...props}>
      {(title || subtitle || action) && (
        <div className={`card-header ${headerClassName}`}>
          <div>
            {title && <h3 className="card-title">{title}</h3>}
            {subtitle && <p className="card-subtitle">{subtitle}</p>}
          </div>
          {action && <div className="card-action">{action}</div>}
        </div>
      )}
      <div className={`card-body ${bodyClassName}`}>
        {children}
      </div>
    </div>
  );
}

