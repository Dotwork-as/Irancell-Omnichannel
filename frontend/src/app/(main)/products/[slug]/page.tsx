import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import ProductDetailView from '@/src/components/shop/productDetailView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('metadata');

  return {
    title: t('product.title'),
    description: t('product.description'),
  };
}

export default async function ProductDetailPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  return <ProductDetailView slug={slug} />;
}
