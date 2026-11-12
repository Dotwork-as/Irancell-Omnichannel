'use client';

import { useEffect, useState } from 'react';

import { getShippingMethods } from '@/src/lib/api/services/checkout';
import { normalizeApiError } from '@/src/lib/api/error';
import type { ApiError } from '@/src/lib/api/error';
import type { ShippingMethod } from '@/src/lib/api/types/schemas';

interface ShippingState {
  methods: ShippingMethod[] | null;
  error: ApiError | null;
}

/**
 * useShippingMethods — GET /api/v1/shipping-methods; fetches once on
 * mount (the set is static within a session).
 */
export function useShippingMethods() {
  const [state, setState] = useState<ShippingState>({ methods: null, error: null });
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let stale = false;
    getShippingMethods()
      .then((methods) => {
        if (!stale) setState({ methods, error: null });
      })
      .catch((thrown) => {
        if (!stale) setState({ methods: [], error: normalizeApiError(thrown) });
      });

    return () => {
      stale = true;
    };
  }, [tick]);

  const loading = state.methods === null && state.error === null;
  return { methods: state.methods ?? [], loading, error: state.error, refetch: () => setTick((t) => t + 1) };
}
