import { cookies } from 'next/headers';
import { getRequestConfig } from 'next-intl/server';

import { defaultLocale, localeCookieName } from './config';

/**
 * Locale resolution WITHOUT i18n routing: existing routes are preserved
 * as-is and the locale rides on a cookie (default fa). This makes the
 * routes dynamic (cookies() cannot be statically prerendered) — an
 * accepted trade-off for keeping every URL unchanged.
 */
export default getRequestConfig(async () => {
  const store = await cookies();
  const requested = store.get(localeCookieName)?.value;

  const locale =
    requested === 'en' || requested === 'fa' ? requested : defaultLocale;

  return {
    locale,
    messages: (await import(`../../messages/${locale}.json`)).default,
  };
});
