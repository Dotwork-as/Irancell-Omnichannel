export const locales = ['fa', 'en'] as const;
export type AppLocale = (typeof locales)[number];

export const defaultLocale: AppLocale = 'fa';

export const localeCookieName = 'locale';
