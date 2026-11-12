import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';

import { Provider } from 'react-redux';

import { useAddToCart } from './useAddToCart';
import { useCart } from './useCart';
import { makeStore } from '@/src/store/store';
import * as cartService from '@/src/lib/api/services/cart';
import type { Cart } from '@/src/lib/api/types/schemas';

vi.mock('@/src/lib/api/services/cart');

const cartAfterAdd: Cart = {
  items: [
    {
      id: 1,
      variantId: 2,
      productSlug: 'iphone-15-pro-max',
      productName: 'آیفون ۱۵ پرو مکس',
      colorKey: 'natural',
      colorName: 'تیتانیوم طبیعی',
      storageGb: 256,
      unitPriceIrr: 489900000,
      quantity: 2,
      lineTotalIrr: 979800000,
      maxQuantity: 7,
    },
  ],
  subtotalIrr: 979800000,
  itemCount: 2,
  currency: 'IRR',
};

let testStore: ReturnType<typeof makeStore>;

function wrapper({ children }: { children: ReactNode }) {
  return <Provider store={testStore}>{children}</Provider>;
}

describe('useAddToCart', () => {
  beforeEach(() => {
    vi.mocked(cartService.addToCart).mockReset();
    vi.mocked(cartService.getCart).mockReset();
    testStore = makeStore();
  });

  it('stores the backend cart and exposes the item count to useCart', async () => {
    vi.mocked(cartService.addToCart).mockResolvedValueOnce(cartAfterAdd);

    const { result } = renderHook(() => useAddToCart(), { wrapper });
    let cart: Cart | null = null;
    await act(async () => {
      cart = await result.current.addToCart(2, 2);
    });

    expect((cart as Cart | null)?.itemCount).toBe(2);
    expect(result.current.error).toBeNull();

    // The slice now holds the cart — useCart sees the badge count.
    const { result: cartResult } = renderHook(() => useCart(), { wrapper });
    await waitFor(() => {
      expect(cartResult.current.cart?.itemCount).toBe(2);
    });
  });

  it('surfaces the backend mutation error without updating state', async () => {
    vi.mocked(cartService.addToCart).mockRejectedValueOnce({
      status: 409,
      data: {
        error: {
          code: 'STOCK_UNAVAILABLE',
          message: 'موجودی کافی نیست.',
          details: [{ variantId: 2, productName: '…', requested: 10, available: 7 }],
        },
      },
    });

    const { result } = renderHook(() => useAddToCart(), { wrapper });
    let cart: Cart | null = null;
    await act(async () => {
      cart = await result.current.addToCart(2, 10);
    });

    expect(cart).toBeNull();
    expect(result.current.error?.code).toBe('STOCK_UNAVAILABLE');
    expect(result.current.error?.status).toBe(409);
  });

  it('requests the cart only once when idle (no duplicate fetches)', async () => {
    vi.mocked(cartService.getCart).mockResolvedValue({
      items: [],
      subtotalIrr: 0,
      itemCount: 0,
      currency: 'IRR',
    });

    const { rerender } = renderHook(() => useCart(), { wrapper });
    await waitFor(() => {
      expect(cartService.getCart).toHaveBeenCalledTimes(1);
    });
    rerender();
    rerender();
    expect(cartService.getCart).toHaveBeenCalledTimes(1);
  });
});
