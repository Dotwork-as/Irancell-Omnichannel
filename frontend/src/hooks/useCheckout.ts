'use client';

import { useCallback, useState } from 'react';

import { useAppDispatch } from '@/src/store/hooks';
import { fetchCart } from '@/src/store/cart.slice';
import { checkout } from '@/src/lib/api/services/checkout';
import { normalizeApiError } from '@/src/lib/api/error';
import type { ApiError } from '@/src/lib/api/error';
import type { CheckoutRequest, Receipt } from '@/src/lib/api/types/schemas';

interface CheckoutState {
  receipt: Receipt | null;
  pending: boolean;
  error: ApiError | null;
}

const RECEIPT_STORAGE_KEY = 'last_receipt';

/**
 * useCheckout — POST /api/v1/checkout. On 201 the paid receipt is
 * stored (sessionStorage) for the success page and the cart slice is
 * refetched — the backend emptied it, and the header badge must
 * follow (mutation → affected-state sync).
 */
export function useCheckout() {
  const dispatch = useAppDispatch();
  const [state, setState] = useState<CheckoutState>({
    receipt: null,
    pending: false,
    error: null,
  });

  const submitCheckout = useCallback(
    async (request: CheckoutRequest): Promise<Receipt | null> => {
      setState({ receipt: null, pending: true, error: null });
      try {
        const receipt = await checkout(request);
        try {
          sessionStorage.setItem(RECEIPT_STORAGE_KEY, JSON.stringify(receipt));
        } catch {
          // Storage may be unavailable (private mode) — the receipt is
          // still valid; the success page falls back gracefully.
        }
        void dispatch(fetchCart());
        setState({ receipt, pending: false, error: null });
        return receipt;
      } catch (thrown) {
        const apiError = normalizeApiError(thrown);
        setState({ receipt: null, pending: false, error: apiError });
        return null;
      }
    },
    [dispatch],
  );

  return { ...state, submitCheckout };
}

/** Reads the receipt stored by the last successful checkout. */
export function getStoredReceipt(): Receipt | null {
  try {
    const raw = sessionStorage.getItem(RECEIPT_STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as Receipt;
  } catch {
    return null;
  }
}
