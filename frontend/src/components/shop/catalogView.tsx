'use client';

import { useTranslations } from 'next-intl';
import { motion } from 'framer-motion';

import EmptyState from '@/src/components/layout/emptyState';
import ProductGrid from '@/src/components/shop/productGrid';
import SectionHeading from '@/src/components/shop/sectionHeading';
import { SkeletonBlock } from '@/src/components/skeleton/skeletonBlock';
import { useProducts } from '@/src/hooks/useProducts';

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function CatalogView({ q }: { q?: string }) {
  const t = useTranslations('products');
  const { products, loading, error, refetch } = useProducts(q);

  const cards = products.map((p) => ({
    slug: p.slug,
    name: p.name,
    storageGb: p.storageGb,
    colorName: p.colorName,
    priceIrr: p.priceIrr,
    compareAtPriceIrr: p.compareAtPriceIrr,
    rating: p.rating,
    reviewCount: p.reviewCount,
    colorKey: p.colorKey,
  }));

  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 pt-8 pb-10 sm:px-6">
      <motion.div {...revealProps}>
        <SectionHeading
          title={t('title')}
          description={q ? t('searchingFor', { q }) : t('description')}
        />
      </motion.div>

      {loading && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((n) => (
            <SkeletonBlock key={n} className="aspect-[4/5] rounded-2xl" />
          ))}
        </div>
      )}

      {!loading && error && (
        <EmptyState
          message={t('error')}
          description={error.message}
          action={
            <button
              type="button"
              onClick={refetch}
              className="rounded-lg border border-line px-4 py-2 text-sm font-medium text-ink transition-colors hover:border-ink/40"
            >
              {t('retry')}
            </button>
          }
        />
      )}

      {!loading && !error && cards.length === 0 && (
        <EmptyState message={t('noResults')} description={t('noResultsHint')} />
      )}

      {!loading && !error && cards.length > 0 && (
        <ProductGrid items={cards} />
      )}
    </main>
  );
}
