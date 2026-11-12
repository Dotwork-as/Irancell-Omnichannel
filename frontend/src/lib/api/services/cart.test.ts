import { beforeEach, describe, expect, it, vi } from 'vitest';

import api from '@/src/lib/api/axios';
import { ApiError, normalizeApiError } from '../error';
import * as cart from './cart';

vi.mock('@/src/lib/api/axios', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockedPost = vi.mocked(api.post);
const mockedPatch = vi.mocked(api.patch);
const mockedDelete = vi.mocked(api.delete);
const mockedGet = vi.mocked(api.get);

const cartBody = {
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
      quantity: 3,
      lineTotalIrr: 1469700000,
      maxQuantity: 7,
    },
  ],
  subtotalIrr: 1469700000,
  itemCount: 3,
  currency: 'IRR',
};

describe('cart service', () => {
  beforeEach(() => {
    mockedGet.mockReset();
    mockedPost.mockReset();
    mockedPatch.mockReset();
    mockedDelete.mockReset();
  });

  it('posts the request body and returns the validated cart', async () => {
    mockedPost.mockResolvedValueOnce({ data: cartBody });
    const result = await cart.addToCart({ variantId: 2, quantity: 3 });
    expect(result.itemCount).toBe(3);
    expect(mockedPost).toHaveBeenCalledWith('/cart/items', {
      variantId: 2,
      quantity: 3,
    });
  });

  it('sends PATCH with the quantity body', async () => {
    mockedPatch.mockResolvedValueOnce({ data: cartBody });
    await cart.updateCartItem(1, { quantity: 5 });
    expect(mockedPatch).toHaveBeenCalledWith('/cart/items/1', { quantity: 5 });
  });

  it('sends DELETE for the line id', async () => {
    mockedDelete.mockResolvedValueOnce({ data: cartBody });
    await cart.removeCartItem(1);
    expect(mockedDelete).toHaveBeenCalledWith('/cart/items/1');
  });

  it('normalizes a §4.3 STOCK_UNAVAILABLE rejection with details', async () => {
    mockedPost.mockRejectedValueOnce({
      status: 409,
      data: {
        error: {
          code: 'STOCK_UNAVAILABLE',
          message: 'موجودی کافی نیست.',
          details: [{ variantId: 2, productName: '…', requested: 10, available: 7 }],
        },
      },
    });
    const err = await cart.addToCart({ variantId: 2, quantity: 5 }).catch(normalizeApiError);
    const apiError = err as ApiError;
    expect(apiError).toBeInstanceOf(ApiError);
    expect(apiError.code).toBe('STOCK_UNAVAILABLE');
    expect(apiError.validationDetails()).toHaveLength(0); // not validation-shaped
  });

  it('normalizes a 404 on an unknown cart line', async () => {
    mockedPatch.mockRejectedValueOnce({
      status: 404,
      data: { error: { code: 'NOT_FOUND', message: 'منبع درخواست‌شده پیدا نشد.' } },
    });
    const err = await cart.updateCartItem(999, { quantity: 1 }).catch(normalizeApiError);
    expect((err as ApiError).status).toBe(404);
  });
});
