'use client';

type Option = {
  label: string;
  value: string;
};

type Props = {
  value: string;
  onChange: (value: string) => void;
  options: Option[];
  label?: string;
};

export default function Select({ value, onChange, options, label }: Props) {
  return (
    <div className="flex flex-col gap-1">
      {label && (
        <label className="text-xs font-medium text-coal">{label}</label>
      )}

      <div className="relative">
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="
          w-full rounded-lg border border-line bg-paper
          px-3 py-2.5 text-sm text-ink outline-none
          transition-all duration-200
          hover:border-ash
          focus:border-ink focus:ring-2 focus:ring-ink/10
          "
        >
          {options.map((opt) => (
            <option
              key={opt.value}
              value={opt.value}
              className="bg-paper text-ink"
            >
              {opt.label}
            </option>
          ))}
        </select>

        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ash">
          ▾
        </span>
      </div>
    </div>
  );
}
