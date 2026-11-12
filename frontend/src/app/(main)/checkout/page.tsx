import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import CheckoutFlow from '@/src/components/shop/checkoutFlow';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('metadata');

  return { title: t('checkout.title') };
}

export default function CheckoutPage() {
  return <CheckoutFlow />;
}
