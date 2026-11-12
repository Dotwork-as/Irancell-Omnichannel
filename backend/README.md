# Irancell Omnichannel — Backend (بک‌اند ایرانسل)

Go + Fiber + SQLite REST API for the Irancell Omnichannel MVP:
localized iPhone catalog with search, guest sessions and cart,
database-owned shipping methods, and a transactional checkout with
mock payment (real orders are always created `paid`; any failure rolls
back the entire transaction).

> **This is an MVP.** Authentication/JWT, order reporting, promo
> codes, an admin panel and real payment integration were
> intentionally left out of scope. They are future capabilities; the
> modular-monolith layout (`internal/<module>`) and the future-work
> inventory in [`../docs/API.md`](../docs/API.md) §7 are designed to
> accommodate them.

## Technologies

Go 1.26 · Fiber v2 · SQLite via `modernc.org/sqlite` (pure Go, no
CGO) · guest cookie sessions (SHA-256-hashed tokens) · structured JSON
logging. No other runtime dependencies.

## Architecture

```
cmd/server/main.go    entrypoint: config → logger → db → migrate →
                      seed → serve (graceful SIGTERM)
internal/
├── platform/         config, SQLite + migrations, router, middleware
│                     (recover, request-id, logging, CORS + Origin
│                     validation, locale fa/en), /api/health, i18n
├── session/          guest session cookie (30 days)
├── catalog/          products + variants + search (read-only)
├── cart/             guest cart + stock/price validation
├── checkout/         shipping methods + transactional checkout
├── orders/           order write model + status enum
└── seed/             idempotent startup seeds (6 products, shipping,
                      7 demo orders)
data/                 SQLite file (gitignored, volume in Docker)
```

Contract source of truth: [`../docs/API.md`](../docs/API.md) —
9 endpoints, integer IRR prices, localized (fa default) responses,
§3 error envelope.

## Run

```bash
go build -o server ./cmd/server
PORT=8080 DATABASE_PATH=./data/omnichannel.db COOKIE_SECURE=false ./server
```

Migrations and seed data apply automatically at startup. Health probe:
`GET /api/health` → `200 {"status":"ok"}` (liveness only).

## Configuration (environment)

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP listen port |
| `DATABASE_PATH` | `./data/omnichannel.db` | SQLite file |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` | Allowed origins |
| `COOKIE_SECURE` | `true` | Set `false` only for local plain-HTTP |
| `APP_SEED_DEMO_ORDERS` | `true` | Seed 7 demo orders |
| `RATE_LIMIT_MAX` | `600` | API-wide requests per client IP per minute |
| `RATE_LIMIT_CHECKOUT_MAX` | `30` | Checkout requests per client IP per minute |
| `TURNSTILE_SECRET_KEY` | *(empty)* | Optional fail-closed Cloudflare Turnstile on checkout |
| `TURNSTILE_VERIFY_URL` | Cloudflare siteverify URL | Override (used by tests to stub verification) |

## Testing

```bash
gofmt -l . && go vet ./... && go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

## Security

Implemented and tested: security headers on every response (nosniff,
DENY framing, no-referrer, deny-all API CSP, Permissions-Policy);
per-IP rate limiting (600/min API, 30/min checkout → localized
`429 RATE_LIMITED` + `Retry-After`; in-memory, single instance);
256KB body limit; 10s/15s/60s read/write/idle timeouts;
X-Forwarded-For never trusted; optional fail-closed Cloudflare
Turnstile on checkout (`TURNSTILE_SECRET_KEY`, disabled when unset);
non-root container user; parameterized SQL with escaped LIKE
patterns; §3 envelopes without internal details. Security tests:
`go test ./internal/platform/server/ -run "TestSecurityHeaders|TestInjection|TestOversized|TestRateLimit"`
and the checkout Turnstile matrix.

## Limitations (MVP)

- Guest checkout only; orders stay `paid` — status transitions are
  demo-seed-only. Order-read/reporting APIs are future work
  ([`../docs/API.md`](../docs/API.md) §7).
- No auth, no admin APIs, no promo codes; rate limiting is
  in-memory only (single instance).
- Seed data is provisional (slugs/rating precision — see
  [`../docs/API.md`](../docs/API.md) §8).
