'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useTransition } from 'react';

import { localeCookieName } from '@/src/i18n/config';

/**
 * Header language switch: writes the locale cookie, updates <html>
 * lang/dir immediately and refreshes the server tree so every
 * translation re-renders. The locale itself always resolves on the
 * server (cookie → request config), so there is no hydration mismatch.
 */
export default function LanguageSwitch() {
  const locale = useLocale();
  const router = useRouter();
  const t = useTranslations('header');
  const [pending, startTransition] = useTransition();

  const nextLocale = locale === 'fa' ? 'en' : 'fa';

  const switchLocale = () => {
    document.cookie = `${localeCookieName}=${nextLocale}; path=/; max-age=31536000; samesite=lax`;
    document.documentElement.lang = nextLocale;
    document.documentElement.dir = nextLocale === 'fa' ? 'rtl' : 'ltr';
    startTransition(() => router.refresh());
  };

  return (
    <button
      type="button"
      onClick={switchLocale}
      disabled={pending}
      aria-label={locale === 'fa' ? t('switchToEn') : t('switchToFa')}
      className="
        flex h-8 w-8 items-center justify-center rounded-lg bg-ink
        text-sm font-black text-brand transition-colors
        hover:bg-coal disabled:cursor-wait disabled:opacity-60
        focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand
      "
    >
      {locale === 'fa' ? 'FA' : 'EN'}
    </button>
  );
}
