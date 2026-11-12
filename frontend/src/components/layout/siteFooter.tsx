'use client';

import { motion } from 'framer-motion';
import { useTranslations } from 'next-intl';
import { Instagram, Linkedin, Send, Youtube } from 'lucide-react';

const socials = [Instagram, Send, Youtube, Linkedin];

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function SiteFooter() {
  const t = useTranslations('footer');

  const columns = [
    { title: t('services.title'), items: t.raw('services.items') as string[] },
    { title: t('guide.title'), items: t.raw('guide.items') as string[] },
  ];

  return (
    <footer className="mt-20 bg-ink text-paper">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-12 px-4 py-14 sm:px-6">
        <div className="grid gap-10 md:grid-cols-3">
          <motion.div
            {...revealProps}
            className="flex flex-col items-start gap-4"
          >
            <span className="text-lg font-extrabold tracking-tight">
              {t('brand')}
              <span className="text-brand">.</span>
            </span>
            <p className="max-w-xs text-sm leading-6 text-paper/60">
              {t('about')}
            </p>
          </motion.div>

          {columns.map(({ title, items }, index) => (
            <motion.div
              key={title}
              {...revealProps}
              transition={{
                duration: 0.5,
                ease: 'easeOut',
                delay: (index + 1) * 0.1,
              }}
              className="flex flex-col gap-4"
            >
              <h3 className="text-base font-semibold">{title}</h3>
              <ul className="flex flex-col gap-3">
                {items.map((item) => (
                  <li key={item}>
                    <span className="cursor-pointer text-sm text-paper/60 transition-colors hover:text-brand">
                      {item}
                    </span>
                  </li>
                ))}
              </ul>
            </motion.div>
          ))}
        </div>

        <motion.div
          {...revealProps}
          transition={{ duration: 0.5, ease: 'easeOut', delay: 0.2 }}
          className="flex flex-col items-center justify-between gap-6 border-t border-paper/10 pt-8 sm:flex-row"
        >
          <div className="flex items-center gap-3">
            {socials.map((Icon, index) => (
              <span
                key={index}
                className="flex h-10 w-10 cursor-pointer items-center justify-center rounded-xl bg-paper/10 text-paper transition-colors hover:bg-brand hover:text-ink"
              >
                <Icon size={18} />
              </span>
            ))}
          </div>

          <p className="text-xs text-paper/50">{t('copyright')}</p>
        </motion.div>
      </div>
    </footer>
  );
}
