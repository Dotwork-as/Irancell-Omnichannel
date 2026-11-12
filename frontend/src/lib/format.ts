import type { AppLocale } from '@/src/i18n/config';

/**
 * Money presentation helper: the backend contract speaks integer IRR
 * (docs/API.md §4); the storefront displays toman (IRR ÷ 10) with
 * locale-aware digits — a presentation-only conversion, the API never
 * sees toman values.
 */
export function formatToman(priceIrr: number, locale: AppLocale): string {
  return new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
    maximumFractionDigits: 0,
  }).format(priceIrr / 10);
}

/** Locale-aware integer formatting (counts, quantities). */
export function formatNumber(value: number, locale: AppLocale): string {
  return new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
    maximumFractionDigits: 0,
  }).format(value);
}
