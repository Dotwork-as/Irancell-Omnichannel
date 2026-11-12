import { describe, expect, it } from 'vitest';

import { ApiError, normalizeApiError } from './error';

describe('normalizeApiError', () => {
  it('maps a §4.3 envelope rejection to a typed ApiError', () => {
    const err = normalizeApiError({
      status: 409,
      data: {
        error: {
          code: 'STOCK_UNAVAILABLE',
          message: 'موجودی کافی نیست.',
          details: [{ variantId: 2, productName: '…', requested: 10, available: 7 }],
        },
      },
      message: 'Request failed with status code 409',
    });
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(409);
    expect(err.code).toBe('STOCK_UNAVAILABLE');
    expect(err.message).toBe('موجودی کافی نیست.');
    expect(err.validationDetails()).toHaveLength(0);
  });

  it('exposes validation details when the backend sends that shape', () => {
    const err = normalizeApiError({
      status: 400,
      data: {
        error: {
          code: 'VALIDATION_ERROR',
          message: 'درخواست نامعتبر است.',
          details: [{ field: 'recipient.phone', code: 'INVALID_PHONE', message: '…' }],
        },
      },
    });
    expect(err.validationDetails()).toEqual([
      { field: 'recipient.phone', code: 'INVALID_PHONE', message: '…' },
    ]);
  });

  it('maps a non-envelope rejection (e.g. 500 HTML) without leaking internals', () => {
    const err = normalizeApiError({ status: 500, data: '<html>boom</html>' });
    expect(err.status).toBe(500);
    expect(err.message).not.toContain('<html>');
    expect(err.message).not.toContain('boom');
  });

  it('maps network failures (status 0)', () => {
    const err = normalizeApiError({ status: 0, message: 'No response from server' });
    expect(err.status).toBe(0);
    expect(err.code).toBe('NETWORK_ERROR');
  });

  it('passes an existing ApiError through untouched', () => {
    const original = new ApiError(404, 'NOT_FOUND', 'منبع درخواست‌شده پیدا نشد.');
    expect(normalizeApiError(original)).toBe(original);
  });

  it('never crashes on undefined', () => {
    expect(normalizeApiError(undefined).status).toBe(0);
  });
});
