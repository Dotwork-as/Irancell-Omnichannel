'use client';

import { useCallback, useState } from 'react';

import { useAppDispatch, useAppSelector } from '@/src/store/hooks';
import { submitRemoveCartItem } from '@/src/store/cart.slice';
import type { CartMutationError } from '@/src/store/cart.slice';

/**
 * useRemoveCartItem — DELETE /api/v1/cart/items/{id} via the cart
 * slice.
 */
export function useRemoveCartItem() {
  const dispatch = useAppDispatch();
  const { mutationPending, mutationError: sliceError } = useAppSelector(
    (state) => state.cart,
  );
  const [error, setError] = useState<CartMutationError | null>(null);

  const removeCartItem = useCallback(
    async (lineId: number) => {
      setError(null);
      try {
        const action = await dispatch(submitRemoveCartItem({ lineId }));
        if (submitRemoveCartItem.fulfilled.match(action)) {
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

  return { removeCartItem, pending: mutationPending, error: error ?? sliceError };
}
