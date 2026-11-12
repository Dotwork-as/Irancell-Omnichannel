import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import CatalogView from '@/src/components/shop/catalogView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('metadata');

  return { title: t('products.title'), description: t('products.description') };
}

export default async function ProductsPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string }>;
}) {
  const { q } = await searchParams;

  return <CatalogView q={q} />;
}
