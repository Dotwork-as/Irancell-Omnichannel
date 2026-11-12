'use client';

import Link from 'next/link';
import { motion } from 'framer-motion';
import { useLocale, useTranslations } from 'next-intl';

import CartLine from '@/src/components/shop/cartLine';
import OrderSummary from '@/src/components/shop/orderSummary';
import EmptyState from '@/src/components/layout/emptyState';
import { SkeletonBlock } from '@/src/components/skeleton/skeletonBlock';

import Button from '@/src/components/ui/Button';
import { useCart } from '@/src/hooks/useCart';
import { useRemoveCartItem } from '@/src/hooks/useRemoveCartItem';
import { useUpdateCartItem } from '@/src/hooks/useUpdateCartItem';
import { formatNumber, formatToman } from '@/src/lib/format';

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function CartView() {
  const t = useTranslations('cart');
  const locale = useLocale() as 'fa' | 'en';
  const { cart, loading, error, refetch } = useCart();
  const { updateCartItem, pending: updatePending, error: updateError } =
    useUpdateCartItem();
  const { removeCartItem, pending: removePending, error: removeError } =
    useRemoveCartItem();

  const busy = updatePending || removePending;
  const isEmpty = !loading && (cart?.items.length ?? 0) === 0;

  if (loading) {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-8 pb-10 sm:px-6">
        <SkeletonBlock className="h-8 w-40" />
        <div className="mt-8 grid gap-8 lg:grid-cols-[1fr_340px]">
          <div className="flex flex-col gap-4">
            {[1, 2].map((n) => (
              <SkeletonBlock key={n} className="h-28 rounded-2xl" />
            ))}
          </div>
          <SkeletonBlock className="h-64 rounded-2xl" />
        </div>
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
              onClick={() => refetch()}
              className="rounded-lg border border-line px-4 py-2 text-sm font-medium text-ink transition-colors hover:border-ink/40"
            >
              {t('retry')}
            </button>
          }
        />
      </main>
    );
  }

  if (isEmpty) {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-16 pb-10 sm:px-6">
        <EmptyState
          message={t('empty')}
          description={t('emptyDescription')}
          action={
            <Link href="/products" className="inline-block">
              <Button variant="brand">{t('browse')}</Button>
            </Link>
          }
        />
      </main>
    );
  }

  const cartItems = cart?.items ?? [];

  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 pt-8 pb-10 sm:px-6">
      <motion.div {...revealProps}>
        <h1 className="text-2xl font-bold text-ink">{t('title')}</h1>
      </motion.div>

      <div className="grid gap-8 lg:grid-cols-[1fr_340px]">
        <section className="flex flex-col">
          {cartItems.map((line, index) => (
            <motion.div
              key={line.id}
              {...revealProps}
              transition={{
                duration: 0.5,
                ease: 'easeOut',
                delay: Math.min(index * 0.08, 0.3),
              }}
            >
              <CartLine
                line={line}
                disabled={busy}
                onQuantityChange={(quantity) => {
                  if (quantity !== line.quantity) {
                    void updateCartItem(line.id, quantity);
                  }
                }}
                onRemove={() => void removeCartItem(line.id)}
              />
            </motion.div>
          ))}

          {(updateError ?? removeError) && (
            <p className="mt-4 rounded-lg bg-smoke px-3 py-2 text-xs text-coal" role="alert">
              {(updateError ?? removeError)?.message}
            </p>
          )}
        </section>

        <OrderSummary
          title={t('summaryTitle')}
          rows={[
            {
              label: t('itemCountLabel'),
              value: formatNumber(cart?.itemCount ?? 0, locale),
            },
          ]}
          totalLabel={t('subtotal')}
          total={`${formatToman(cart?.subtotalIrr ?? 0, locale)} ${t('toman')}`}
        >
          <Link href="/checkout" className="block">
            <Button
              variant="brand"
              size="lg"
              fullWidth
              className="cursor-pointer"
              disabled={busy}
            >
              {t('checkoutCta')}
            </Button>
          </Link>

          <Link
            href="/products"
            className="text-center text-xs font-medium text-ash transition-colors hover:text-ink"
          >
            {t('backToShop')}
          </Link>
        </OrderSummary>
      </div>
    </main>
  );
}
