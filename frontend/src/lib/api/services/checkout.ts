import api from '@/src/lib/api/axios';
import { normalizeApiError } from '@/src/lib/api/error';
import {
  checkoutRequestSchema,
  receiptSchema,
  shippingMethodListSchema,
  type CheckoutRequest,
  type Receipt,
  type ShippingMethod,
} from '@/src/lib/api/types/schemas';

/** GET /api/v1/shipping-methods — active methods, localized. */
export async function getShippingMethods(): Promise<ShippingMethod[]> {
  try {
    const response = await api.get('/shipping-methods');
    return shippingMethodListSchema.parse(response.data).items;
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}

/**
 * POST /api/v1/checkout — §4.6 transactional checkout; 201 returns
 * the paid order receipt, every failure raises a normalized ApiError
 * (the backend rolled back).
 */
export async function checkout(request: CheckoutRequest): Promise<Receipt> {
  const body = checkoutRequestSchema.parse(request);
  try {
    const response = await api.post('/checkout', body);
    return receiptSchema.parse(response.data);
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}
