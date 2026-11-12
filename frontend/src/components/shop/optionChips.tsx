'use client';

type Option = {
  value: string;
  label: string;
  /** Optional color dot (for color pickers). */
  dot?: string;
  /** Unavailable combinations render muted and stay unselectable. */
  disabled?: boolean;
};

type OptionChipsProps = {
  options: Option[];
  selected: string;
  onSelect: (value: string) => void;
  ariaLabel?: string;
};

export default function OptionChips({
  options,
  selected,
  onSelect,
  ariaLabel,
}: OptionChipsProps) {
  return (
    <div
      role="group"
      aria-label={ariaLabel}
      className="flex flex-wrap items-center gap-2"
    >
      {options.map(({ value, label, dot, disabled = false }) => {
        const isActive = value === selected;

        if (disabled) {
          return (
            <span
              key={value}
              aria-disabled="true"
              className="
                flex cursor-not-allowed items-center gap-2 rounded-lg border border-line
                bg-smoke/60 px-4 py-2 text-sm font-medium text-ash line-through opacity-70
              "
            >
              {dot && (
                <span
                  className="h-3.5 w-3.5 rounded-full border border-black/20"
                  style={{ backgroundColor: dot }}
                />
              )}
              {label}
            </span>
          );
        }

        return (
          <button
            key={value}
            type="button"
            aria-pressed={isActive}
            onClick={() => onSelect(value)}
            className={`
              flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium
              transition-all duration-200
              ${
                isActive
                  ? 'border-ink bg-ink text-paper'
                  : 'border-line bg-paper text-coal hover:border-ink/40 hover:text-ink'
              }
            `}
          >
            {dot && (
              <span
                className="h-3.5 w-3.5 rounded-full border border-black/20"
                style={{ backgroundColor: dot }}
              />
            )}
            {label}
          </button>
        );
      })}
    </div>
  );
}
