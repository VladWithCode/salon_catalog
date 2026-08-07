package routes

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/vladwithcode/salon_catalog/internal/cart"
)

const (
	// maxCartFormBytes matches the JSON API's body cap: enough for any
	// legitimate form, small enough to bound abuse.
	maxCartFormBytes int64 = 8 * 1024
	// cartFallbackReturnTo is used whenever return_to is missing or fails
	// validation. The invalid value is never reflected back to the client.
	cartFallbackReturnTo = "/carrito"
)

var (
	errCartFormInvalid          = errors.New("invalid cart form")
	errCartFormTooLarge         = errors.New("cart form too large")
	errCartFormUnsupportedMedia = errors.New("unsupported cart form media type")
)

// cartReturnToAllowedPaths are the only page families a return_to value may
// target. Anything else — including a technically-local path outside these
// families — falls back to /carrito rather than being trusted blindly.
//
// /catalogo and /solicitar-cotizacion were removed from this list in
// 5B7B6A: neither page renders cart_status/cart_error (their handlers live
// in internal/routes/routes.go, which is outside this phase's file
// permissions), so keeping them in the allowlist would silently drop the
// mutation result. /carrito is the only destination that renders a message
// for every approved code.
var cartReturnToAllowedPaths = []string{"/carrito"}

// generateCartIdempotencyKey and newCartIdempotencyKey delegate to
// internal/cart, the single implementation of idempotency key generation
// shared by the JSON API, the HTML fallback, and any Go-rendered form.
func generateCartIdempotencyKey(randReader io.Reader) (string, error) {
	return cart.GenerateIdempotencyKey(randReader)
}

func newCartIdempotencyKey() (string, error) {
	return cart.NewIdempotencyKey()
}

// sanitizeCartReturnTo validates a client-supplied return_to value against a
// strict allowlist and returns cartFallbackReturnTo for anything that does
// not clearly resolve to a local page in an approved family. An invalid
// value is discarded, never echoed back.
func sanitizeCartReturnTo(raw string) string {
	if raw == "" {
		return cartFallbackReturnTo
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return cartFallbackReturnTo
	}
	if strings.ContainsRune(raw, '\\') || strings.Contains(raw, "#") {
		return cartFallbackReturnTo
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return cartFallbackReturnTo
		}
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" {
		return cartFallbackReturnTo
	}

	for _, allowed := range cartReturnToAllowedPaths {
		if parsed.Path == allowed || strings.HasPrefix(parsed.Path, allowed+"/") {
			return raw
		}
	}
	return cartFallbackReturnTo
}

// parseCartForm enforces the fallback form contract shared by every cart
// command route: exactly application/x-www-form-urlencoded (an optional
// valid charset parameter is tolerated), capped at maxCartFormBytes and
// applied before ParseForm ever reads the body.
func parseCartForm(w http.ResponseWriter, r *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return errCartFormUnsupportedMedia
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCartFormBytes)
	if err := r.ParseForm(); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return errCartFormTooLarge
		}
		return errCartFormInvalid
	}
	return nil
}

// formValueOnce returns a field's single value. A field present more than
// once is always rejected, even when optional. A required field absent
// entirely is rejected; an optional field absent entirely returns ok=true
// with an empty value.
func formValueOnce(form url.Values, key string, required bool) (string, bool) {
	values, present := form[key]
	if !present {
		return "", !required
	}
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

// formHasOnlyAllowedFields rejects any field not explicitly named, including
// cart_id, token, max_quantity, available, source, price, stock, or any
// other field a client should never control.
func formHasOnlyAllowedFields(form url.Values, allowed ...string) bool {
	allowedSet := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = true
	}
	for key := range form {
		if !allowedSet[key] {
			return false
		}
	}
	return true
}

// cartAddItemForm is the fully validated result of parsing the "add"
// fallback/HTMX form. product_id is already a canonical UUID; quantity is
// already a positive integer; idempotency_key already matches the shared
// pattern; return_to is already sanitized.
type cartAddItemForm struct {
	productID      string
	quantity       int
	idempotencyKey string
	returnTo       string
}

