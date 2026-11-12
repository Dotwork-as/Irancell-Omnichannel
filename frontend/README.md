# Irancell Omnichannel — Frontend (فروشگاه ایرانسل)

Next.js 16 storefront for the Irancell Omnichannel MVP: a Persian/RTL
iPhone store (English LTR switch included) fully integrated with the
Go backend — live catalog with search, guest cart, shipping methods,
and a three-stage checkout with mock payment.

> **This is an MVP.** Authentication/identity, order reporting, promo
> codes, marketplace sellers and real payment integration were
> intentionally left out of scope. They are future capabilities; the
> API contract ([`../docs/API.md`](../docs/API.md) §7) and this
> frontend's service/hook architecture are designed to accommodate
> them.

## Technologies

| Area | Choice |
| --- | --- |
| Framework | Next.js 16 (App Router), React 18 |
| Language | TypeScript 5 (strict) |
| Styling | Tailwind CSS v4 (7 design tokens, RTL-first) |
| i18n | next-intl 4.14 (fa/en via cookie, no `[locale]` segment) |
| State | Redux Toolkit (cart slice) |
| API | Axios + Zod runtime validation (hooks + services, no `any`) |
| Animation | framer-motion (scroll reveals, route transitions) |
| Testing | Vitest + Testing Library (jsdom) |

## Architecture

```
src/
├── app/                  Server pages + (main) route group
│   └── (main)/           home · products · products/[slug] ·
│                         cart · checkout · checkout/success
├── components/           layout / shop / ui / skeleton / animations
├── hooks/                useProducts, useProduct, useCart,
│                         useAddToCart, useUpdateCartItem,
│                         useRemoveCartItem, useShippingMethods,
│                         useCheckout
├── lib/api/              axios client (credentials + Accept-Language)
│   ├── services/         catalog · cart · checkout
│   └── types/schemas.ts  Zod schemas mirroring the API contract
├── store/                Redux store + cart slice (server-driven)
└── i18n/                 cookie-based locale resolution
```

Components never call Axios: data flows `service → hook → component`,
and every response is Zod-validated before rendering. The contract
source of truth is [`../docs/API.md`](../docs/API.md).

## Product imagery

Product images live locally under `public/` (no external image URLs):
`public/home/` holds the homepage hero asset and
`public/products/{slug}/` one render per catalog product. The backend
never sends image paths ([`../docs/API.md`](../docs/API.md) §4) — the
UI maps the stable `slug`/`colorKey` pair to an asset via
`src/lib/productImages.ts`; `src/components/shop/productArt.tsx`
renders the mapped image and falls back to the `PhoneArt` vector for
any unmapped pair, so a missing asset can never surface as a broken
image. Assets are exports of the Figma "E-Commerce UI Kit" design;
renders are named for the device they actually depict and the mapping
to catalog colors is **provisional** (the kit imagery does not cover
every seeded color — e.g. the Silver render is shared by the two
blue-toned products; see the map's header comment and
`productImages.test.ts`, which asserts on-disk presence and the
documented sharing allowlist).

## Setup

```bash
npm install
cp .env.example .env.local   # optional — defaults to localhost:8080
```

`NEXT_PUBLIC_API_BASE_URL` defaults to `http://localhost:8080/api/v1`
and must point at the running backend (see `../backend/README.md`).

## Run

```bash
npm run dev      # development — http://localhost:3000
npm run build && npm run start   # production
```

Local HTTP note: keep the backend on `COOKIE_SECURE=false` and browse
via `http://localhost:3000` so the guest-session cookie persists
(SameSite=Lax is host-exact — 127.0.0.1 ≠ localhost). Or use Docker:
`docker compose up --build -d` from the repository root (see the
root `README.md`).

## Scripts

```bash
npm run dev | build | start | lint
npx tsc --noEmit    # type check
npx vitest run      # API-layer tests (schemas/services/hooks)
```

## Security

Next.js server applies security headers to every page (nosniff, DENY
framing, no-referrer, Permissions-Policy, CSP with
`connect-src` for the API origin). React escapes all rendered data;
there is no `dangerouslySetInnerHTML` anywhere. Backend responses are
Zod-validated before rendering, so malformed payloads cannot reach
the DOM. Test dependencies (Vitest/Testing Library/jsdom) are
dev-only and never shipped.

## Limitations (MVP)

- Guest checkout only; no login/identity.
- Mock payment (no gateway, no card data).
- Product catalog is iPhone-only seed data; promo codes and order
  history/reporting are out of scope.
