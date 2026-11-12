::: {align="center"}
# Irancell Omnichannel

### Independent Full-Stack E-commerce Portfolio Project

**A Persian-first, RTL iPhone storefront with a Go API, guest cart, and
transactional checkout.**

`<br/>`{=html}

![Next.js](https://img.shields.io/badge/Next.js-16.4.5-black?logo=nextdotjs)
![React](https://img.shields.io/badge/React-18.2.0-61DAFB?logo=react&logoColor=black)
![Go](https://img.shields.io/badge/Go-1.26.0-00ADD8?logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-1.62.0-003B57?logo=sqlite&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5.9.3-3178C6?logo=typescript&logoColor=white)
![Tests](https://img.shields.io/badge/Automated_Tests-127_passed-2E8B57)

`<br/>`{=html}

**[Features](#-features)** · **[Architecture](#-architecture)** ·
**[Getting Started](#-getting-started)** · **[API](#-api-overview)** ·
**[Testing](#-testing)**
:::

> \[!IMPORTANT\] **Independent Portfolio Project --- Not Affiliated with
> Irancell**
>
> This repository contains an independently developed sample e-commerce
> application created for portfolio and demonstration purposes. It is
> not affiliated with, endorsed by, sponsored by, or officially
> maintained by Irancell or its parent company. The Irancell name and
> branding identify the subject of this fictional sample storefront
> only.

------------------------------------------------------------------------

## Table of contents

-   [Project overview](#-project-overview)
-   [Features](#-features)
-   [Architecture](#-architecture)
-   [Technology stack](#-technology-stack)
-   [Project structure](#-project-structure)
-   [Getting started](#-getting-started)
-   [API overview](#-api-overview)
-   [Testing](#-testing)
-   [Security](#-security-considerations)
-   [Known limitations](#-known-limitations)
-   [License](#-license)

## 🛍️ Project overview

Irancell Omnichannel is an independent, full-stack **MVP e-commerce
storefront concept** inspired by the Irancell brand. It brings a focused
iPhone shopping journey into one sample experience: product discovery,
variant selection, a guest shopping cart, and a three-step checkout
ending in a paid demo order with a tracking code.

The catalog contains six seeded sample products. Each product has color
and storage variants with independent prices and stock availability. All
catalog data is served by this repository's own Go backend; it does
**not** represent Irancell's real products, prices, inventory, or
production systems.

The storefront is **Persian-first** with right-to-left (RTL) layouts and
an English left-to-right (LTR) switch. Prices use integer Iranian rial
(IRR) throughout storage, transport, and validation---never
floating-point values.

## ✨ Features

  -----------------------------------------------------------------------
  Area                                What's included
  ----------------------------------- -----------------------------------
  🌐 Localization                     Persian-first fa/en experience,
                                      RTL/LTR layouts, `next-intl`,
                                      cookie-based locale switching, and
                                      localized API responses via
                                      `Accept-Language`

  📱 Product catalog                  API-backed searchable catalog and
                                      dynamic product details, with
                                      seeded sample specs, ratings,
                                      reviews, colors, and storage
                                      variants

  🎨 Variant selection                Independent price and stock
                                      availability for each color ×
                                      storage combination

  🛒 Guest cart                       Add, update, and remove cart lines;
                                      server-side stock validation and
                                      `409 STOCK_UNAVAILABLE` responses

  📦 Checkout                         Three steps: recipient details →
                                      shipping method → confirmation,
                                      with frontend and backend
                                      validation

  💳 Demo payment                     Mock payment always succeeds and
                                      records a synthetic transaction ID;
                                      no payment gateway or card data is
                                      used

  🔒 Transactional stock              Checkout reloads prices and stock
                                      inside one SQLite transaction;
                                      stock deduction and order creation
                                      are atomic

  🍪 Guest session                    HttpOnly `guest_sid` cookie, issued
                                      on the first successful cart write;
                                      no account or login required

  🎞️ Motion & responsive UI           Mobile and desktop layouts, Framer
                                      Motion scroll reveals and route
                                      transitions, with
                                      `prefers-reduced-motion` support
  -----------------------------------------------------------------------

## 🏗️ Architecture

The repository contains two independent applications communicating
through a versioned REST API (`/api/v1`).

-   **Frontend --- Next.js:** renders the storefront. Components do not
    call the API directly; data flows through
    `service → hook → component`. API responses are Zod-validated before
    rendering, while Redux Toolkit mirrors server-driven cart state.
-   **Backend --- Go/Fiber:** a modular monolith organized under
    `internal/`. It owns persistence, guest sessions, pricing, stock
    checks, validation, localization, and checkout. SQLite schema
    changes are managed through migrations, with idempotent seed data
    applied at startup.

``` mermaid
flowchart LR
    subgraph Browser
        UI[Next.js Storefront<br/>fa-RTL / en-LTR]
    end

    subgraph Backend["Go + Fiber backend (:8080)"]
        MW["Middleware: CORS + Origin validation<br/>rate limit, locale, security headers"]
        CAT[Catalog]
        CART["Cart + guest sessions"]
        CHK["Shipping + checkout<br/>single SQLite transaction"]
        SEED["Migrations + seeds"]
    end

    DB[(SQLite<br/>WAL + BEGIN IMMEDIATE)]

    UI -- "HTTP(S) JSON /api/v1<br/>guest_sid cookie + Accept-Language" --> MW
    MW --> CAT
    MW --> CART
    MW --> CHK
    CAT --> DB
    CART --> DB
    CHK --> DB
    SEED --> DB
    CHK -- "Mock payment + atomic stock deduction" --> DB
```

### Guest cart and checkout

The first successful cart write issues an opaque `guest_sid` cookie. The
cookie is HttpOnly and SameSite=Lax, with a 30-day lifetime; the server
stores only its SHA-256 hash.

Cart lines contain variant IDs and quantities---not trusted prices or
totals. Prices, totals, and stock availability are recalculated from
database records on every read and write.

Checkout runs in one `BEGIN IMMEDIATE` transaction:

1.  Validate the request.
2.  Reload the cart and re-price it from current database data.
3.  Re-check stock availability.
4.  Execute the mock payment.
5.  Deduct stock and create a `paid` order.
6.  Clear the cart and commit.

If any step fails, the transaction rolls back.

### Database and persistence

The backend uses a single SQLite database file in WAL mode, with a
5-second busy timeout and foreign keys enabled. The database is accessed
through `modernc.org/sqlite`, a pure-Go driver that does not require
CGO.

Schema changes use an ordered migration history, with each migration
running in its own transaction. Seed data---including products,
variants, shipping methods, and optional demo orders---is applied
idempotently at startup. Docker Compose stores the database in a named
volume.

The complete API contract---including endpoints, response schemas, error
format, cookie/CORS rules, and checkout behavior---is documented in
[`docs/API.md`](docs/API.md).

## 🧰 Technology stack

### Frontend

Versions verified against `frontend/package.json` and
`frontend/package-lock.json`.

  --------------------------------------------------------------------------------------------------
  Technology                                                           Version Role
  ----------------------------------------------- ---------------------------- ---------------------
  [Next.js](https://nextjs.org/)                                        16.4.5 App Router,
                                                                               storefront framework,
                                                                               server components,
                                                                               metadata

  [React](https://react.dev/) / React DOM                               18.2.0 UI runtime

  [TypeScript](https://www.typescriptlang.org/)                          5.9.3 Strict type safety

  [Tailwind CSS](https://tailwindcss.com/)                               4.3.5 RTL-first styling and
                                                                               design tokens

  [Framer Motion](https://motion.dev/)                                 12.43.0 Scroll reveals and
                                                                               route transitions

  [next-intl](https://next-intl.dev/)                                   4.14.9 Persian/English
                                                                               catalogs and locale
                                                                               handling

  Redux Toolkit / React Redux                                   2.14.0 / 9.3.1 Client-side cart
                                                                               state mirrored from
                                                                               API

  Axios                                                                 1.21.0 HTTP client with
                                                                               credentials and
                                                                               locale headers

  [Zod](https://zod.dev/)                                                4.6.7 Runtime API response
                                                                               validation

  [lucide-react](https://lucide.dev/)                                  0.563.0 Icon library

  Vitest / Testing Library / jsdom                     3.2.7 / 16.3.3 / 25.0.1 Frontend unit and
                                                                               component tests

  ESLint / Prettier                                            9.39.5 / 3.9.10 Linting and
                                                                               formatting
  --------------------------------------------------------------------------------------------------

### Backend

Versions verified against `backend/go.mod`.

  ----------------------------------------------------------------------------------------------------------------
  Technology                                                                         Version Role
  ------------------------------------------------------------- ---------------------------- ---------------------
  Go                                                                                  1.26.0 Backend language and
                                                                                             standard library

  [Fiber](https://gofiber.io/)                                                       2.52.16 HTTP framework

  [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)                         1.62.0 Pure-Go SQLite driver

  REST API                                                                         `/api/v1` Versioned JSON API

  Middleware                                                                             --- Recovery, request ID,
                                                                                             structured logging,
                                                                                             CORS/Origin
                                                                                             validation, locale,
                                                                                             rate limiting,
                                                                                             security headers,
                                                                                             body-size limit

  Validation                                                                             --- Localized field-level
                                                                                             errors; prices and
                                                                                             stock are never
                                                                                             trusted from the
                                                                                             client
  ----------------------------------------------------------------------------------------------------------------

The direct runtime dependencies are Fiber and the SQLite driver.

## 📁 Project structure

``` text
.
├── backend/
│   ├── cmd/server/           # Config → database → migrations → seeds → server
│   ├── internal/platform/    # Config, database, migrations, router, middleware, i18n
│   ├── internal/catalog/     # Products, variants, search
│   ├── internal/cart/        # Guest cart and stock/price validation
│   ├── internal/session/     # Guest session cookies
│   ├── internal/checkout/    # Shipping and transactional checkout
│   ├── internal/orders/      # Order write model and status
│   ├── internal/seed/        # Idempotent sample data
│   └── Dockerfile
├── frontend/
│   ├── src/app/              # Home, products, product details, cart, checkout
│   ├── src/components/       # Layout, shop, UI, skeletons, animations
│   ├── src/hooks/            # API hooks
│   ├── src/lib/api/          # Axios client, services, Zod schemas
│   ├── src/store/            # Redux store and cart slice
│   ├── src/i18n/             # Locale resolution
│   ├── messages/             # Persian and English catalogs
│   └── public/               # Local product and hero imagery
├── docs/API.md               # Full REST API contract
├── scripts/run-tests.sh      # Quality gates
└── docker-compose.yml        # Local multi-container setup
```

## 🚀 Getting started

### Prerequisites

Choose one of the following:

-   **Docker:** Docker Engine and Docker Compose v2 (recommended).
-   **Local development:** Go 1.26+ and Node.js 22+ with npm.

### Environment configuration

Both applications have sensible defaults and can start without an `.env`
file.

Optional backend variables are documented in
[`backend/.env.example`](backend/.env.example), including `PORT`,
`DATABASE_PATH`, `CORS_ALLOWED_ORIGINS`, `COOKIE_SECURE`,
`APP_SEED_DEMO_ORDERS`, `RATE_LIMIT_*`, and optional `TURNSTILE_*`
settings.

The frontend variable `NEXT_PUBLIC_API_BASE_URL` is documented in
[`frontend/.env.example`](frontend/.env.example).

### Run with Docker Compose

``` bash
docker compose up --build -d
```

  Service      Local address
  ------------ ----------------------------------
  Storefront   http://localhost:3000
  API health   http://localhost:8080/api/health

The API health endpoint returns `{"status":"ok"}` when healthy.

SQLite data persists in the `omnichannel-data` named volume. Migrations
and seeds run automatically on startup.

``` bash
# Stop containers; keep database data
docker compose down

# Stop containers and remove database data
docker compose down -v
```

### Run locally

Start the backend:

``` bash
cd backend
go build -o server ./cmd/server
COOKIE_SECURE=false ./server
```

In a second terminal, start the frontend:

``` bash
cd frontend
npm install
npm run dev
```

For the guest-session cookie to persist over plain HTTP, use
`COOKIE_SECURE=false` and browse through `http://localhost:3000`.
`localhost` and `127.0.0.1` are different cookie hosts.

### Run the quality gates

From the repository root:

``` bash
./scripts/run-tests.sh
```

## 🔌 API overview

The API is versioned under `/api/v1`. All responses are single-locale
JSON (`Accept-Language: fa` by default), and monetary values are integer
IRR. See [`docs/API.md`](docs/API.md) for the full contract.

  ----------------------------------------------------------------------------
  Method                  Endpoint                     Description
  ----------------------- ---------------------------- -----------------------
  `GET`                   `/api/v1/products?q=`        Search/list products by
                                                       name, color, SKU, or
                                                       storage in Persian and
                                                       English

  `GET`                   `/api/v1/products/{slug}`    Product details,
                                                       specifications,
                                                       reviews, and variants

  `GET`                   `/api/v1/cart`               Current session cart;
                                                       does not issue a cookie

  `POST`                  `/api/v1/cart/items`         Add a variant or
                                                       increment a line;
                                                       issues `guest_sid` on
                                                       first successful write

  `PATCH`                 `/api/v1/cart/items/{id}`    Update line quantity
                                                       with stock validation

  `DELETE`                `/api/v1/cart/items/{id}`    Remove a cart line

  `GET`                   `/api/v1/shipping-methods`   Active delivery options
                                                       and prices

  `POST`                  `/api/v1/checkout`           Validate, re-price,
                                                       mock-pay, deduct stock
                                                       atomically, and create
                                                       a `201` order

  `GET`                   `/api/health`                Liveness endpoint used
                                                       by Docker Compose
  ----------------------------------------------------------------------------

Errors use a shared envelope:

``` json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Localized message",
    "details": {}
  }
}
```

Implemented error codes include `400 VALIDATION_ERROR`,
`403 FORBIDDEN_ORIGIN`, `404 NOT_FOUND`, `409 CART_EMPTY`,
`409 STOCK_UNAVAILABLE`, `429 RATE_LIMITED`, and `500 INTERNAL_ERROR`.

There is no `401` response: a missing or expired guest session is
treated as "no session yet".

## 🧪 Testing

The following checks were reported as passing in the project
verification run. The counts below distinguish automated tests from
manual QA.

### Automated verification

  -----------------------------------------------------------------------
  Area                                Result
  ----------------------------------- -----------------------------------
  Go backend                          **85 tests passed** across 10
                                      packages with
                                      `go test -count=1 ./...`

  Go race detector                    `go test -race -count=1 ./...`
                                      passed across all 10 packages

  Go quality                          `gofmt -l .`, `go vet ./...`, and
                                      `go build ./...` clean

  Frontend                            **42 Vitest tests passed** across 7
                                      files

  Frontend quality                    ESLint and `npx tsc --noEmit` clean

  Production build                    `npm run build` completed
                                      successfully

  Docker Compose                      `docker compose config` validated
                                      successfully
  -----------------------------------------------------------------------

Frontend tests cover Zod schemas and contract fixtures, API services and
error mapping, the `useCheckout` and `useAddToCart` hooks, and the
product imagery map.

### Security-related automated tests

These are included in the 85 Go tests above, not additional tests.

-   Security headers are asserted on API responses.
-   SQL-injection payloads are tested against parameterized queries and
    escaped `LIKE` patterns.
-   XSS-related input handling is checked; user input is not rendered as
    raw HTML.
-   Oversized request bodies are rejected at the 256 KB limit.
-   Rate limiting returns `429` with `Retry-After`.
-   Turnstile behavior is tested against a stubbed verification
    endpoint, including fail-closed behavior when configured.
-   Session uniqueness, expiry, invalid-session rejection, and
    concurrent limited-stock checkouts are tested.

### Manual verification

The application was manually exercised against a locally running
frontend and backend in both Persian (RTL) and English (LTR):

-   Product catalog browsing and search.
-   Product detail pages and variant switching.
-   Cart add/update/remove operations and stock-limit feedback.
-   Three-step checkout.
-   Receipt page and generated tracking code.
-   Locale rendering and switching.

The Docker Compose stack was also run locally: both containers reported
healthy, the storefront responded on port `3000`, and a guest
cart-to-checkout journey was completed through the API on port `8080`.

> **Not performed:** There are no Playwright or other browser-based
> automated E2E tests, and no penetration test has been conducted.

## 🛡️ Security considerations

Implemented safeguards include:

-   **Security headers:** `nosniff`, `X-Frame-Options: DENY`,
    `Referrer-Policy: no-referrer`, a deny-all API CSP, and
    `Permissions-Policy`; the storefront also applies a CSP.
-   **Origin validation:** state-changing `POST`, `PATCH`, and `DELETE`
    requests with a missing or disallowed `Origin` are rejected with
    `403 FORBIDDEN_ORIGIN`. The session cookie uses `SameSite=Lax`.
-   **Rate limiting:** in-memory per-IP limits of 600 requests/minute
    API-wide and 30 requests/minute for checkout, with `Retry-After`.
-   **Input handling:** 256 KB request body limit, parameterized SQL
    with escaped `LIKE` patterns, and Zod validation of API responses
    before rendering.
-   **Session protection:** opaque guest tokens are stored server-side
    only as SHA-256 hashes. Cookies are HttpOnly and use `Secure` in
    HTTPS deployments; Secure is disabled only for local plain-HTTP
    development.
-   **Optional bot verification:** Cloudflare Turnstile is fail-closed
    on checkout when `TURNSTILE_SECRET_KEY` is configured; it is
    disabled by default.
-   **Container hardening:** application containers run as non-root
    users.

This is **not production-hardened**. No penetration test has been
performed. Rate limiting is in-memory, applies to a single instance, and
resets on restart. HTTPS termination and HSTS are expected to be
provided by a deployment reverse proxy. CSRF tokens, full
authentication, and an admin interface are outside the current MVP
scope.

## ⚠️ Known limitations

This project is deliberately scoped as an MVP:

-   **Mock payment only:** payment always succeeds and records a
    synthetic transaction ID; no real payment gateway is integrated.
-   **No authentication:** the application uses guest sessions only;
    login/JWT flows are future work.
-   **No order reads or reporting:** checkout creates a `paid` order and
    tracking code, but order history, status transitions, and
    admin/reporting APIs are not implemented.
-   **Single seller and fixed catalog:** only seeded iPhone sample
    products are included; there are no marketplace sellers, offers,
    product mutation APIs, or promo codes.
-   **No pagination:** catalog and review endpoints return the full
    seeded set; search is the only catalog filter.
-   **Provisional imagery mapping:** some product colors are mapped to
    fallback imagery when the design assets do not include an exact
    match. See `frontend/src/lib/productImages.ts`.

## 📄 License

No license has been specified for this repository. All rights are
reserved by the repository owner; no permission to reuse, redistribute,
or commercially exploit this code is granted.

Third-party assets, trademarks, and branding remain the property of
their respective owners and may be subject to separate terms.

This is an independent portfolio sample created for demonstration
purposes only. It has no official relationship with, endorsement from,
or authorization by Irancell.
