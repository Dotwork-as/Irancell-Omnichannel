'use client';

import Link from 'next/link';
import { motion } from 'framer-motion';
import { useTranslations } from 'next-intl';
import {
  ArrowLeft,
  BadgeCheck,
  CreditCard,
  Truck,
  type LucideIcon,
} from 'lucide-react';

const primaryLinkClass =
  'inline-flex min-h-12 items-center justify-center gap-2 rounded-lg bg-brand px-6 py-3 text-sm font-bold text-ink transition-colors hover:bg-brand/90 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-brand';

const secondaryLinkClass =
  'inline-flex min-h-12 items-center justify-center rounded-lg border border-paper/30 px-6 py-3 text-sm font-bold text-paper transition-colors hover:bg-paper/10 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-paper';

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function HomePage() {
  const t = useTranslations('home');

  const features: {
    icon: LucideIcon;
    title: string;
    text: string;
  }[] = [
    {
      icon: Truck,
      title: t('features.shipping.title'),
      text: t('features.shipping.text'),
    },
    {
      icon: BadgeCheck,
      title: t('features.authenticity.title'),
      text: t('features.authenticity.text'),
    },
    {
      icon: CreditCard,
      title: t('features.payment.title'),
      text: t('features.payment.text'),
    },
  ];

  return (
    <main className="flex flex-col">
      {/* Hero */}
      <section className="bg-ink text-paper">
        <div className="mx-auto grid w-full max-w-6xl items-center gap-10 px-4 py-16 sm:px-6 md:grid-cols-2 md:py-24">
          <motion.div
            {...revealProps}
            className="flex flex-col items-start gap-6"
          >
            <span className="rounded-full bg-brand/15 px-4 py-1.5 text-xs font-bold text-brand">
              {t('heroBadge')}
            </span>

            <h1 className="text-3xl font-extrabold leading-tight sm:text-4xl md:text-5xl">
              {t('heroTitle')}
              <span className="mt-2 block text-paper/60">
                <span className="block">{t('heroSubtitle1')}</span>
                <span className="block">{t('heroSubtitle2')}</span>
              </span>
            </h1>

            <p className="max-w-md text-sm leading-7 text-paper/70 sm:text-base">
              {t('heroDescription')}
            </p>

            <div className="flex flex-wrap items-center gap-3">
              <Link
                href="/products/iphone-15-pro-max"
                className={primaryLinkClass}
              >
                {t('heroCtaPrimary')}
                <ArrowLeft size={18} aria-hidden="true" />
              </Link>

              <Link href="/products" className={secondaryLinkClass}>
                {t('heroCtaSecondary')}
              </Link>
            </div>
          </motion.div>

          <motion.div
            {...revealProps}
            transition={{ duration: 0.6, ease: 'easeOut', delay: 0.15 }}
            className="relative mx-auto hidden h-80 w-full items-center justify-center md:flex"
          >
            <span
              aria-hidden="true"
              className="absolute h-64 w-64 rounded-full bg-brand/20 blur-3xl"
            />

            {/* Figma kit hero asset (E-Commerce UI Kit export) —
                plain <img>: local static asset, CSS-controlled sizing. */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/home/iphone-14-pro-deep-purple.png"
              alt={t('heroImageAlt')}
              className="relative h-full w-auto object-contain drop-shadow-2xl"
            />
          </motion.div>
        </div>
      </section>

      {/* Features */}
      <section className="border-b border-line">
        <div className="mx-auto grid w-full max-w-6xl gap-6 px-4 py-10 sm:grid-cols-3 sm:px-6">
          {features.map(({ icon: Icon, title, text }, index) => (
            <motion.div
              key={title}
              {...revealProps}
              transition={{
                duration: 0.5,
                ease: 'easeOut',
                delay: index * 0.12,
              }}
              className="flex items-start gap-4"
            >
              <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-smoke text-ink">
                <Icon size={22} aria-hidden="true" />
              </span>

              <div className="flex flex-col gap-1">
                <h2 className="text-sm font-bold text-ink">{title}</h2>
                <p className="text-xs leading-5 text-ash">{text}</p>
              </div>
            </motion.div>
          ))}
        </div>
      </section>

      {/* Promo Banner */}
      <section className="mx-auto w-full max-w-6xl px-4 pt-14 sm:px-6">
        <motion.div
          {...revealProps}
          className="flex flex-col items-start justify-between gap-6 rounded-2xl bg-brand p-8 sm:flex-row sm:items-center"
        >
          <div className="flex flex-col gap-2">
            <h2 className="text-xl font-extrabold text-ink sm:text-2xl">
              {t('promo.title')}
            </h2>

            <p className="text-sm leading-7 text-ink/70">{t('promo.text')}</p>
          </div>

          <Link
            href="/products"
            className="inline-flex min-h-12 shrink-0 items-center justify-center rounded-lg bg-ink px-6 py-3 text-sm font-bold text-paper transition-colors hover:bg-coal focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ink"
          >
            {t('promo.cta')}
          </Link>
        </motion.div>
      </section>
    </main>
  );
}
