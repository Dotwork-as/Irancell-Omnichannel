// Package session owns guest session management (docs/API.md §5): a
// cryptographically random opaque token rides in the guest_sid
// HttpOnly cookie; only its SHA-256 hash is stored server-side with
// a 30-day expiry. Sessions are issued exclusively by the first
// successful cart write; reads never set a cookie, and missing,
// invalid or expired cookies simply mean "no session" — never 401.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/database"
)

// CookieName is the guest session cookie (docs/API.md §5).
const CookieName = "guest_sid"

// CookieMaxAge is the approved 30-day lifetime in seconds.
const CookieMaxAge = 2592000

// Service manages guest sessions.
type Service struct {
	db           *database.DB
	cookieSecure bool
}

// New builds a session Service; cookieSecure controls the cookie's
// Secure flag (COOKIE_SECURE env — true everywhere except local
// plain-HTTP development).
func New(db *database.DB, cookieSecure bool) *Service {
	return &Service{db: db, cookieSecure: cookieSecure}
}

// newToken returns a fresh 256-bit random token and its SHA-256 hex
// hash (what gets stored).
func newToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("session: generate token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

func hashOf(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// CreateIn inserts a new guest session inside the caller's
// transaction (the cart write's transaction, so cookie issuance and
// the cart mutation are atomic). Returns the raw token for the
// cookie and the session row id.
func (s *Service) CreateIn(ctx context.Context, tx database.Querier) (rawToken string, id int64, err error) {
	raw, hash, err := newToken()
	if err != nil {
		return "", 0, err
	}
	expires := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	res, err := tx.ExecContext(ctx,
		"INSERT INTO guest_sessions (token_hash, expires_at) VALUES (?, ?)", hash, expires)
	if err != nil {
		return "", 0, fmt.Errorf("session: insert: %w", err)
	}
	if id, err = res.LastInsertId(); err != nil {
		return "", 0, fmt.Errorf("session: insert id: %w", err)
	}
	return raw, id, nil
}

// Create inserts a standalone session in its own transaction.
func (s *Service) Create(ctx context.Context) (rawToken string, id int64, err error) {
	err = s.db.WithTx(ctx, func(tx database.Querier) error {
		rawToken, id, err = s.CreateIn(ctx, tx)
		return err
	})
	return rawToken, id, err
}

// Resolve maps a raw cookie token to a live session id. Missing,
// unknown, or expired tokens resolve to 0 ("no session") — callers
// treat that as an empty cart, never an auth error. Expired rows are
// left in place: the cleanup job is deferred beyond the MVP
// (docs/API.md §7).
func (s *Service) Resolve(ctx context.Context, rawToken string) (int64, error) {
	return s.ResolveWith(ctx, s.db, rawToken)
}

// ResolveWith is Resolve against a specific query surface — pass the
// transaction's connection when resolving inside a WithTx block (the
// pool's single connection is pinned for the transaction's duration).
func (s *Service) ResolveWith(ctx context.Context, q database.Querier, rawToken string) (int64, error) {
	if rawToken == "" {
		return 0, nil
	}
	var id int64
	var expires string
	err := q.QueryRowContext(ctx,
		"SELECT id, expires_at FROM guest_sessions WHERE token_hash = ?",
		hashOf(rawToken)).Scan(&id, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("session: resolve: %w", err)
	}
	exp, err := time.Parse(time.RFC3339, expires)
	if err != nil {
		return 0, fmt.Errorf("session: parse expiry: %w", err)
	}
	if !time.Now().UTC().Before(exp) {
		return 0, nil // expired
	}
	return id, nil
}

// Cookie builds the Set-Cookie cookie for a freshly issued token
// with the approved attributes (HttpOnly, Secure per config,
// SameSite=Lax, Path=/, Max-Age 30 days).
func (s *Service) Cookie(rawToken string) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     CookieName,
		Value:    rawToken,
		Path:     "/",
		MaxAge:   CookieMaxAge,
		Expires:  time.Now().UTC().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   s.cookieSecure,
		SameSite: "Lax",
	}
}

// TokenFrom reads the raw guest cookie value from a request ("" when
// absent).
func TokenFrom(c *fiber.Ctx) string {
	return c.Cookies(CookieName)
}
