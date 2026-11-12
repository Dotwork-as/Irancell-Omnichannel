'use client';

type Props = {
  value: string | number;
  onChange: (value: string) => void;
  type?: 'text' | 'number';
  label: string;
  placeholder?: string;
};

export default function Input({
  value,
  onChange,
  type = 'text',
  label,
  placeholder,
}: Props) {
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label className="text-xs font-medium text-coal">{label}</label>
      )}

      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="
          w-full rounded-lg border border-line bg-paper
          px-3 py-3 text-sm text-ink outline-none
          transition-all duration-200 placeholder:text-ash
          hover:border-ash
          focus:border-ink focus:ring-2 focus:ring-ink/10
        "
      />
    </div>
  );
}
