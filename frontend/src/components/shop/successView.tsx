'use client';

import Link from 'next/link';
import { useEffect, useSyncExternalStore } from 'react';
import { useRouter } from 'next/navigation';
import { motion } from 'framer-motion';
import { useLocale, useTranslations } from 'next-intl';
import { Check, PackageSearch } from 'lucide-react';

import Button from '@/src/components/ui/Button';
import { getStoredReceipt } from '@/src/hooks/useCheckout';
import { formatToman } from '@/src/lib/format';

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function SuccessView() {
  const t = useTranslations('success');
  const locale = useLocale() as 'fa' | 'en';
  const router = useRouter();
  // Storage only exists on the client: mounted flips after hydration
  // (useSyncExternalStore keeps this effect- and setState-free).
  const mounted = useSyncExternalStore(
    () => () => {},
    () => true,
    () => false,
  );
  const receipt = mounted ? getStoredReceipt() : null;

  useEffect(() => {
    if (mounted && !receipt) {
      // No checkout happened in this session — the page is not
      // reachable directly.
      router.replace('/');
    }
  }, [mounted, receipt, router]);

  if (!mounted || receipt === null) {
    return null;
  }

  return (
    <main className="mx-auto flex w-full max-w-xl flex-col items-center gap-6 px-4 pt-20 pb-10 text-center sm:px-6">
      <motion.span
        {...revealProps}
        initial={{ opacity: 0, y: 24, scale: 0.97 }}
        whileInView={{ opacity: 1, y: 0, scale: 1 }}
        className="flex h-24 w-24 items-center justify-center rounded-full bg-brand shadow-xl shadow-brand/30"
      >
        <Check size={48} strokeWidth={3} className="text-ink" />
      </motion.span>

      <motion.div
        {...revealProps}
        transition={{ duration: 0.5, ease: 'easeOut', delay: 0.12 }}
        className="flex flex-col items-center gap-3"
      >
        <h1 className="text-2xl font-extrabold text-ink">{t('heading')}</h1>

        <p className="text-sm leading-7 text-coal">
          {t('statusPaid')}
        </p>

        <p className="text-sm leading-7 text-coal">
          {t.rich('trackingText', {
            code: receipt.trackingCode,
            ltr: (chunks) => (
              <span dir="ltr" className="font-bold text-ink">
                {chunks}
              </span>
            ),
          })}
        </p>

        <p className="text-sm font-bold text-ink">
          {t('total')}: {formatToman(receipt.totalIrr, locale)} {t('toman')}
        </p>

        <p className="text-xs leading-6 text-ash">{t('disclaimer')}</p>
      </motion.div>

      <motion.div
        {...revealProps}
        transition={{ duration: 0.5, ease: 'easeOut', delay: 0.24 }}
        className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row"
      >
        <Link href="/products">
          <Button variant="brand" size="lg" className="cursor-pointer">
            <PackageSearch size={18} />
            {t('backToShop')}
          </Button>
        </Link>

        <Link href="/">
          <Button variant="outline" size="lg" className="cursor-pointer">
            {t('home')}
          </Button>
        </Link>
      </motion.div>
    </main>
  );
}
