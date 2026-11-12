'use client';

import { useEffect } from 'react';

import { useAppDispatch, useAppSelector } from '@/src/store/hooks';
import { fetchCart } from '@/src/store/cart.slice';

/**
 * useCart — the session cart from the shared Redux slice (feeds the
 * header badge everywhere). Fetches once on mount when the slice is
 * still idle; mutations elsewhere keep it synchronized.
 */
export function useCart() {
  const dispatch = useAppDispatch();
  const { cart, status, mutationPending, error } = useAppSelector(
    (state) => state.cart,
  );

  useEffect(() => {
    if (status === 'idle') {
      void dispatch(fetchCart());
    }
  }, [dispatch, status]);

  return {
    cart,
    loading: status === 'loading',
    mutationPending,
    error,
    refetch: () => void dispatch(fetchCart()),
  };
}
