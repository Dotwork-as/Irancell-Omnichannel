type Props =
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?:
      | 'primary'
      | 'brand'
      | 'secondary'
      | 'ghost'
      | 'outline'
      | 'onDark';

    size?: 'xs' | 'sm' | 'md' | 'lg';

    fullWidth?: boolean;

    loading?: boolean;
  };

export default function Button({
  variant = 'primary',
  size = 'md',
  fullWidth = false,
  loading = false,
  disabled,
  className = '',
  children,
  ...props
}: Props) {
  const baseClasses =
    'inline-flex items-center justify-center gap-2 font-medium transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed';

  const sizeClasses = {
    xs: 'px-2 py-1 text-xs rounded-md',
    sm: 'px-3 py-1.5 text-sm rounded-md',
    md: 'px-5 py-2.5 text-sm rounded-md',
    lg: 'px-8 py-4 text-base rounded-md',
  };

  const variantClasses = {
    primary: 'bg-ink text-paper hover:bg-coal',

    brand: 'bg-brand text-ink hover:bg-brand/90',

    secondary: 'bg-smoke text-ink hover:bg-line',

    ghost: 'bg-transparent text-coal hover:bg-smoke hover:text-ink',

    outline: 'border border-ink bg-paper text-ink hover:bg-smoke',

    onDark: 'border border-paper/60 bg-transparent text-paper hover:bg-paper/10',
  };

  return (
    <button
      disabled={disabled || loading}
      className={`
        ${baseClasses}
        ${sizeClasses[size]}
        ${variantClasses[variant]}
        ${fullWidth ? 'w-full' : ''}
        ${className}
      `}
      {...props}
    >
      {loading && (
        <div className="h-4 w-4 animate-spin rounded-full border-2 border-current/30 border-t-current" />
      )}

      {children}
    </button>
  );
}
