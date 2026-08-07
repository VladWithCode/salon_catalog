// Package session provides authenticated request identities for public sessions.
package session

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const (
	// CartCookieName is the host-only cookie used for the authenticated cart identity.
	CartCookieName = "cart_id"
	// CartTokenLifetime is both the cryptographic lifetime and browser lifetime.
	CartTokenLifetime = 30 * 24 * time.Hour

	cartTokenVersion   = "v1"
	maxFutureClockSkew = 5 * time.Minute
	maxCartTokenLength = 128
	minimumSecretBytes = 32
)

var (
	ErrCartCookieSecretMissing  = errors.New("required env var CART_COOKIE_SECRET is not set")
	ErrCartCookieSecretTooShort = errors.New("CART_COOKIE_SECRET must be at least 32 bytes")
	ErrCartCookieSecureMissing  = errors.New("required env var CART_COOKIE_SECURE is not set")
	ErrCartCookieSecureInvalid  = errors.New("CART_COOKIE_SECURE must be exactly true or false")
	ErrInvalidCartToken         = errors.New("invalid cart token")
	ErrExpiredCartToken         = errors.New("expired cart token")
	ErrFutureCartToken          = errors.New("cart token issued too far in the future")
)

// Config contains the dependencies needed to create a CartManager.
type Config struct {
	Secret        string
	Secure        bool
	Clock         func() time.Time
	UUIDGenerator func() (uuid.UUID, error)
}

// CartManager signs, verifies, and resolves cart identities for requests.
type CartManager struct {
	secret        []byte
	secure        bool
	clock         func() time.Time
	uuidGenerator func() (uuid.UUID, error)
}

type cartIDContextKey struct{}

// NewCartManager validates the supplied configuration and constructs a manager.
func NewCartManager(config Config) (*CartManager, error) {
	if config.Secret == "" {
		return nil, ErrCartCookieSecretMissing
	}
	if len([]byte(config.Secret)) < minimumSecretBytes {
		return nil, ErrCartCookieSecretTooShort
	}

	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}
	uuidGenerator := config.UUIDGenerator
	if uuidGenerator == nil {
		uuidGenerator = uuid.NewV7
	}

	secret := append([]byte(nil), []byte(config.Secret)...)
	return &CartManager{
		secret:        secret,
		secure:        config.Secure,
		clock:         clock,
		uuidGenerator: uuidGenerator,
	}, nil
}

// NewCartManagerFromEnv loads and validates the process-level cart cookie settings.
func NewCartManagerFromEnv() (*CartManager, error) {
	config, err := cartConfigFromLookup(os.LookupEnv)
	if err != nil {
		return nil, err
	}
	return NewCartManager(config)
}

func cartConfigFromLookup(lookupEnv func(string) (string, bool)) (Config, error) {
	secret, ok := lookupEnv("CART_COOKIE_SECRET")
	if !ok || secret == "" {
		return Config{}, ErrCartCookieSecretMissing
	}
	if len([]byte(secret)) < minimumSecretBytes {
		return Config{}, ErrCartCookieSecretTooShort
	}

	secureValue, ok := lookupEnv("CART_COOKIE_SECURE")
	if !ok || secureValue == "" {
		return Config{}, ErrCartCookieSecureMissing
	}

	var secure bool
	switch secureValue {
	case "true":
		secure = true
	case "false":
		secure = false
	default:
		return Config{}, ErrCartCookieSecureInvalid
	}

	return Config{Secret: secret, Secure: secure}, nil
}

// Middleware resolves the authenticated cart identity once and adds it to context.
func (m *CartManager) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := m.clock().UTC().Truncate(time.Second)
		cartID, valid := m.cartIDFromRequest(r, now)
		if !valid {
			var err error
			cartID, err = m.uuidGenerator()
			if err != nil || !isUUIDv7(cartID) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			token := m.signToken(cartID, now)
			http.SetCookie(w, &http.Cookie{
				Name:     CartCookieName,
				Value:    token,
				Path:     "/",
				Domain:   "",
				Expires:  now.Add(CartTokenLifetime),
				MaxAge:   int(CartTokenLifetime / time.Second),
				HttpOnly: true,
				Secure:   m.secure,
				SameSite: http.SameSiteLaxMode,
			})
		}

		ctx := context.WithValue(r.Context(), cartIDContextKey{}, cartID)
		next(w, r.WithContext(ctx))
	}
}

// CartIDFromContext returns the authenticated UUIDv7 resolved by CartManager.
func CartIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	cartID, ok := ctx.Value(cartIDContextKey{}).(uuid.UUID)
	return cartID, ok && isUUIDv7(cartID)
}

func (m *CartManager) cartIDFromRequest(r *http.Request, now time.Time) (uuid.UUID, bool) {
	var cartCookies []*http.Cookie
	for _, cookie := range r.Cookies() {
		if cookie.Name == CartCookieName {
			cartCookies = append(cartCookies, cookie)
		}
	}
	if len(cartCookies) != 1 {
		return uuid.Nil, false
	}

	cartID, err := m.verifyToken(cartCookies[0].Value, now)
	return cartID, err == nil
}

func (m *CartManager) signToken(cartID uuid.UUID, issuedAt time.Time) string {
	payload := fmt.Sprintf("%s.%s.%d", cartTokenVersion, cartID.String(), issuedAt.Unix())
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + signature
}

func (m *CartManager) verifyToken(token string, now time.Time) (uuid.UUID, error) {
	if token == "" || len(token) > maxCartTokenLength || strings.ContainsRune(token, '\x00') {
		return uuid.Nil, ErrInvalidCartToken
	}
	if strings.IndexFunc(token, unicode.IsSpace) >= 0 {
		return uuid.Nil, ErrInvalidCartToken
	}

	segments := strings.Split(token, ".")
	if len(segments) != 4 || segments[0] != cartTokenVersion {
		return uuid.Nil, ErrInvalidCartToken
	}

	cartID, err := uuid.Parse(segments[1])
	if err != nil || cartID.String() != segments[1] || !isUUIDv7(cartID) {
		return uuid.Nil, ErrInvalidCartToken
	}

	issuedAtUnix, err := strconv.ParseInt(segments[2], 10, 64)
	if err != nil || strconv.FormatInt(issuedAtUnix, 10) != segments[2] {
		return uuid.Nil, ErrInvalidCartToken
	}

	receivedSignature, err := base64.RawURLEncoding.DecodeString(segments[3])
	if err != nil || len(receivedSignature) != sha256.Size || base64.RawURLEncoding.EncodeToString(receivedSignature) != segments[3] {
		return uuid.Nil, ErrInvalidCartToken
	}

	payload := strings.Join(segments[:3], ".")
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	if !hmac.Equal(receivedSignature, mac.Sum(nil)) {
		return uuid.Nil, ErrInvalidCartToken
	}

	now = now.UTC().Truncate(time.Second)
	issuedAt := time.Unix(issuedAtUnix, 0).UTC()
	if issuedAt.Before(now.Add(-CartTokenLifetime)) {
		return uuid.Nil, ErrExpiredCartToken
	}
	if issuedAt.After(now.Add(maxFutureClockSkew)) {
		return uuid.Nil, ErrFutureCartToken
	}

	return cartID, nil
}

func isUUIDv7(value uuid.UUID) bool {
	return value != uuid.Nil && value.Version() == uuid.Version(7) && value.Variant() == uuid.RFC4122
}
