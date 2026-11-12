import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import SuccessView from '@/src/components/shop/successView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('metadata');

  return { title: t('success.title') };
}

export default function CheckoutSuccessPage() {
  return <SuccessView />;
}
