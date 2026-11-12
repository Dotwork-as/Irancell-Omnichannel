# Irancell Omnichannel — REST API Contract

The single source of truth for the API served by the Go backend under
`/api/v1`. The frontend implements this contract twice: as Zod schemas
(`frontend/src/lib/api/types/schemas.ts`) and as Go types in the
backend. If the contract changes, this document changes first.

## Domain model

- **Product** — the item being sold; seeded data, never mutated via the API.
- **Variant** — a color/storage combination with its own price and stock.
- **GuestSession** — anonymous shopper identified by a secure HttpOnly
  cookie with a 30-day lifetime (no authentication in the MVP).
- **Cart** — belongs to one guest session; stores variant ids and
  quantities only, never prices.
- **ShippingMethod** — backend-owned delivery option with a price.
- **Checkout** — validates recipient + shipping, recalculates the cart
  from the session on the backend, mock-pays and atomically deducts
  stock.
- **Order** — the result of checkout, with a trackable status; seeded
  demo orders are flagged and always distinguishable from real orders.

## 1. General conventions

- Base URL: served by the backend under `/api/v1`; the frontend Axios
  client points at `NEXT_PUBLIC_API_BASE_URL`.
- All request bodies and responses are `application/json`.
- **Locale**: every response is single-locale, selected from the
  `Accept-Language` request header (supported: `fa`, `en`; missing or
  unsupported → `fa`). The backend localizes product names, color
  names, product specifications, ratings, review counts and review
  content, shipping method names/descriptions and all error messages.
  Static UI chrome copy stays in the frontend i18n catalogs.
- **Money**: integer rial — never a float, never a formatted string.
  Monetary fields are suffixed `Irr` (`priceIrr`, `totalIrr`, …) and
  cart/order summaries additionally carry `"currency": "IRR"`.
- **Timestamps**: RFC 3339 UTC strings (`2026-11-08T14:30:00Z`).
- **Identifiers**: products are addressed by `slug` (matching the
  frontend route `/products/{slug}`); variant, cart-line and order ids
  are opaque integers; the order `trackingCode` is the human handle.
- List endpoints always return `{ "items": [ … ] }`.
- **Security conventions:** every response carries
  `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`, a deny-all API CSP and a
  `Permissions-Policy`; the Next.js server applies the HTML policy
  (`script-src 'unsafe-inline'` for runtime inlining, `connect-src`
  includes the API origin). Requests are rate-limited per client IP
  (in-memory, single instance): 600/min API-wide (`RATE_LIMIT_MAX`),
  30/min checkout (`RATE_LIMIT_CHECKOUT_MAX`) — over-limit requests get
  `429 RATE_LIMITED` with `Retry-After`. Fiber runs with a 256KB body
  limit and 10s/15s/60s read/write/idle timeouts and never trusts
  `X-Forwarded-For` (empty trusted-proxy list). Checkout accepts an
  **optional** `turnstileToken` (Cloudflare Turnstile): enforced ONLY
  when `TURNSTILE_SECRET_KEY` is configured server-side, fail-closed
  (missing/invalid/expired/unreachable → `400 TURNSTILE_REQUIRED` /
  `403 TURNSTILE_FAILED`), disabled otherwise (local MVP behavior).
  Search terms are capped at 100 characters.

## 2. Endpoint inventory

