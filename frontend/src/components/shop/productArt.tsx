'use client';

import { useTranslations } from 'next-intl';

import PhoneArt from './phoneArt';
import { productImageFor } from '@/src/lib/productImages';
import { colorHexFor } from '@/src/lib/productColors';

type ProductArtProps = {
  slug: string;
  colorKey: string;
  className?: string;
};

/**
 * Product imagery: renders the genuine Figma-sourced asset for the
 * product/color when the kit provides one, and keeps the tinted vector
 * illustration (PhoneArt) as the fallback for every product the kit
 * does not cover — never a broken reference or a mismatched photo.
 */
export default function ProductArt({
  slug,
  colorKey,
  className = '',
}: ProductArtProps) {
  const t = useTranslations('phoneArt');
  const src = productImageFor(slug, colorKey);

  if (!src) {
    return <PhoneArt body={colorHexFor(slug, colorKey)} className={className} />;
  }

  // Plain <img> on purpose: local static assets with CSS-controlled
  // sizing; next/image would add an on-demand optimizer dependency the
  // standalone Docker runtime does not ship.
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={src} alt={t('aria')} className={className} />
  );
}
