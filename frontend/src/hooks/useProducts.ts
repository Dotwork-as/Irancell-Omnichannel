'use client';

import { useEffect, useState } from 'react';

import { listProducts } from '@/src/lib/api/services/catalog';
import { normalizeApiError } from '@/src/lib/api/error';
import type { ApiError } from '@/src/lib/api/error';
import type { ProductSummary } from '@/src/lib/api/types/schemas';

interface ProductsState {
  /** null = still loading (derived, so effects stay setState-free). */
  products: ProductSummary[] | null;
  error: ApiError | null;
}

/**
 * useProducts — GET /api/v1/products?q=… Global search with
 * unavailable products included. Re-runs when `q` changes; the stale
 * flag guards against out-of-order responses.
 */
export function useProducts(query?: string) {
  const [state, setState] = useState<ProductsState>({ products: null, error: null });
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let stale = false;
    listProducts(query ? { q: query } : {})
      .then((products) => {
        if (!stale) setState({ products, error: null });
      })
      .catch((thrown) => {
        if (!stale) setState({ products: [], error: normalizeApiError(thrown) });
      });

    return () => {
      stale = true;
    };
  }, [query, tick]);

  const loading = state.products === null && state.error === null;
  return { products: state.products ?? [], loading, error: state.error, refetch: () => setTick((t) => t + 1) };
}
