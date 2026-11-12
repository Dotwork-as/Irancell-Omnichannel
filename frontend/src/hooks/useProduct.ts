'use client';

import { useEffect, useState } from 'react';

import { getProduct } from '@/src/lib/api/services/catalog';
import { normalizeApiError } from '@/src/lib/api/error';
import type { ApiError } from '@/src/lib/api/error';
import type { ProductDetail } from '@/src/lib/api/types/schemas';

interface ProductState {
  product: ProductDetail | null;
  /** True when the backend answered 404 NOT_FOUND for this slug. */
  notFound: boolean;
  error: ApiError | null;
}

/**
 * useProduct — GET /api/v1/products/{slug}; distinguishes "unknown
 * product" (notFound) from transport failures. Loading is derived
 * (no product, no error, not notFound) so effects stay setState-free.
 */
export function useProduct(slug: string) {
  const [state, setState] = useState<ProductState>({
    product: null,
    notFound: false,
    error: null,
  });
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let stale = false;
    getProduct(slug)
      .then((product) => {
        if (!stale) setState({ product, notFound: false, error: null });
      })
      .catch((thrown) => {
        if (stale) return;
        const apiError = normalizeApiError(thrown);
        setState({
          product: null,
          notFound: apiError.status === 404,
          error: apiError.status === 404 ? null : apiError,
        });
      });

    return () => {
      stale = true;
    };
  }, [slug, tick]);

  const loading = state.product === null && state.error === null && !state.notFound;
  return {
    ...state,
    loading,
    refetch: () => setTick((t) => t + 1),
  };
}
