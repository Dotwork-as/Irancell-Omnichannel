'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';
import { Search, ShoppingBag } from 'lucide-react';

import LanguageSwitch from './languageSwitch';
import { useCart } from '@/src/hooks/useCart';
import { formatNumber } from '@/src/lib/format';

type NavLink = {
  href: string;
  labelKey: 'home' | 'shop';
};

const links: NavLink[] = [
  { href: '/', labelKey: 'home' },
  { href: '/products', labelKey: 'shop' },
];

export default function SiteHeader() {
  const pathname = usePathname();
  const router = useRouter();
  const locale = useLocale() as 'fa' | 'en';
  const t = useTranslations('header');
  const { cart } = useCart();
  const [searchOpen, setSearchOpen] = useState(false);
  const [term, setTerm] = useState('');

  const itemCount = cart?.itemCount ?? 0;

  const submitSearch = () => {
    const q = term.trim();
    if (q === '') {
      setSearchOpen(false);
      return;
    }
    setSearchOpen(false);
    router.push(`/products?q=${encodeURIComponent(q)}`);
  };

  return (
    <header className="sticky top-0 z-50 border-b border-line bg-paper/90 backdrop-blur-md">
      <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-6 px-4 sm:px-6">
        <div className="flex items-center gap-3">
          <LanguageSwitch />
          <Link href="/" className="flex items-center gap-2">
            <span className="text-lg font-extrabold tracking-tight text-ink">
              {t('brand')}
              <span className="text-brand">.</span>
            </span>
          </Link>
        </div>

        <nav className="hidden items-center gap-1 sm:flex">
          {links.map(({ href, labelKey }) => {
            const isActive =
              href === '/' ? pathname === '/' : pathname.startsWith(href);

            return (
              <Link
                key={href}
                href={href}
                className={`
                  rounded-lg px-4 py-2 text-sm font-medium transition-colors
                  ${
                    isActive
                      ? 'bg-smoke text-ink'
                      : 'text-coal hover:bg-smoke hover:text-ink'
                  }
                `}
              >
                {t(`nav.${labelKey}`)}
              </Link>
            );
          })}
        </nav>

        <div className="flex items-center gap-2">
          {searchOpen ? (
            <form
              className="flex items-center gap-2"
              onSubmit={(event) => {
                event.preventDefault();
                submitSearch();
              }}
            >
              <input
                autoFocus
                type="search"
                value={term}
                onChange={(event) => setTerm(event.target.value)}
                placeholder={t('searchPlaceholder')}
                aria-label={t('search')}
                className="
                  h-10 w-44 rounded-lg border border-line bg-paper px-3 text-sm
                  text-ink outline-none transition-all placeholder:text-ash
                  focus:border-ink focus:ring-2 focus:ring-ink/10 sm:w-56
                "
              />
              <button
                type="submit"
                aria-label={t('search')}
                className="flex h-10 w-10 items-center justify-center rounded-xl bg-ink text-paper transition-colors hover:bg-coal"
              >
                <Search size={18} />
              </button>
            </form>
          ) : (
            <button
              type="button"
              aria-label={t('search')}
              onClick={() => setSearchOpen(true)}
              className="flex h-10 w-10 items-center justify-center rounded-xl text-coal transition-colors hover:bg-smoke hover:text-ink"
            >
              <Search size={20} />
            </button>
          )}

          <Link
            href="/cart"
            aria-label={t('cart')}
            className="relative flex h-10 w-10 items-center justify-center rounded-xl text-coal transition-colors hover:bg-smoke hover:text-ink"
          >
            <ShoppingBag size={20} />
            {itemCount > 0 && (
              <span className="absolute -top-0.5 -left-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-brand text-[10px] font-bold text-ink">
                {formatNumber(itemCount, locale)}
              </span>
            )}
          </Link>
        </div>
      </div>
    </header>
  );
}
