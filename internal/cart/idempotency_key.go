package cart

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

// idempotencyKeyBytes of crypto/rand entropy, base64url-encoded without
// padding, yields 24 characters and 144 bits of entropy — above the 128-bit
// minimum and inside the [16,128] length range shared by every idempotency
// guard in this codebase (JSON header and HTML form alike).
const idempotencyKeyBytes = 18

// GenerateIdempotencyKey produces a fresh idempotency key from randReader.
// It exists so both the JSON route guards and any Go-rendered HTML form can
// generate a key with the exact same algorithm — there is only one
// implementation of key generation in this codebase.
func GenerateIdempotencyKey(randReader io.Reader) (string, error) {
	buf := make([]byte, idempotencyKeyBytes)
	if _, err := io.ReadFull(randReader, buf); err != nil {
		return "", err
	}
	// base64.RawURLEncoding's alphabet (A-Za-z0-9-_) is a strict subset of
	// the ^[A-Za-z0-9._:-]{16,128}$ pattern every idempotency guard accepts.
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NewIdempotencyKey is the production entry point: crypto/rand.Reader,
// never math/rand, never a timestamp, never a product ID or
// CART_COOKIE_SECRET.
func NewIdempotencyKey() (string, error) {
	return GenerateIdempotencyKey(rand.Reader)
}
