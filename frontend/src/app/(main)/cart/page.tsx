import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import CartView from '@/src/components/shop/cartView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('metadata');

  return { title: t('cart.title') };
}

export default function CartPage() {
  return <CartView />;
}
