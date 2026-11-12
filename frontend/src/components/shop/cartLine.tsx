'use client';

import { useLocale, useTranslations } from 'next-intl';
import { Trash2 } from 'lucide-react';

import ProductArt from './productArt';
import QuantityStepper from './quantityStepper';
import { formatNumber, formatToman } from '@/src/lib/format';
import type { CartLine as CartLineData } from '@/src/lib/api/types/schemas';

type CartLineProps = {
  line: CartLineData;
  onQuantityChange: (quantity: number) => void;
  onRemove: () => void;
  disabled?: boolean;
};

export default function CartLine({
  line,
  onQuantityChange,
  onRemove,
  disabled = false,
}: CartLineProps) {
  const t = useTranslations('cart');
  const tc = useTranslations('common');
  const locale = useLocale() as 'fa' | 'en';

  return (
    <div className="flex items-center gap-4 border-b border-line py-5 last:border-b-0">
      <div className="flex h-20 w-20 shrink-0 items-center justify-center rounded-xl bg-smoke p-2">
        <ProductArt
          slug={line.productSlug}
          colorKey={line.colorKey}
          className="h-full w-auto object-contain"
        />
      </div>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <h3 className="truncate text-sm font-semibold text-ink sm:text-base">
          {line.productName}
        </h3>
        <p className="text-xs text-ash">
          {line.colorName} · {formatNumber(line.storageGb, locale)} {tc('storageGb')}
        </p>
        <div className="mt-2">
          <QuantityStepper
            value={line.quantity}
            min={1}
            max={Math.max(line.maxQuantity, 1)}
            onChange={onQuantityChange}
            disabled={disabled}
          />
        </div>
      </div>

      <div className="flex shrink-0 flex-col items-end gap-3">
        <span className="text-sm font-bold text-ink sm:text-base">
          {formatToman(line.lineTotalIrr, locale)} {t('toman')}
        </span>
        <button
          type="button"
          aria-label={t('remove')}
          onClick={onRemove}
          disabled={disabled}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-ash transition-colors hover:bg-smoke hover:text-ink disabled:opacity-40"
        >
          <Trash2 size={16} />
        </button>
      </div>
    </div>
  );
}
