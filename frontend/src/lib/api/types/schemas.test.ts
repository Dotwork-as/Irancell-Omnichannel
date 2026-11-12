import { describe, expect, it } from 'vitest';

import {
  addCartItemRequestSchema,
  cartSchema,
  checkoutRequestSchema,
  errorEnvelopeSchema,
  productDetailSchema,
  productListSchema,
  receiptSchema,
  shippingMethodListSchema,
} from './schemas';

const summary = {
  slug: 'iphone-15-pro-max',
  name: 'آیفون ۱۵ پرو مکس',
  defaultVariantId: 2,
  colorKey: 'natural',
  colorName: 'تیتانیوم طبیعی',
  storageGb: 256,
  priceIrr: 489900000,
  compareAtPriceIrr: 525000000,
  inStock: true,
  rating: 4.8,
  reviewCount: 246,
};

describe('productListSchema', () => {
  it('accepts the backend list envelope', () => {
    const parsed = productListSchema.parse({ items: [summary] });
    expect(parsed.items).toHaveLength(1);
  });

  it('accepts null compareAtPriceIrr', () => {
    expect(
      productListSchema.parse({ items: [{ ...summary, compareAtPriceIrr: null }] }),
    ).toBeDefined();
  });

  it('rejects float prices (contract: integer IRR)', () => {
    expect(() =>
      productListSchema.parse({ items: [{ ...summary, priceIrr: 1.5 }] }),
    ).toThrow();
  });

  it('rejects missing required fields', () => {
    const broken = { ...summary };
    delete (broken as Record<string, unknown>).reviewCount;
    expect(() => productListSchema.parse({ items: [broken] })).toThrow();
  });

  it('rejects a non-envelope response (e.g. a bare array)', () => {
    expect(() => productListSchema.parse([summary])).toThrow();
  });
});

describe('productDetailSchema', () => {
  const detail = {
    slug: 'iphone-15-pro-max',
    name: 'آیفون ۱۵ پرو مکس',
    defaultVariantId: 2,
    rating: 4.8,
    reviewCount: 246,
    specs: [{ label: 'تراشه', value: 'Apple A17 Pro' }],
    reviews: [{ author: 'سارا محمدی', rating: 5, text: '…' }],
    variants: [
      {
        id: 2,
        sku: 'IP15PM-NAT-256',
        colorKey: 'natural',
        colorName: 'تیتانیوم طبیعی',
        storageGb: 256,
        priceIrr: 489900000,
        compareAtPriceIrr: null,
        stockQty: 7,
        inStock: true,
      },
    ],
  };

  it('accepts a full detail', () => {
    expect(productDetailSchema.parse(detail)).toBeDefined();
  });

  it('rejects a review rating outside 1–5', () => {
    const broken = {
      ...detail,
      reviews: [{ author: 'x', rating: 6, text: '…' }],
    };
    expect(() => productDetailSchema.parse(broken)).toThrow();
  });
});

describe('cartSchema', () => {
  it('requires the IRR currency literal', () => {
    const cart = {
      items: [],
      subtotalIrr: 0,
      itemCount: 0,
      currency: 'USD',
    };
    expect(() => cartSchema.parse(cart)).toThrow();
  });

  it('accepts an empty cart and enforces line quantity ≥ 1', () => {
    expect(cartSchema.parse({ items: [], subtotalIrr: 0, itemCount: 0, currency: 'IRR' })).toBeDefined();
    const withLine = {
      items: [
        {
          id: 1,
          variantId: 2,
          productSlug: 'iphone-15-pro-max',
          productName: '…',
          colorKey: 'natural',
          colorName: '…',
          storageGb: 256,
          unitPriceIrr: 489900000,
          quantity: 0,
          lineTotalIrr: 0,
          maxQuantity: 7,
        },
      ],
      subtotalIrr: 0,
      itemCount: 0,
      currency: 'IRR',
    };
    expect(() => cartSchema.parse(withLine)).toThrow();
  });
});

describe('shippingMethodListSchema + receiptSchema', () => {
  it('parses shipping methods', () => {
    expect(
      shippingMethodListSchema.parse({
        items: [
          { id: 'express', name: 'پیک اکسپرس تهران', description: '…', priceIrr: 490000 },
          { id: 'post', name: 'پست پیشتاز', description: '…', priceIrr: 0 },
        ],
      }),
    ).toBeDefined();
  });

  it('rejects an unknown order status on the receipt', () => {
    const receipt = {
      trackingCode: 'IRC-2X9Q3YAE',
      status: 'queued',
      recipient: {
        firstName: 'نرگس',
        lastName: 'رضایی',
        phone: '09123456789',
        address: 'تهران، خیابان نمونه، پلاک ۱۰',
      },
      shippingMethod: { id: 'express', name: 'ارسال فوری', priceIrr: 490000 },
      items: [],
      subtotalIrr: 0,
      shippingIrr: 0,
      totalIrr: 0,
      currency: 'IRR',
      payment: { method: 'mock', transactionId: 'TXN-x' },
      createdAt: '2026-11-08T14:08:30Z',
      paidAt: '2026-11-08T14:08:31Z',
    };
    expect(() => receiptSchema.parse(receipt)).toThrow();
  });
});

describe('request schemas', () => {
  it('bounds cart quantities to 1–99', () => {
    expect(() => addCartItemRequestSchema.parse({ variantId: 2, quantity: 0 })).toThrow();
    expect(() => addCartItemRequestSchema.parse({ variantId: 2, quantity: 100 })).toThrow();
    expect(addCartItemRequestSchema.parse({ variantId: 2, quantity: 99 })).toBeDefined();
  });

  it('enforces the Iranian mobile pattern on checkout', () => {
    const base = {
      recipient: {
        firstName: 'نرگس',
        lastName: 'رضایی',
        phone: '09123456789',
        address: 'تهران، خیابان نمونه، پلاک ۱۰',
      },
      shippingMethodId: 'express',
    };
    expect(checkoutRequestSchema.parse(base)).toBeDefined();
    expect(() =>
      checkoutRequestSchema.parse({
        ...base,
        recipient: { ...base.recipient, phone: '09123' },
      }),
    ).toThrow();
  });
});

describe('errorEnvelopeSchema', () => {
  it('parses the §4.3 envelope with validation details', () => {
    expect(
      errorEnvelopeSchema.parse({
        error: {
          code: 'VALIDATION_ERROR',
          message: 'درخواست نامعتبر است.',
          details: [
            { field: 'recipient.phone', code: 'INVALID_PHONE', message: '…' },
          ],
        },
      }),
    ).toBeDefined();
  });

  it('accepts stock-conflict details (free-form objects)', () => {
    expect(
      errorEnvelopeSchema.parse({
        error: {
          code: 'STOCK_UNAVAILABLE',
          message: 'موجودی کافی نیست.',
          details: [{ variantId: 2, productName: '…', requested: 10, available: 7 }],
        },
      }),
    ).toBeDefined();
  });

  it('parses an envelope without details', () => {
    expect(
      errorEnvelopeSchema.parse({ error: { code: 'NOT_FOUND', message: '…' } }),
    ).toBeDefined();
  });
});
