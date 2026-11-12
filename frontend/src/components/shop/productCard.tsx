'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import ProductArt from './productArt';
import RatingStars from './ratingStars';
import { formatNumber, formatToman } from '@/src/lib/format';

export type ProductCardProps = {
  slug: string;
  name: string;
  /** Catalog spec line inputs: storage tier + color of the default variant. */
  storageGb: number;
  colorName: string;
  /** Stable backend color key the imagery map resolves by. */
  colorKey: string;
  priceIrr: number;
  compareAtPriceIrr?: number | null;
  rating: number;
  reviewCount: number;
};

export default function ProductCard({
  slug,
  name,
  storageGb,
  colorName,
  colorKey,
  priceIrr,
  compareAtPriceIrr,
  rating,
  reviewCount,
}: ProductCardProps) {
  const t = useTranslations('products');
  const locale = useLocale() as 'fa' | 'en';

  return (
    <Link
      href={`/products/${slug}`}
      className="
        group flex flex-col gap-3 rounded-2xl border border-line bg-paper p-4
        transition-all duration-200
        hover:-translate-y-1 hover:border-ink/20 hover:shadow-lg hover:shadow-ink/10
      "
    >
      <div className="flex aspect-square items-center justify-center rounded-xl bg-smoke p-6">
        <ProductArt
          slug={slug}
          colorKey={colorKey}
          className="h-full w-auto object-contain transition-transform duration-300 group-hover:scale-105"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <h3 className="text-sm font-semibold text-ink sm:text-base">{name}</h3>
        <p className="text-xs text-ash">
          {formatNumber(storageGb, locale)} {t('storageGb')} · {colorName}
        </p>
        <div className="flex items-center gap-2">
          <RatingStars value={rating} />
          <span className="text-[11px] text-ash">
            ({formatNumber(reviewCount, locale)})
          </span>
        </div>
      </div>

      <div className="mt-auto flex items-baseline gap-2 pt-1">
        <span className="text-base font-bold text-ink sm:text-lg">
          {formatToman(priceIrr, locale)} {t('toman')}
        </span>
        {compareAtPriceIrr != null && (
          <span className="text-xs text-ash line-through">
            {formatToman(compareAtPriceIrr, locale)} {t('toman')}
          </span>
        )}
      </div>
    </Link>
  );
}
