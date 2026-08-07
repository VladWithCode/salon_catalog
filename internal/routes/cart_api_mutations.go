package routes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal/cart"
	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

const maxCartAPIRequestBytes int64 = 8 * 1024

var (
	errCartAPIInvalidRequest         = errors.New("invalid cart API request")
	errCartAPIRequestTooLarge        = errors.New("cart API request too large")
	errCartAPIUnsupportedMedia       = errors.New("unsupported cart API media type")
	errCartAPIProductNotFound        = errors.New("cart API product not found")
	errCartAPIProductUnavailable     = errors.New("cart API product unavailable")
	errCartAPIInsufficientStock      = errors.New("cart API insufficient stock")
	errCartAPIItemNotFound           = errors.New("cart API item not found")
	errCartAPIUnavailable            = errors.New("cart API unavailable")
	errCartAPIIdempotencyKeyRequired = errors.New("cart API idempotency key required")
	errCartAPIInvalidIdempotencyKey  = errors.New("cart API invalid idempotency key")
	errCartAPIIdempotencyConflict    = errors.New("cart API idempotency conflict")
)

// idempotencyKeyPattern matches the client-generated Idempotency-Key header
// required on POST /api/cart/items: 16-128 characters from a fixed ASCII
// set. A value outside this pattern — including one with interior or
// exterior whitespace, commas, NUL, or non-ASCII characters — is rejected
// outright, never trimmed or corrected.
var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{16,128}$`)

type addCartAPIItemRequest struct {
	ProductID *string `json:"product_id"`
	Quantity  *int    `json:"quantity"`
}

type updateCartAPIItemRequest struct {
	Quantity *int `json:"quantity"`
}

type cartAPIMutationService interface {
	AddItem(context.Context, string, string, int) error
	AddItemIdempotent(ctx context.Context, cartID string, productID string, quantity int, keyHash []byte, requestHash []byte) (cart.AddItemOutcome, error)
	SetItemQuantity(context.Context, string, string, int) error
	DeleteItem(context.Context, string, string) error
	Clear(context.Context, string) error
}

// cartMutationOperations is the shape of internal/cart.Service. Routes
// depend on this narrow interface, not the concrete type, so the HTTP-to-
// error-code mapping below can be tested with a fake that returns
// internal/cart's typed sentinels without a database.
type cartMutationOperations interface {
	AddItem(context.Context, string, string, int) error
	AddItemIdempotent(ctx context.Context, cartID string, productID string, quantity int, keyHash []byte, requestHash []byte) (cart.AddItemOutcome, error)
	SetItemQuantity(context.Context, string, string, int) error
	DeleteItem(context.Context, string, string) error
	Clear(context.Context, string) error
}

// cartMutationServiceAdapter translates the typed errors returned by
// internal/cart (the single atomic implementation of cart mutations) into
// the errCartAPI* sentinels writeCartAPIMutationError already maps to the
// stable public HTTP contract. It performs no validation or business logic
// of its own.
type cartMutationServiceAdapter struct {
	operations cartMutationOperations
}

func newCartAPIMutationService(operations cartMutationOperations) cartAPIMutationService {
	return &cartMutationServiceAdapter{operations: operations}
}

func (adapter *cartMutationServiceAdapter) AddItem(ctx context.Context, cartID string, productID string, quantity int) error {
	return translateCartServiceError(adapter.operations.AddItem(ctx, cartID, productID, quantity))
}

func (adapter *cartMutationServiceAdapter) AddItemIdempotent(
	ctx context.Context,
	cartID string,
	productID string,
	quantity int,
	keyHash []byte,
	requestHash []byte,
) (cart.AddItemOutcome, error) {
	outcome, err := adapter.operations.AddItemIdempotent(ctx, cartID, productID, quantity, keyHash, requestHash)
	return outcome, translateCartServiceError(err)
}

func (adapter *cartMutationServiceAdapter) SetItemQuantity(ctx context.Context, cartID string, productID string, quantity int) error {
	return translateCartServiceError(adapter.operations.SetItemQuantity(ctx, cartID, productID, quantity))
}

func (adapter *cartMutationServiceAdapter) DeleteItem(ctx context.Context, cartID string, productID string) error {
	return translateCartServiceError(adapter.operations.DeleteItem(ctx, cartID, productID))
}

func (adapter *cartMutationServiceAdapter) Clear(ctx context.Context, cartID string) error {
	return translateCartServiceError(adapter.operations.Clear(ctx, cartID))
}

func translateCartServiceError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, cart.ErrProductNotFound):
		return errCartAPIProductNotFound
	case errors.Is(err, cart.ErrProductUnavailable):
		return errCartAPIProductUnavailable
	case errors.Is(err, cart.ErrInsufficientStock):
		return errCartAPIInsufficientStock
	case errors.Is(err, cart.ErrCartItemNotFound):
		return errCartAPIItemNotFound
	case errors.Is(err, cart.ErrIdempotencyConflict):
		return errCartAPIIdempotencyConflict
	default:
		return errCartAPIUnavailable
	}
}

func RegisterCartAPIMutationRoutes(router *customServeMux, cartSessions *session.CartManager, csrfGuard *appsecurity.CSRFGuard) {
	registerCartAPIMutationRoutes(
		router,
		cartSessions,
		csrfGuard,
		databaseCartAPIDataLoader{},
		newCartAPIMutationService(cart.NewService()),
	)
}

func registerCartAPIMutationRoutes(router *customServeMux, cartSessions *session.CartManager, csrfGuard *appsecurity.CSRFGuard, loader cartAPIDataLoader, mutations cartAPIMutationService) {
	router.HandleFunc(
		"POST /api/cart/items",
		csrfGuard.Protect(requireCartAPIIdempotencyKey(withCartSession(cartSessions, postCartAPIItemHandler(loader, mutations)))),
	)
	router.HandleFunc("PATCH /api/cart/items/{product_id}", withProtectedCartSession(csrfGuard, cartSessions, patchCartAPIItemHandler(loader, mutations)))
	router.HandleFunc("DELETE /api/cart/items/{product_id}", withProtectedCartSession(csrfGuard, cartSessions, deleteCartAPIItemHandler(loader, mutations)))
	router.HandleFunc("DELETE /api/cart", withProtectedCartSession(csrfGuard, cartSessions, deleteCartAPIHandler(loader, mutations)))
}

// cartAPIIdempotencyKeyContextKey stores the already-validated
// Idempotency-Key header value so postCartAPIItemHandler never has to
// re-parse or re-validate it: requireCartAPIIdempotencyKey is the only place
// that reads the raw header.
type cartAPIIdempotencyKeyContextKey struct{}

// requireCartAPIIdempotencyKey enforces the mandatory Idempotency-Key header
// on POST /api/cart/items before anything else in the chain runs: before the
// cart session middleware resolves or issues a cart identity, before
// Set-Cookie, before the handler, before any product read or transaction. A
// missing or malformed key is rejected here and next is never called.
func requireCartAPIIdempotencyKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, err := validateCartAPIIdempotencyKeyHeader(r)
		if err != nil {
			writeCartAPIMutationError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), cartAPIIdempotencyKeyContextKey{}, key)
		next(w, r.WithContext(ctx))
	}
}

func validateCartAPIIdempotencyKeyHeader(r *http.Request) (string, error) {
	values := r.Header.Values("Idempotency-Key")
	if len(values) == 0 {
		return "", errCartAPIIdempotencyKeyRequired
	}
	if len(values) != 1 || !idempotencyKeyPattern.MatchString(values[0]) {
		return "", errCartAPIInvalidIdempotencyKey
	}
	return values[0], nil
}

func cartAPIIdempotencyKeyFromContext(r *http.Request) (string, bool) {
	key, ok := r.Context().Value(cartAPIIdempotencyKeyContextKey{}).(string)
	return key, ok
}

// hashCartAPIIdempotencyKey computes the SHA-256 digest stored in place of
// the raw Idempotency-Key header value. The raw key is never written to the
// database or logged.
func hashCartAPIIdempotencyKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// hashCartAPIAddItemRequest computes the SHA-256 digest of the canonical
// representation of an add-item operation: method, path, the canonical
// product UUID, and the base-10 quantity, each on its own line. It is
// derived from the parsed and validated fields, never from the raw JSON
// body, so unrelated JSON formatting differences (field order, whitespace)
// never change the hash, while a different product or quantity always does.
func hashCartAPIAddItemRequest(productID string, quantity int) []byte {
	canonical := "POST /api/cart/items\n" + productID + "\n" + strconv.Itoa(quantity)
	sum := sha256.Sum256([]byte(canonical))
	return sum[:]
}

func postCartAPIItemHandler(loader cartAPIDataLoader, mutations cartAPIMutationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input addCartAPIItemRequest
		if err := decodeCartAPIJSON(w, r, &input); err != nil {
			writeCartAPIMutationError(w, err)
			return
		}
		if input.ProductID == nil || input.Quantity == nil || *input.Quantity < 1 {
			writeCartAPIMutationError(w, errCartAPIInvalidRequest)
			return
		}
		productID, valid := normalizeCartAPIUUID(*input.ProductID)
		if !valid {
			writeCartAPIMutationError(w, errCartAPIInvalidRequest)
			return
		}
		idempotencyKey, ok := cartAPIIdempotencyKeyFromContext(r)
		if !ok {
			// requireCartAPIIdempotencyKey always sets this before this
			// handler can run. Its absence means a wiring mistake, not a
			// client error: fail closed rather than proceed unprotected.
			writeCartAPIMutationError(w, errCartAPIUnavailable)
			return
		}
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			writeCartAPIMutationError(w, errCartAPIUnavailable)
			return
		}

		keyHash := hashCartAPIIdempotencyKey(idempotencyKey)
		requestHash := hashCartAPIAddItemRequest(productID, *input.Quantity)

		outcome, err := mutations.AddItemIdempotent(r.Context(), cartID, productID, *input.Quantity, keyHash, requestHash)
		if err != nil {
			writeCartAPIMutationError(w, err)
			return
		}
		writeReloadedCartAPIWithReplayHeader(w, r, loader, cartID, outcome == cart.AddItemReplayed)
	}
}

func patchCartAPIItemHandler(loader cartAPIDataLoader, mutations cartAPIMutationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, valid := normalizeCartAPIUUID(r.PathValue("product_id"))
		if !valid {
			writeCartAPIMutationError(w, errCartAPIInvalidRequest)
			return
		}
		var input updateCartAPIItemRequest
		if err := decodeCartAPIJSON(w, r, &input); err != nil {
			writeCartAPIMutationError(w, err)
			return
		}
		if input.Quantity == nil || *input.Quantity < 1 {
			writeCartAPIMutationError(w, errCartAPIInvalidRequest)
			return
		}
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			writeCartAPIMutationError(w, errCartAPIUnavailable)
			return
		}
		if err = mutations.SetItemQuantity(r.Context(), cartID, productID, *input.Quantity); err != nil {
			writeCartAPIMutationError(w, err)
			return
		}
		writeReloadedCartAPI(w, r, loader, cartID)
	}
}

func deleteCartAPIItemHandler(loader cartAPIDataLoader, mutations cartAPIMutationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, valid := normalizeCartAPIUUID(r.PathValue("product_id"))
		if !valid {
			writeCartAPIMutationError(w, errCartAPIInvalidRequest)
			return
		}
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			writeCartAPIMutationError(w, errCartAPIUnavailable)
			return
		}
		if err = mutations.DeleteItem(r.Context(), cartID, productID); err != nil {
			writeCartAPIMutationError(w, err)
			return
		}
		writeReloadedCartAPI(w, r, loader, cartID)
	}
}

func deleteCartAPIHandler(loader cartAPIDataLoader, mutations cartAPIMutationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			writeCartAPIMutationError(w, errCartAPIUnavailable)
			return
		}
		if err = mutations.Clear(r.Context(), cartID); err != nil {
			writeCartAPIMutationError(w, err)
			return
		}
		writeReloadedCartAPI(w, r, loader, cartID)
	}
}

func decodeCartAPIJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errCartAPIUnsupportedMedia
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCartAPIRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return classifyCartAPIDecodeError(err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return classifyCartAPIDecodeError(err)
	}
	return nil
}

func classifyCartAPIDecodeError(err error) error {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return errCartAPIRequestTooLarge
	}
	return errCartAPIInvalidRequest
}

func normalizeCartAPIUUID(value string) (string, bool) {
	value = strings.TrimSpace(value)
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil || parsed.String() != value {
		return "", false
	}
	return parsed.String(), true
}

func writeReloadedCartAPI(w http.ResponseWriter, r *http.Request, loader cartAPIDataLoader, cartID string) {
	response, err := loadCartAPIResponse(r.Context(), loader, cartID)
	if err != nil {
		writeCartAPIMutationError(w, errCartAPIUnavailable)
		return
	}
	writeCartAPIJSON(w, http.StatusOK, response)
}

// writeReloadedCartAPIWithReplayHeader is writeReloadedCartAPI plus an
// Idempotency-Replayed header, set to exactly "true" and only present when
// replayed is true. A freshly applied operation gets no such header at all,
// rather than an explicit "false": the header's mere presence is the
// signal, so there is exactly one way to spell "this was a replay."
func writeReloadedCartAPIWithReplayHeader(w http.ResponseWriter, r *http.Request, loader cartAPIDataLoader, cartID string, replayed bool) {
	response, err := loadCartAPIResponse(r.Context(), loader, cartID)
	if err != nil {
		writeCartAPIMutationError(w, errCartAPIUnavailable)
		return
	}
	if replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	writeCartAPIJSON(w, http.StatusOK, response)
}

func writeCartAPIMutationError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	code := "cart_unavailable"
	switch {
	case errors.Is(err, errCartAPIInvalidRequest):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, errCartAPIRequestTooLarge):
		status, code = http.StatusRequestEntityTooLarge, "request_too_large"
	case errors.Is(err, errCartAPIUnsupportedMedia):
		status, code = http.StatusUnsupportedMediaType, "unsupported_media_type"
	case errors.Is(err, errCartAPIProductNotFound):
		status, code = http.StatusNotFound, "product_not_found"
	case errors.Is(err, errCartAPIProductUnavailable):
		status, code = http.StatusConflict, "product_unavailable"
	case errors.Is(err, errCartAPIInsufficientStock):
		status, code = http.StatusConflict, "insufficient_stock"
	case errors.Is(err, errCartAPIItemNotFound):
		status, code = http.StatusNotFound, "cart_item_not_found"
	case errors.Is(err, errCartAPIIdempotencyKeyRequired):
		status, code = http.StatusBadRequest, "idempotency_key_required"
	case errors.Is(err, errCartAPIInvalidIdempotencyKey):
		status, code = http.StatusBadRequest, "invalid_idempotency_key"
	case errors.Is(err, errCartAPIIdempotencyConflict):
		status, code = http.StatusConflict, "idempotency_conflict"
	}
	writeCartAPIJSON(w, status, cartAPIErrorResponse{Error: code})
}
