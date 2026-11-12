'use client';

import { useCallback, useState } from 'react';

import { useAppDispatch, useAppSelector } from '@/src/store/hooks';
import { submitAddToCart } from '@/src/store/cart.slice';
import type { CartMutationError } from '@/src/store/cart.slice';

/**
 * useAddToCart — POST /api/v1/cart/items via the cart slice thunk.
 * The thunk stores the backend's recomputed cart, keeping the header
 * badge and any open cart view in sync. Returns the updated cart
 * (null when the mutation failed).
 */
export function useAddToCart() {
  const dispatch = useAppDispatch();
  const { mutationPending, mutationError: sliceError } = useAppSelector(
    (state) => state.cart,
  );
  const [error, setError] = useState<CartMutationError | null>(null);

  const addToCart = useCallback(
    async (variantId: number, quantity: number) => {
      setError(null);
      try {
        const action = await dispatch(submitAddToCart({ variantId, quantity }));
        if (submitAddToCart.fulfilled.match(action)) {
          return action.payload;
        }
        setError(action.payload ?? null);
        return null;
      } catch {
        setError({ status: 0, code: 'UNKNOWN', message: 'Unexpected error' });
        return null;
      }
    },
    [dispatch],
  );

  return { addToCart, pending: mutationPending, error: error ?? sliceError };
}