func parseCartAddItemForm(r *http.Request) (cartAddItemForm, error) {
	if !formHasOnlyAllowedFields(r.PostForm, "product_id", "quantity", "idempotency_key", "return_to") {
		return cartAddItemForm{}, errCartFormInvalid
	}

	rawProductID, ok := formValueOnce(r.PostForm, "product_id", true)
	if !ok {
		return cartAddItemForm{}, errCartFormInvalid
	}
	productID, valid := normalizeCartAPIUUID(rawProductID)
	if !valid {
		return cartAddItemForm{}, errCartFormInvalid
	}

	rawQuantity, ok := formValueOnce(r.PostForm, "quantity", true)
	if !ok {
		return cartAddItemForm{}, errCartFormInvalid
	}
	quantity, err := strconv.Atoi(strings.TrimSpace(rawQuantity))
	if err != nil || quantity < 1 {
		return cartAddItemForm{}, errCartFormInvalid
	}

	rawKey, ok := formValueOnce(r.PostForm, "idempotency_key", true)
	if !ok || !idempotencyKeyPattern.MatchString(rawKey) {
		return cartAddItemForm{}, errCartFormInvalid
	}

	rawReturnTo, ok := formValueOnce(r.PostForm, "return_to", false)
	if !ok {
		return cartAddItemForm{}, errCartFormInvalid
	}

	return cartAddItemForm{
		productID:      productID,
		quantity:       quantity,
		idempotencyKey: rawKey,
		returnTo:       sanitizeCartReturnTo(rawReturnTo),
	}, nil
}

// cartQuantityForm is the validated result of parsing the "set quantity"
// fallback/HTMX form. product_id comes from the route, not the body.
type cartQuantityForm struct {
	quantity int
	returnTo string
}

func parseCartQuantityForm(r *http.Request) (cartQuantityForm, error) {
	if !formHasOnlyAllowedFields(r.PostForm, "quantity", "return_to") {
		return cartQuantityForm{}, errCartFormInvalid
	}

	rawQuantity, ok := formValueOnce(r.PostForm, "quantity", true)
	if !ok {
		return cartQuantityForm{}, errCartFormInvalid
	}
	quantity, err := strconv.Atoi(strings.TrimSpace(rawQuantity))
	if err != nil || quantity < 1 {
		return cartQuantityForm{}, errCartFormInvalid
	}

	rawReturnTo, ok := formValueOnce(r.PostForm, "return_to", false)
	if !ok {
		return cartQuantityForm{}, errCartFormInvalid
	}

	return cartQuantityForm{quantity: quantity, returnTo: sanitizeCartReturnTo(rawReturnTo)}, nil
}

// cartReturnToOnlyForm is the validated result of parsing the "remove" and
// "clear" fallback/HTMX forms, which carry no field besides return_to.
type cartReturnToOnlyForm struct {
	returnTo string
}

func parseCartReturnToOnlyForm(r *http.Request) (cartReturnToOnlyForm, error) {
	if !formHasOnlyAllowedFields(r.PostForm, "return_to") {
		return cartReturnToOnlyForm{}, errCartFormInvalid
	}
	rawReturnTo, ok := formValueOnce(r.PostForm, "return_to", false)
	if !ok {
		return cartReturnToOnlyForm{}, errCartFormInvalid
	}
	return cartReturnToOnlyForm{returnTo: sanitizeCartReturnTo(rawReturnTo)}, nil
}

// cartLegacyQuantityForm is the validated result of parsing PATCH
// /carrito/items' historical id/action/quantity contract. product_id is
// intentionally not normalized as a UUID here — this route predates that
// check and changing it would alter existing behavior beyond what this
// phase authorizes.
type cartLegacyQuantityForm struct {
	productID string
	action    string
	quantity  int
}

func parseCartLegacyQuantityForm(r *http.Request) (cartLegacyQuantityForm, error) {
	if !formHasOnlyAllowedFields(r.PostForm, "id", "action", "quantity") {
		return cartLegacyQuantityForm{}, errCartFormInvalid
	}

	productID, ok := formValueOnce(r.PostForm, "id", true)
	if !ok {
		return cartLegacyQuantityForm{}, errCartFormInvalid
	}

	action, ok := formValueOnce(r.PostForm, "action", true)
	if !ok {
		return cartLegacyQuantityForm{}, errCartFormInvalid
	}

	rawQuantity, ok := formValueOnce(r.PostForm, "quantity", false)
	if !ok {
		return cartLegacyQuantityForm{}, errCartFormInvalid
	}
	quantity := 0
	if rawQuantity != "" {
		q, err := parseStrictPositiveOrZero(rawQuantity)
		if err != nil {
			return cartLegacyQuantityForm{}, errCartFormInvalid
		}
		quantity = q
	}

	return cartLegacyQuantityForm{productID: productID, action: action, quantity: quantity}, nil
}

// cartFormMutationService is the narrow dependency the HTML cart handlers
// need from internal/cart.Service: the exact same atomic, stock-validated
// operations the JSON API uses. There is exactly one implementation of cart
// stock/persistence rules in this codebase; this interface exists only so
// tests can substitute a fake without a database.
type cartFormMutationService interface {
	AddItemIdempotent(ctx context.Context, cartID string, productID string, quantity int, keyHash []byte, requestHash []byte) (cart.AddItemOutcome, error)
	SetItemQuantity(ctx context.Context, cartID string, productID string, quantity int) error
	DeleteItem(ctx context.Context, cartID string, productID string) error
	Clear(ctx context.Context, cartID string) error
}
