import { z } from 'zod';

/**
 * Zod schemas mirroring the finalized backend contract (docs/API.md §4)
 * — the backend is the source of truth; every API response is
 * validated at runtime before it reaches a component. Types are
 * derived with z.infer, never re-declared by hand.
 */

// ─── Shared primitives ───────────────────────────────────────────

/** Integer rial prices — never floats, never formatted strings. */
const irrInt = z.number().int();

const nullableCompareAt = irrInt.nullable();

// ─── §4.3 error envelope ─────────────────────────────────────────

export const validationDetailSchema = z.object({
  field: z.string(),
  code: z.string(),
  message: z.string(),
});

export const errorEnvelopeSchema = z.object({
  error: z.object({
    code: z.string(),
    message: z.string(),
    details: z
      .union([z.array(validationDetailSchema), z.array(z.record(z.string(), z.unknown()))])
      .optional(),
  }),
});

export type ValidationDetail = z.infer<typeof validationDetailSchema>;
export type ErrorEnvelope = z.infer<typeof errorEnvelopeSchema>;

// ─── §4.4 catalog ────────────────────────────────────────────────

export const variantSchema = z.object({
  id: z.number().int(),
  sku: z.string(),
  colorKey: z.string(),
  colorName: z.string(),
  storageGb: z.number().int().positive(),
  priceIrr: irrInt.nonnegative(),
  compareAtPriceIrr: nullableCompareAt,
  stockQty: z.number().int().nonnegative(),
  inStock: z.boolean(),
});

export const specSchema = z.object({
  label: z.string(),
  value: z.string(),
});

export const reviewSchema = z.object({
  author: z.string(),
  rating: z.number().int().min(1).max(5),
  text: z.string(),
});

export const productSummarySchema = z.object({
  slug: z.string(),
  name: z.string(),
  defaultVariantId: z.number().int(),
  colorKey: z.string(),
  colorName: z.string(),
  storageGb: z.number().int().positive(),
  priceIrr: irrInt.nonnegative(),
  compareAtPriceIrr: nullableCompareAt,
  inStock: z.boolean(),
  rating: z.number().min(0).max(5),
  reviewCount: z.number().int().nonnegative(),
});

export const productDetailSchema = productSummarySchema.omit({
  colorKey: true,
  colorName: true,
  storageGb: true,
  priceIrr: true,
  compareAtPriceIrr: true,
  inStock: true,
}).extend({
  specs: z.array(specSchema),
  reviews: z.array(reviewSchema),
  variants: z.array(variantSchema),
});

export const productListSchema = z.object({
  items: z.array(productSummarySchema),
});

export type Variant = z.infer<typeof variantSchema>;
export type Spec = z.infer<typeof specSchema>;
export type Review = z.infer<typeof reviewSchema>;
export type ProductSummary = z.infer<typeof productSummarySchema>;
export type ProductDetail = z.infer<typeof productDetailSchema>;

// Request: GET /products query string.
export const productListQuerySchema = z.object({
  q: z.string().trim().max(100).optional(),
});
export type ProductListQuery = z.infer<typeof productListQuerySchema>;

// ─── §4.4 cart ───────────────────────────────────────────────────

export const cartLineSchema = z.object({
  id: z.number().int(),
  variantId: z.number().int(),
  productSlug: z.string(),
  productName: z.string(),
  colorKey: z.string(),
  colorName: z.string(),
  storageGb: z.number().int().positive(),
  unitPriceIrr: irrInt.nonnegative(),
  quantity: z.number().int().min(1),
  lineTotalIrr: irrInt.nonnegative(),
  maxQuantity: z.number().int().nonnegative(),
});

export const cartSchema = z.object({
  items: z.array(cartLineSchema),
  subtotalIrr: irrInt.nonnegative(),
  itemCount: z.number().int().nonnegative(),
  currency: z.literal('IRR'),
});

export type CartLine = z.infer<typeof cartLineSchema>;
export type Cart = z.infer<typeof cartSchema>;

// Request bodies.
export const addCartItemRequestSchema = z.object({
  variantId: z.number().int().positive(),
  quantity: z.number().int().min(1).max(99),
});
export const updateCartItemRequestSchema = z.object({
  quantity: z.number().int().min(1).max(99),
});

export type AddCartItemRequest = z.infer<typeof addCartItemRequestSchema>;
export type UpdateCartItemRequest = z.infer<typeof updateCartItemRequestSchema>;

// ─── §4.4 shipping ───────────────────────────────────────────────

export const shippingMethodSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  priceIrr: irrInt.nonnegative(),
});

export const shippingMethodListSchema = z.object({
  items: z.array(shippingMethodSchema),
});

export type ShippingMethod = z.infer<typeof shippingMethodSchema>;

// ─── §4.4 checkout ───────────────────────────────────────────────

export const recipientSchema = z.object({
  firstName: z.string().trim().min(2).max(60),
  lastName: z.string().trim().min(2).max(60),
  phone: z.string().regex(/^09\d{9}$/),
  address: z.string().trim().min(10).max(500),
});

export const checkoutRequestSchema = z.object({
  recipient: recipientSchema,
  shippingMethodId: z.string().trim().min(1),
  /** Cloudflare Turnstile token — required by the backend ONLY when
   * human verification is configured server-side; otherwise absent. */
  turnstileToken: z.string().max(2048).optional(),
});

export type Recipient = z.infer<typeof recipientSchema>;
export type CheckoutRequest = z.infer<typeof checkoutRequestSchema>;

export const receiptItemSchema = z.object({
  productName: z.string(),
  colorName: z.string(),
  storageGb: z.number().int().positive(),
  unitPriceIrr: irrInt.nonnegative(),
  quantity: z.number().int().min(1),
  lineTotalIrr: irrInt.nonnegative(),
});

export const receiptSchema = z.object({
  trackingCode: z.string(),
  status: z.enum([
    'pending',
    'paid',
    'processing',
    'shipped',
    'delivered',
    'cancelled',
    'failed',
  ]),
  recipient: recipientSchema,
  shippingMethod: z.object({
    id: z.string(),
    name: z.string(),
    priceIrr: irrInt.nonnegative(),
  }),
  items: z.array(receiptItemSchema),
  subtotalIrr: irrInt.nonnegative(),
  shippingIrr: irrInt.nonnegative(),
  totalIrr: irrInt.nonnegative(),
  currency: z.literal('IRR'),
  payment: z.object({
    method: z.string(),
    transactionId: z.string(),
  }),
  createdAt: z.string(),
  paidAt: z.string(),
});

export type ReceiptItem = z.infer<typeof receiptItemSchema>;
export type Receipt = z.infer<typeof receiptSchema>;