| Method & path | Purpose | Success response |
| --- | --- | --- |
| `GET /api/v1/products?q=<term>` | Product search / list. `q` optional (omit → list all). Global: matches product name (fa/en), variant color name (fa/en), `sku` and storage value, case-insensitive. **Includes unavailable products.** | `200 { "items": [ProductSummary] }` |
| `GET /api/v1/products/{slug}` | Product detail with every variant (color × storage, each with independent price and stock). | `200 ProductDetail` |
| `GET /api/v1/cart` | The session's cart. No session yet → empty cart; never issues a cookie. | `200 Cart` |
| `POST /api/v1/cart/items` | Add a variant (or increment an existing line). Body: `AddCartItemRequest`. Creates the guest session (Set-Cookie) on the first successful write. | `200 Cart` |
| `PATCH /api/v1/cart/items/{id}` | Set a line's quantity (≥ 1; removal is `DELETE`). Body: `{"quantity": n}`. | `200 Cart` |
| `DELETE /api/v1/cart/items/{id}` | Remove a cart line. | `200 Cart` |
| `GET /api/v1/shipping-methods` | Active shipping methods with prices. | `200 { "items": [ShippingMethod] }` |
| `POST /api/v1/checkout` | Validate recipient + shipping, recalculate the cart from the session, run the mock payment, atomically deduct stock, create the order, clear the cart. Body: `CheckoutRequest`. | `201 CheckoutOrder` |
| `GET /api/health` | Liveness probe for Docker Compose. Outside `/api/v1`; no session. | `200 {"status":"ok"}` |

Cart writes validate the resulting quantity against current stock and
fail with `409 STOCK_UNAVAILABLE` when exceeded.

## 3. Response and error formats

Success responses are the resource itself at top level (no envelope);
lists per §1.

