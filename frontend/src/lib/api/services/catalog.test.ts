import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError, normalizeApiError } from '../error';
import * as catalog from './catalog';

vi.mock('@/src/lib/api/axios', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

import api from '@/src/lib/api/axios';

const mockedGet = vi.mocked(api.get);

const listBody = {
  items: [
    {
      slug: 'iphone-se-3',
      name: 'آیفون SE نسل سوم',
      defaultVariantId: 20,
      colorKey: 'white',
      colorName: 'سفید ستاره‌ای',
      storageGb: 128,
      priceIrr: 198000000,
      compareAtPriceIrr: null,
      inStock: false,
      rating: 4,
      reviewCount: 64,
    },
  ],
};

describe('catalog service', () => {
  beforeEach(() => {
    mockedGet.mockReset();
  });

  it('sends q as a query parameter and validates the response', async () => {
    mockedGet.mockResolvedValueOnce({ data: listBody });
    const items = await catalog.listProducts({ q: 'SE' });
    expect(items[0].inStock).toBe(false);
    expect(mockedGet).toHaveBeenCalledWith('/products', { params: { q: 'SE' } });
  });

  it('omits the q parameter when empty', async () => {
    mockedGet.mockResolvedValueOnce({ data: { items: [] } });
    await catalog.listProducts();
    expect(mockedGet).toHaveBeenCalledWith('/products', { params: undefined });
  });

  it('raises a normalized ApiError on a malformed response', async () => {
    mockedGet.mockResolvedValueOnce({ data: { nope: true } });
    const err = await catalog.listProducts().catch(normalizeApiError);
    expect(err).toBeInstanceOf(ApiError);
  });
});

describe('getProduct', () => {
  it('encodes the slug into the path', async () => {
    mockedGet.mockResolvedValueOnce({
      data: {
        slug: 'iphone-15-pro-max',
        name: '…',
        defaultVariantId: 1,
        rating: 5,
        reviewCount: 1,
        specs: [],
        reviews: [],
        variants: [],
      },
    });
    await catalog.getProduct('iphone-15-pro-max');
    expect(mockedGet).toHaveBeenCalledWith('/products/iphone-15-pro-max');
  });

  it('normalizes backend error envelopes into ApiError', async () => {
    mockedGet.mockRejectedValueOnce({
      status: 404,
      data: { error: { code: 'NOT_FOUND', message: 'منبع درخواست‌شده پیدا نشد.' } },
    });
    const err = await catalog.getProduct('nope').catch(normalizeApiError);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe('NOT_FOUND');
    expect((err as ApiError).status).toBe(404);
  });
});
