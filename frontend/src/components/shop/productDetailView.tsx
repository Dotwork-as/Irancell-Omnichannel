'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { ChevronLeft, ShieldCheck, ShoppingCart, Truck } from 'lucide-react';

import EmptyState from '@/src/components/layout/emptyState';
import OptionChips from '@/src/components/shop/optionChips';
import ProductArt from '@/src/components/shop/productArt';
import QuantityStepper from '@/src/components/shop/quantityStepper';
import RatingStars from '@/src/components/shop/ratingStars';
import { SkeletonBlock } from '@/src/components/skeleton/skeletonBlock';

import Button from '@/src/components/ui/Button';
import { useAddToCart } from '@/src/hooks/useAddToCart';
import { useProduct } from '@/src/hooks/useProduct';
import { formatNumber, formatToman } from '@/src/lib/format';
import { colorHexFor } from '@/src/lib/productColors';

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function ProductDetailView({ slug }: { slug: string }) {
  const t = useTranslations('productDetail');
  const tc = useTranslations('common');
  const locale = useLocale() as 'fa' | 'en';
  const router = useRouter();
  const { product, loading, notFound, error, refetch } = useProduct(slug);
  const { addToCart, pending: addPending, error: addError } = useAddToCart();

  const [colorKey, setColorKey] = useState<string | null>(null);
  const [storageGb, setStorageGb] = useState<number | null>(null);
  const [quantity, setQuantity] = useState(1);
  const [added, setAdded] = useState(false);

  // Variant selection: derive the option sets from the variants the
  // backend actually defines; picking a color snaps storage to the
  // nearest existing combination (not every pairing exists).
  const selectedVariant = useMemo(() => {
    if (!product) return null;
    const colors = [...new Set(product.variants.map((v) => v.colorKey))];
    const activeColor = colorKey ?? colors[0] ?? null;
    const forColor = product.variants.filter((v) => v.colorKey === activeColor);
    const storages = [...new Set(forColor.map((v) => v.storageGb))];
    const activeStorage =
      storageGb != null && storages.includes(storageGb)
        ? storageGb
        : (forColor.find((v) => v.inStock) ?? forColor[0])?.storageGb ?? null;
    return {
      colors,
      activeColor,
      forColor,
      storages,
      activeStorage,
      variant:
        forColor.find((v) => v.storageGb === activeStorage) ?? null,
    };
  }, [product, colorKey, storageGb]);

  const selectColor = (key: string) => {
    setColorKey(key);
    setStorageGb(null); // snap storage to the nearest existing combo
    setQuantity(1);
    setAdded(false);
  };

  const selectStorage = (gb: number) => {
    setStorageGb(gb);
    setQuantity(1);
    setAdded(false);
  };

  const submitAdd = async (thenNavigate: boolean) => {
    if (!selectedVariant?.variant || !selectedVariant.variant.inStock) return;
    const cart = await addToCart(selectedVariant.variant.id, quantity);
    if (!cart) return;
    setAdded(true);
    if (thenNavigate) router.push('/checkout');
  };

  if (loading) {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-8 pb-10 sm:px-6">
        <div className="grid gap-10 md:grid-cols-2">
          <SkeletonBlock className="aspect-square rounded-2xl" />
          <div className="flex flex-col gap-4">
            <SkeletonBlock className="h-6 w-24 rounded-full" />
            <SkeletonBlock className="h-9 w-2/3" />
            <SkeletonBlock className="h-7 w-40" />
            <SkeletonBlock className="h-12 w-full" />
            <SkeletonBlock className="h-12 w-full" />
          </div>
        </div>
      </main>
    );
  }

  if (notFound || !product) {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-16 pb-10 sm:px-6">
        <EmptyState
          message={t('notFound')}
          description={t('notFoundDescription')}
          action={
            <Link href="/products" className="inline-block">
              <Button variant="brand">{t('browse')}</Button>
            </Link>
          }
        />
      </main>
    );
  }

  if (error) {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-16 pb-10 sm:px-6">
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
      </main>
    );
  }

  const variant = selectedVariant?.variant ?? null;
  const outOfStock = !variant?.inStock;

  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-4 pt-8 pb-10 sm:px-6">
      {/* Breadcrumb */}
      <nav
        aria-label={t('breadcrumbLabel')}
        className="flex items-center gap-1 text-xs text-ash"
      >
        <Link href="/" className="transition-colors hover:text-ink">
          {t('breadcrumbHome')}
        </Link>
        <ChevronLeft size={12} />
        <Link href="/products" className="transition-colors hover:text-ink">
          {t('breadcrumbShop')}
        </Link>
        <ChevronLeft size={12} />
        <span className="font-medium text-ink">{product.name}</span>
      </nav>

      <div className="grid gap-10 md:grid-cols-2">
        {/* Gallery */}
        <div className="flex flex-col gap-4">
          <motion.div
            {...revealProps}
            initial={{ opacity: 0, y: 24, scale: 0.97 }}
            whileInView={{ opacity: 1, y: 0, scale: 1 }}
            className="relative flex aspect-square items-center justify-center overflow-hidden rounded-2xl bg-smoke p-12"
          >
            <AnimatePresence mode="wait">
              <motion.div
                key={selectedVariant?.activeColor ?? product.slug}
                initial={{ opacity: 0, scale: 0.94 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 1.04 }}
                transition={{ duration: 0.3, ease: 'easeOut' }}
                className="flex h-full items-center justify-center"
              >
                <ProductArt
                  slug={product.slug}
                  colorKey={selectedVariant?.activeColor ?? ''}
                  className="h-full w-auto object-contain drop-shadow-xl"
                />
              </motion.div>
            </AnimatePresence>
          </motion.div>

          <motion.div
            {...revealProps}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.1 }}
            className="grid grid-cols-4 gap-3"
          >
            {selectedVariant?.colors.map((key) => (
              <button
                key={key}
                type="button"
                aria-label={product.variants.find((v) => v.colorKey === key)?.colorName ?? key}
                aria-pressed={key === selectedVariant.activeColor}
                onClick={() => selectColor(key)}
                className={`
                  flex aspect-square items-center justify-center rounded-xl bg-smoke p-4
                  transition-all duration-200
                  ${
                    key === selectedVariant.activeColor
                      ? 'ring-2 ring-ink ring-offset-2 ring-offset-paper'
                      : 'hover:bg-line/60'
                  }
                `}
              >
                <ProductArt
                  slug={product.slug}
                  colorKey={key}
                  className="h-full w-auto object-contain"
                />
              </button>
            ))}
          </motion.div>
        </div>

        {/* Info */}
        <div className="flex flex-col gap-6">
          <motion.div {...revealProps} className="flex flex-col gap-3">
            <span className="w-fit rounded-full bg-brand px-3 py-1 text-[11px] font-bold text-ink">
              {t('badge')}
            </span>
            <h1 className="text-2xl font-extrabold text-ink sm:text-3xl">
              {product.name}
            </h1>
            <div className="flex items-center gap-2">
              <RatingStars value={product.rating} size={16} />
              <span className="text-xs text-ash">
                {formatNumber(product.rating, locale)} {t('ratingOutOf')} —{' '}
                {formatNumber(product.reviewCount, locale)} {t('reviewCountUnit')}
              </span>
            </div>
          </motion.div>

          <motion.div
            {...revealProps}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.08 }}
            className="flex items-baseline gap-3"
          >
            <span className="text-2xl font-extrabold text-ink">
              {variant ? `${formatToman(variant.priceIrr, locale)} ${tc('toman')}` : '—'}
            </span>
            {variant?.compareAtPriceIrr != null && (
              <span className="text-sm text-ash line-through">
                {formatToman(variant.compareAtPriceIrr, locale)} {tc('toman')}
              </span>
            )}
          </motion.div>

          {outOfStock && (
            <p className="w-fit rounded-lg bg-smoke px-3 py-1.5 text-xs font-bold text-coal">
              {t('outOfStock')}
            </p>
          )}

          <motion.div
            {...revealProps}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.16 }}
            className="flex flex-col gap-3"
          >
            <p className="text-sm font-bold text-ink">{t('chooseColor')}</p>
            <OptionChips
              options={(selectedVariant?.colors ?? []).map((key) => ({
                value: key,
                label:
                  product.variants.find((v) => v.colorKey === key)?.colorName ?? key,
                dot: colorHexFor(product.slug, key),
              }))}
              selected={selectedVariant?.activeColor ?? ''}
              onSelect={selectColor}
              ariaLabel={t('chooseColor')}
            />
          </motion.div>

          <motion.div
            {...revealProps}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.24 }}
            className="flex flex-col gap-3"
          >
            <p className="text-sm font-bold text-ink">{t('chooseStorage')}</p>
            <OptionChips
              options={(selectedVariant?.storages ?? []).map((gb) => {
                const combo = selectedVariant?.forColor.find((v) => v.storageGb === gb);
                return {
                  value: String(gb),
                  label: `${formatNumber(gb, locale)} ${tc('storageGb')}`,
                  disabled: !combo?.inStock,
                };
              })}
              selected={selectedVariant?.activeStorage != null ? String(selectedVariant.activeStorage) : ''}
              onSelect={(value) => selectStorage(Number(value))}
              ariaLabel={t('chooseStorage')}
            />
          </motion.div>

          <motion.div
            {...revealProps}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.32 }}
            className="flex items-center gap-4"
          >
            <QuantityStepper
              value={quantity}
              min={1}
              max={variant?.stockQty ?? 1}
              onChange={setQuantity}
              disabled={outOfStock}
            />
            {variant && variant.inStock && (
              <span className="text-xs text-ash">
                {formatNumber(variant.stockQty, locale)} {t('inStockUnit')}
              </span>
            )}
          </motion.div>

          <motion.div
            {...revealProps}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.32 }}
            className="flex flex-col gap-3 sm:flex-row"
          >
            <Button
              variant="brand"
              size="lg"
              fullWidth
              className="cursor-pointer"
              disabled={outOfStock || addPending}
              onClick={() => void submitAdd(false)}
            >
              <ShoppingCart size={18} />
              {addPending ? t('adding') : added ? t('added') : t('addToCart')}
            </Button>
            <Button
              variant="primary"
              size="lg"
              fullWidth
              className="cursor-pointer"
              disabled={outOfStock || addPending}
              onClick={() => void submitAdd(true)}
            >
              {t('buyNow')}
            </Button>
          </motion.div>

          {addError && (
            <p className="rounded-lg bg-smoke px-3 py-2 text-xs text-coal" role="alert">
              {addError.message}
            </p>
          )}

          <motion.div
            {...revealProps}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.4 }}
            className="grid gap-3 rounded-2xl border border-line p-4 sm:grid-cols-2"
          >
            <div className="flex items-center gap-3">
              <Truck size={20} className="shrink-0 text-coal" />
              <p className="text-xs leading-5 text-coal">{t('shippingNote')}</p>
            </div>
            <div className="flex items-center gap-3">
              <ShieldCheck size={20} className="shrink-0 text-coal" />
              <p className="text-xs leading-5 text-coal">{t('warrantyNote')}</p>
            </div>
          </motion.div>
        </div>
      </div>

      {/* Specs */}
      <section className="flex flex-col gap-5">
        <motion.div {...revealProps}>
          <h2 className="text-xl font-bold text-ink">{t('specsTitle')}</h2>
        </motion.div>

        <motion.div
          {...revealProps}
          transition={{ duration: 0.5, ease: 'easeOut', delay: 0.1 }}
        >
          <dl className="overflow-hidden rounded-2xl border border-line">
            {product.specs.map((spec, index) => (
              <div
                key={spec.label}
                className={`
                  flex flex-col gap-1 px-5 py-4 sm:flex-row sm:items-center sm:gap-4
                  ${index % 2 === 0 ? 'bg-paper' : 'bg-smoke/60'}
                `}
              >
                <dt className="w-40 shrink-0 text-sm font-medium text-ash">
                  {spec.label}
                </dt>
                <dd className="text-sm text-ink">{spec.value}</dd>
              </div>
            ))}
          </dl>
        </motion.div>
      </section>

      {/* Reviews */}
      <section className="flex flex-col gap-5">
        <motion.div {...revealProps}>
          <h2 className="text-xl font-bold text-ink">{t('reviewsTitle')}</h2>
        </motion.div>

        <div className="grid gap-4 md:grid-cols-3">
          {product.reviews.map((review, index) => (
            <motion.div
              key={review.author}
              {...revealProps}
              transition={{ duration: 0.5, ease: 'easeOut', delay: index * 0.1 }}
            >
              <article className="flex h-full flex-col gap-3 rounded-2xl border border-line p-5">
                <RatingStars value={review.rating} />
                <p className="text-sm leading-6 text-coal">{review.text}</p>
                <span className="text-xs font-medium text-ash">{review.author}</span>
              </article>
            </motion.div>
          ))}
        </div>
      </section>
    </main>
  );
}