Every non-2xx response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "<localized human-readable>",
    "details": [ ]
  }
}
```

`details` is optional; its shape depends on `code` (documented below)
and it is absent for most errors.

| Code | HTTP | When | `details` shape |
| --- | --- | --- | --- |
| `VALIDATION_ERROR` | 400 | request body or params invalid | `[ { "field": "recipient.phone", "code": "INVALID_PHONE", "message": "<localized>" } ]` |
| `FORBIDDEN_ORIGIN` | 403 | state-changing request (`POST`/`PATCH`/`DELETE`) whose `Origin` header is missing or not in the allowed origins | — |
| `NOT_FOUND` | 404 | unknown slug / cart line id / route | — |
| `CART_EMPTY` | 409 | checkout with no cart lines | — |
| `STOCK_UNAVAILABLE` | 409 | requested quantity exceeds stock (cart write or checkout revalidation) | `[ { "variantId": 12, "productName": "…", "requested": 2, "available": 1 } ]` |
| `INTERNAL_ERROR` | 500 | unexpected failure | — |

HTTP status conventions: `200` for reads and for cart mutations (they
return the updated `Cart`), `201` only for `POST /api/v1/checkout` (a
new order), `400` validation, `403` rejected `Origin` on
state-changing requests, `404` unknown resource, `405` wrong method
(framework-generated), `409` cart/stock state conflicts. **401 is
unused in the MVP**: an expired, invalid or missing session cookie is
treated as "no session" (empty cart; `CART_EMPTY` at checkout), never
as an authentication error.

## 4. Resource schemas

**ProductSummary** — one row per product in search/list responses:

```json
{
  "slug": "iphone-15-pro-max",
  "name": "آیفون ۱۵ پرو مکس",
  "defaultVariantId": 12,
  "colorKey": "natural",
  "colorName": "تیتانیوم طبیعی",
  "storageGb": 256,
  "priceIrr": 489900000,
  "compareAtPriceIrr": 525000000,
  "inStock": true,
  "rating": 5,
  "reviewCount": 246
}
```

`defaultVariantId` selects the seeded display variant (the price and
color the catalog card shows). `compareAtPriceIrr` may be `null` (no
strike-through price). `colorKey` is the stable key the frontend maps
to imagery/hex — the backend never sends image paths or color hex
values.

**ProductDetail**:

```json
{
  "slug": "iphone-15-pro-max",
  "name": "آیفون ۱۵ پرو مکس",
  "defaultVariantId": 12,
  "rating": 5,
  "reviewCount": 246,
  "specs": [
    { "label": "تراشه", "value": "A17 Pro" },
    { "label": "…", "value": "…" }
  ],
  "reviews": [
    { "author": "سارا محمدی", "rating": 5, "text": "…" }
  ],
  "variants": [ "Variant objects, see below" ]
}
```

Specifications, ratings, review counts and review content are
backend-owned seed data: `specs` is an ordered list of localized
label/value pairs (seed order), `reviews` the full seeded list — no
pagination in the MVP — with localized author names and text. `rating`
is a number 0–5 (one decimal allowed; the seeded catalog uses 4.8) and
`reviewCount` an integer.

**Variant** (the unit added to the cart; identified by `id`):

```json
{
  "id": 12,
  "sku": "IP15PM-NAT-256",
  "colorKey": "natural",
  "colorName": "تیتانیوم طبیعی",
  "storageGb": 256,
  "priceIrr": 489900000,
  "compareAtPriceIrr": 525000000,
  "stockQty": 7,
  "inStock": true
}
```

Color and storage chip options are derived client-side from the
`variants` array — not every color × storage combination is guaranteed
to exist as a variant.

**Cart** — the response of every cart endpoint; totals are recomputed
from current variant data on every read (cart lines store no prices):

```json
{
  "items": [
    {
      "id": 3,
      "variantId": 12,
      "productSlug": "iphone-15-pro-max",
      "productName": "آیفون ۱۵ پرو مکس",
      "colorKey": "natural",
      "colorName": "تیتانیوم طبیعی",
      "storageGb": 256,
      "unitPriceIrr": 489900000,
      "quantity": 2,
      "lineTotalIrr": 979800000,
      "maxQuantity": 7
    }
  ],
  "subtotalIrr": 979800000,
  "itemCount": 2,
  "currency": "IRR"
}
```

`maxQuantity` is the variant's current stock (quantity-stepper upper
bound). `itemCount` is the sum of quantities (the header badge number).
There are no discount fields — no promo codes in the MVP.

**ShippingMethod** — stable string ids; the frontend checkout keys its
options by them:

```json
{ "id": "express", "name": "ارسال فوری", "description": "…", "priceIrr": 49000000 }
```

**AddCartItemRequest**: `{ "variantId": 12, "quantity": 1 }`
(`quantity` 1–99; posting an existing variant increments that line).

**CheckoutRequest**:

```json
{
  "recipient": {
    "firstName": "…",
    "lastName": "…",
    "phone": "09123456789",
    "address": "…"
  },
  "shippingMethodId": "express"
}
```

Validation — mandatory on the frontend **and** the backend: first and
last name 2–60 characters; phone an Iranian mobile number
(`^09\d{9}$`); address 10–500 characters; `shippingMethodId` must be
an active method. **No payment fields are accepted**: the card form is
decorative in the MVP and card data is never sent to, validated by or
stored on the backend.

**CheckoutOrder** — the `201` response (names of items and shipping are
snapshots stored in both locales, returned in the request locale):

```json
{
  "trackingCode": "IRC-7F3K9Q2A",
  "status": "paid",
  "recipient": { "firstName": "…", "lastName": "…", "phone": "…", "address": "…" },
  "shippingMethod": { "id": "express", "name": "ارسال فوری", "priceIrr": 49000000 },
  "items": [
    {
      "productName": "آیفون ۱۵ پرو مکس",
      "colorName": "تیتانیوم طبیعی",
      "storageGb": 256,
      "unitPriceIrr": 489900000,
      "quantity": 2,
      "lineTotalIrr": 979800000
    }
  ],
  "subtotalIrr": 979800000,
  "shippingIrr": 49000000,
  "totalIrr": 1028800000,
  "currency": "IRR",
  "payment": { "method": "mock", "transactionId": "TXN-8H2M4P" },
  "createdAt": "2026-11-08T14:30:00Z",
  "paidAt": "2026-11-08T14:30:01Z"
}
```

Order status values: `pending`, `paid`, `processing`, `shipped`,
`delivered`, `cancelled`, `failed`. **Real checkout orders can only
ever hold the status `paid` in the MVP** — every other status is
reserved for clearly marked demo records (`is_demo = 1`). Order-read
APIs and status transitions are future work (§7).

## 5. Guest session cookie, CORS and credentials

- Cookie `guest_sid`, issued **only** by the first successful cart
  write: `HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=2592000`
  (30 days). Value: opaque 256-bit random token; stored server-side as
  a SHA-256 hash with an `expires_at`. Reads (`GET /cart`) never issue
  a cookie.
- `Secure` requires HTTPS; local plain-HTTP development may disable it
  via `COOKIE_SECURE=false` (env) — production keeps it on.
- Expired or invalid cookie → treated as "no session" (empty cart);
  the next cart write issues a fresh one. Never a 401.
- Cart and checkout depend on the cookie, so the frontend calls the
  API with `withCredentials: true` and sends the current locale via
  `Accept-Language`.
- CORS: origins from `CORS_ALLOWED_ORIGINS` (default
  `http://localhost:3000`) with
  `Access-Control-Allow-Credentials: true`, methods
  `GET, POST, PATCH, DELETE`, request headers
  `Content-Type, Accept-Language`.
