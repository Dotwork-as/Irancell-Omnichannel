import { z } from 'zod';

import api from '@/src/lib/api/axios';
import { normalizeApiError } from '@/src/lib/api/error';
import {
  productListSchema,
  productListQuerySchema,
  productDetailSchema,
  type ProductListQuery,
  type ProductSummary,
  type ProductDetail,
} from '@/src/lib/api/types/schemas';

/** GET /api/v1/products?q=… — global search, unavailable included. */
export async function listProducts(
  query: ProductListQuery = {},
): Promise<ProductSummary[]> {
  const parsed = productListQuerySchema.parse(query);
  try {
    const response = await api.get('/products', {
      params: parsed.q ? { q: parsed.q } : undefined,
    });
    return productListSchema.parse(response.data).items;
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}

/** GET /api/v1/products/{slug} — full localized detail. */
export async function getProduct(slug: string): Promise<ProductDetail> {
  const slugSchema = z.string().min(1).max(120);
  const safeSlug = slugSchema.parse(slug);
  try {
    const response = await api.get(`/products/${encodeURIComponent(safeSlug)}`);
    return productDetailSchema.parse(response.data);
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}
