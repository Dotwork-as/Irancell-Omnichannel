import { renderHook, act } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';

import { Provider } from 'react-redux';

import { useCheckout } from './useCheckout';
import { makeStore } from '@/src/store/store';
import * as checkoutService from '@/src/lib/api/services/checkout';
import * as cartService from '@/src/lib/api/services/cart';
import type { Cart, Receipt } from '@/src/lib/api/types/schemas';

vi.mock('@/src/lib/api/services/checkout');
vi.mock('@/src/lib/api/services/cart');

let testStore: ReturnType<typeof makeStore>;

function wrapper({ children }: { children: ReactNode }) {
  return <Provider store={testStore}>{children}</Provider>;
}

const receipt: Receipt = {
  trackingCode: 'IRC-2X9Q3YAE',
  status: 'paid',
  recipient: {
    firstName: 'نرگس',
    lastName: 'رضایی',
    phone: '09123456789',
    address: 'تهران، خیابان نمونه، پلاک ۱۰',
  },
  shippingMethod: { id: 'express', name: 'ارسال فوری', priceIrr: 490000 },
  items: [],
  subtotalIrr: 979800000,
  shippingIrr: 490000,
  totalIrr: 980290000,
  currency: 'IRR',
  payment: { method: 'mock', transactionId: 'TXN-abc123' },
  createdAt: '2026-11-08T14:08:30Z',
  paidAt: '2026-11-08T14:08:31Z',
};

const emptyCart: Cart = { items: [], subtotalIrr: 0, itemCount: 0, currency: 'IRR' };

describe('useCheckout', () => {
  beforeEach(() => {
    vi.mocked(checkoutService.checkout).mockReset();
    vi.mocked(cartService.getCart).mockReset();
    sessionStorage.clear();
    testStore = makeStore();
  });

  it('stores the paid receipt and re-syncs the emptied cart', async () => {
    vi.mocked(checkoutService.checkout).mockResolvedValueOnce(receipt);
    vi.mocked(cartService.getCart).mockResolvedValue(emptyCart);

    const { result } = renderHook(() => useCheckout(), { wrapper });
    let out: Receipt | null = null;
    await act(async () => {
      out = await result.current.submitCheckout({
        recipient: receipt.recipient,
        shippingMethodId: 'express',
      });
    });

    expect((out as Receipt | null)?.trackingCode).toBe('IRC-2X9Q3YAE');
    expect(result.current.error).toBeNull();

    // The cart slice was re-synced (backend emptied it).
    await waitForAsync();
    expect(testStore.getState().cart.cart?.itemCount).toBe(0);

    // The receipt is stored for the success page.
    expect(sessionStorage.getItem('last_receipt')).not.toBeNull();
  });

  it('surfaces a localized backend error and stores no receipt', async () => {
    vi.mocked(checkoutService.checkout).mockRejectedValueOnce({
      status: 409,
      data: { error: { code: 'CART_EMPTY', message: 'سبد خرید خالی است.' } },
    });

    const { result } = renderHook(() => useCheckout(), { wrapper });
    let out: Receipt | null = null;
    await act(async () => {
      out = await result.current.submitCheckout({
        recipient: receipt.recipient,
        shippingMethodId: 'post',
      });
    });

    expect(out).toBeNull();
    expect(result.current.error?.code).toBe('CART_EMPTY');
    expect(result.current.error?.message).toBe('سبد خرید خالی است.');
    expect(sessionStorage.getItem('last_receipt')).toBeNull();
  });

  it('maps server VALIDATION_ERROR details onto fields', async () => {
    vi.mocked(checkoutService.checkout).mockRejectedValueOnce({
      status: 400,
      data: {
        error: {
          code: 'VALIDATION_ERROR',
          message: 'درخواست نامعتبر است.',
          details: [
            { field: 'recipient.phone', code: 'INVALID_PHONE', message: '…' },
          ],
        },
      },
    });

    const { result } = renderHook(() => useCheckout(), { wrapper });
    await act(async () => {
      await result.current.submitCheckout({
        recipient: receipt.recipient,
        shippingMethodId: 'express',
      });
    });

    expect(result.current.error?.validationDetails()[0].field).toBe('recipient.phone');
  });
});

async function waitForAsync() {
  await new Promise((resolve) => setTimeout(resolve, 0));
}