- CSRF posture: `SameSite=Lax` plus **Origin validation** — every
  state-changing request (`POST`, `PATCH`, `DELETE`) must send an
  `Origin` header matching `CORS_ALLOWED_ORIGINS`; otherwise the
  backend rejects it with `403 FORBIDDEN_ORIGIN` (§3). Reads are
  exempt. CSRF tokens remain future work (§7).

## 6. Checkout transaction (flow of record)

One SQLite transaction (`BEGIN IMMEDIATE`) per checkout request:

1. Validate the body and the shipping method (→ 400
   `VALIDATION_ERROR` before opening the transaction).
2. Load the session's cart lines; empty → 409 `CART_EMPTY`.
3. Re-read the referenced variants and recompute every unit price,
   line total and the subtotal from **current database prices** —
   client input never sets or confirms a price.
4. Revalidate stock per line (`quantity ≤ stockQty`); violation →
   rollback, 409 `STOCK_UNAVAILABLE` with per-variant details.
   Semantics: the **entire** checkout transaction rolls back — no
   order and no payment record of any kind persists.
5. Mock payment: generate `transactionId`, succeed (the normal flow
   always succeeds; no gateway call, no card data).
6. Atomically in the same transaction: deduct stock
   (`stock_qty = stock_qty - quantity`), insert the order with status
   `paid`, its item snapshots and the shipping snapshot, record the
   payment, delete the session's cart lines.
7. Commit → `201 CheckoutOrder`. Stock deduction is therefore atomic
   with payment success; a failure at any step rolls back the whole
   transaction — per step 4, nothing persists.

## 7. Future work (explicitly not MVP)

Deferred, designed-for later: JWT authentication + login endpoint,
order reads (`GET /api/v1/orders`, `GET /api/v1/orders/{trackingCode}`)
for a reporting UI, order-status transitions
(`processing`/`shipped`/`delivered`) and cancel/refund, promo codes,
marketplace sellers and offers, an admin panel and any product
mutation API, pagination and filtering beyond `q`, real IPG
integration and payment reversal, an expired-session/cart cleanup job,
CSRF tokens.

## 8. Provisional decisions

Known provisional choices in the seed data and wire format, kept
deliberate and isolated:

- Product `slug` values are provisional; they live in exactly one
  place (`backend/internal/seed/data.go`) and map 1:1 to the frontend
  routes (`/products/iphone-15-pro-max`, …).
- `rating` wire precision: a number 0–5 with one decimal allowed (the
  seeded catalog uses 4.8), not restricted to integers.
- `ProductSummary.inStock` reflects the **default variant only** — a
  product whose default variant is out of stock shows as unavailable in
  the catalog even if other variants still have stock (the product
  detail page shows live per-variant availability).
