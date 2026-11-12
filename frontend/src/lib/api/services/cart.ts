import { z } from 'zod';

import api from '@/src/lib/api/axios';
import { normalizeApiError } from '@/src/lib/api/error';
import {
  addCartItemRequestSchema,
  cartSchema,
  updateCartItemRequestSchema,
  type AddCartItemRequest,
  type Cart,
  type UpdateCartItemRequest,
} from '@/src/lib/api/types/schemas';

/** GET /api/v1/cart — session cart (empty when no valid session). */
export async function getCart(): Promise<Cart> {
  try {
    const response = await api.get('/cart');
    return cartSchema.parse(response.data);
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}

/**
 * POST /api/v1/cart/items — add a variant (or increment an existing
 * line). The backend returns the updated cart; the session cookie is
 * set by the backend on the first successful write.
 */
export async function addToCart(request: AddCartItemRequest): Promise<Cart> {
  const body = addCartItemRequestSchema.parse(request);
  try {
    const response = await api.post('/cart/items', body);
    return cartSchema.parse(response.data);
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}

/** PATCH /api/v1/cart/items/{id} — set a line quantity (1–99). */
export async function updateCartItem(
  lineId: number,
  request: UpdateCartItemRequest,
): Promise<Cart> {
  const id = z.number().int().positive().parse(lineId);
  const body = updateCartItemRequestSchema.parse(request);
  try {
    const response = await api.patch(`/cart/items/${id}`, body);
    return cartSchema.parse(response.data);
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}

/** DELETE /api/v1/cart/items/{id} — remove a line. */
export async function removeCartItem(lineId: number): Promise<Cart> {
  const id = z.number().int().positive().parse(lineId);
  try {
    const response = await api.delete(`/cart/items/${id}`);
    return cartSchema.parse(response.data);
  } catch (thrown) {
    throw normalizeApiError(thrown);
  }
}
