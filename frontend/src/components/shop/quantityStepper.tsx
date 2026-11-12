'use client';

import { useLocale, useTranslations } from 'next-intl';
import { Minus, Plus } from 'lucide-react';

type QuantityStepperProps = {
  /** Controlled value — the source of truth lives with the caller. */
  value: number;
  onChange: (value: number) => void;
  min?: number;
  /** Upper bound (live stock / 99), mirrors the backend constraint. */
  max?: number;
  disabled?: boolean;
};

export default function QuantityStepper({
  value,
  onChange,
  min = 1,
  max = 99,
  disabled = false,
}: QuantityStepperProps) {
  const t = useTranslations('quantity');
  const locale = useLocale();

  return (
    <div className="flex h-10 items-center overflow-hidden rounded-lg border border-line bg-paper">
      <button
        type="button"
        aria-label={t('increase')}
        onClick={() => onChange(Math.min(max, value + 1))}
        className="flex h-full w-9 items-center justify-center text-coal transition-colors hover:bg-smoke hover:text-ink disabled:opacity-40"
        disabled={disabled || value >= max}
      >
        <Plus size={14} />
      </button>

      <span className="w-8 text-center text-sm font-semibold text-ink">
        {value.toLocaleString(locale)}
      </span>

      <button
        type="button"
        aria-label={t('decrease')}
        onClick={() => onChange(Math.max(min, value - 1))}
        className="flex h-full w-9 items-center justify-center text-coal transition-colors hover:bg-smoke hover:text-ink disabled:opacity-40"
        disabled={disabled || value <= min}
      >
        <Minus size={14} />
      </button>
    </div>
  );
}
