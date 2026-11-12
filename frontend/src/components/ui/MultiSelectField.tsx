'use client';

type Option = {
  label: string;
  value: string;
};

type Props = {
  label?: string;
  values: string[];
  options: Option[];
  onChange: (values: string[]) => void;
};

export function MultiSelectField({ label, values, options, onChange }: Props) {
  const toggle = (val: string) => {
    if (values.includes(val)) {
      onChange(values.filter((v) => v !== val));
    } else {
      onChange([...values, val]);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      {/* LABEL */}
      {label && (
        <label className="text-xs font-medium text-coal">{label}</label>
      )}

      {/* OPTIONS */}
      <div className="flex flex-wrap gap-2">
        {options.map((opt) => {
          const active = values.includes(opt.value);

          return (
            <button
              key={opt.value}
              type="button"
              onClick={() => toggle(opt.value)}
              className={`
                cursor-pointer rounded-full px-3 py-1 text-xs
                border transition-all duration-200
                select-none

                ${
                  active
                    ? 'bg-ink text-paper border-ink shadow-sm'
                    : 'border-line text-coal bg-paper hover:bg-smoke hover:border-ink/30'
                }
              `}
            >
              {opt.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}
