'use client';

import { useCallback, useState } from 'react';

import { useAppDispatch, useAppSelector } from '@/src/store/hooks';
import { submitUpdateCartItem } from '@/src/store/cart.slice';
import type { CartMutationError } from '@/src/store/cart.slice';

/**
 * useUpdateCartItem — PATCH /api/v1/cart/items/{id} via the cart
 * slice; the backend's recomputed cart replaces the stored one.
 */
export function useUpdateCartItem() {
  const dispatch = useAppDispatch();
  const { mutationPending, mutationError: sliceError } = useAppSelector(
    (state) => state.cart,
  );
  const [error, setError] = useState<CartMutationError | null>(null);

  const updateCartItem = useCallback(
    async (lineId: number, quantity: number) => {
      setError(null);
      try {
        const action = await dispatch(
          submitUpdateCartItem({ lineId, quantity }),
        );
        if (submitUpdateCartItem.fulfilled.match(action)) {
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

  return { updateCartItem, pending: mutationPending, error: error ?? sliceError };
}
