package routes

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal/cart"
	"github.com/vladwithcode/salon_catalog/internal/db"
	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
	"github.com/vladwithcode/salon_catalog/internal/templates/util"
)

// cartStatusMessages and cartErrorMessages are the only copy the /carrito
// page and HTMX fragments ever render for a mutation outcome. Both the
// fallback redirect (?cart_status=/?cart_error=) and HTMX paths select from
// these fixed strings; neither ever echoes client- or database-supplied
// text.
var cartStatusMessages = map[string]string{
	"added":   "Se añadió el producto a tu selección.",
	"updated": "Se actualizó tu selección.",
	"removed": "Se eliminó el producto de tu selección.",
	"cleared": "Se limpió tu selección.",
}

var cartErrorMessages = map[string]string{
	"product_not_found":    "No encontramos ese producto.",
	"product_unavailable":  "Ese producto ya no está disponible.",
	"insufficient_stock":   "No hay stock suficiente disponible.",
	"cart_item_not_found":  "Ese elemento ya no está en tu selección.",
	"idempotency_conflict": "Esa solicitud ya se procesó con otros datos. Intenta de nuevo.",
	"cart_unavailable":     "Error al actualizar el carrito",
}

func cartErrorQueryCode(err error) string {
	switch {
	case errors.Is(err, cart.ErrProductNotFound):
		return "product_not_found"
	case errors.Is(err, cart.ErrProductUnavailable):
		return "product_unavailable"
	case errors.Is(err, cart.ErrInsufficientStock):
		return "insufficient_stock"
	case errors.Is(err, cart.ErrCartItemNotFound):
		return "cart_item_not_found"
	case errors.Is(err, cart.ErrIdempotencyConflict):
		return "idempotency_conflict"
	default:
		return "cart_unavailable"
	}
}

// buildCartRedirectLocation appends the appropriate cart_status or
// cart_error to an already-sanitized return_to, replacing any prior value of
// either, and preserves every other query parameter return_to already
// carried (buscar, categoria, pagina, por_pagina, or anything else the
// originating page needs). url.Values.Encode handles percent-encoding, so
// accented characters and other special characters are preserved correctly.
func buildCartRedirectLocation(returnTo string, statusCode string, errorCode string) string {
	parsed, err := url.Parse(returnTo)
	if err != nil {
		parsed, _ = url.Parse(cartFallbackReturnTo)
	}
	query := parsed.Query()
	query.Del("cart_status")
	query.Del("cart_error")
	if statusCode != "" {
		query.Set("cart_status", statusCode)
	}
	if errorCode != "" {
		query.Set("cart_error", errorCode)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func productionCartFormMutationService() cartFormMutationService {
	return cart.NewService()
}

func RegisterCartRoutes(router *customServeMux, cartSessions *session.CartManager, csrfGuard *appsecurity.CSRFGuard) {
	registerCartRoutes(router, cartSessions, csrfGuard, productionCartFormMutationService())
}

func registerCartRoutes(router *customServeMux, cartSessions *session.CartManager, csrfGuard *appsecurity.CSRFGuard, mutations cartFormMutationService) {
	// GET /carrito is the only route here that renders a full page (via
	// BaseLayout, when not an HX-Request), so it is the only one that needs
	// publicMiddleware's social-links context for the footer. Every
	// mutation below only ever produces a redirect or an HTMX fragment,
	// neither of which touches social links, so they use the lighter
	// withCartSession directly — the same pattern the JSON API already
	// uses, and one that does not require a database connection just to
	// resolve the cart session and CSRF guard.
	router.HandleFunc("GET /carrito", publicMiddleware(cartSessions, getCartHandler()))

	// Legacy HTMX routes. Kept registered for rollback only — no active
	// template sends a request to PUT /carrito anymore (5B7B6A migrated the
	// CTA, overlay, and modal add controls to POST /carrito/items). If it
	// is ever exercised again, it now demands a client-supplied
	// idempotency_key like every other add path; it no longer mints one
	// itself. Migrated internally to internal/cart.Service so they share
	// the same atomic, stock-validated writes as everything else.
	router.HandleFunc("PUT /carrito", csrfGuard.Protect(withCartAddItemLegacyFormGuard(cartSessions, addToCartHandler(mutations))))
	router.HandleFunc("PATCH /carrito/items", csrfGuard.Protect(withCartLegacyQuantityFormGuard(cartSessions, updateCartQuantityLegacyHandler(mutations))))
	router.HandleFunc("DELETE /carrito/items", csrfGuard.Protect(withCartSession(cartSessions, clearCartHandler(mutations))))
	router.HandleFunc("DELETE /carrito/items/{id}", csrfGuard.Protect(withCartSession(cartSessions, removeFromCartLegacyHandler(mutations))))

	// Progressive fallback routes: real forms, functional without
	// JavaScript, Post/Redirect/Get on success and on business errors. Form
	// parsing and validation run entirely before cart session middleware —
	// an invalid request never generates a cart UUID, never sets a cookie,
	// never reaches the handler or internal/cart.Service.
	router.HandleFunc("POST /carrito/items", csrfGuard.Protect(withCartAddItemFormGuard(cartSessions, addToCartFallbackHandler(mutations))))
	router.HandleFunc("POST /carrito/items/{product_id}/cantidad", csrfGuard.Protect(withCartQuantityFormGuard(cartSessions, updateCartQuantityFallbackHandler(mutations))))
	router.HandleFunc("POST /carrito/items/{product_id}/eliminar", csrfGuard.Protect(withCartRemoveFormGuard(cartSessions, removeFromCartFallbackHandler(mutations))))
	router.HandleFunc("POST /carrito/vaciar", csrfGuard.Protect(withCartReturnToOnlyFormGuard(cartSessions, clearCartFallbackHandler(mutations))))
}

// withCartAddItemFormGuard parses and validates the add-item form —
// including idempotency_key — entirely before the cart session middleware
// runs. A missing or malformed field is rejected here: no cart UUID is
// generated, no Set-Cookie is issued, the handler never runs, and no
// database call ever happens.
func withCartAddItemFormGuard(cartSessions *session.CartManager, next func(http.ResponseWriter, *http.Request, cartAddItemForm)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := parseCartForm(w, r); err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		form, err := parseCartAddItemForm(r)
		if err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		withCartSession(cartSessions, func(w http.ResponseWriter, r *http.Request) {
			next(w, r, form)
		})(w, r)
	}
}

// withCartAddItemLegacyFormGuard is the equivalent guard for the legacy PUT
// /carrito rollback route. As of 5B7B6A it no longer mints a fresh
// idempotency_key server-side: a stale key generated per-request could not
// recognize a retried request, defeating idempotency entirely. It now
// requires the client to supply idempotency_key like every other add path;
// missing or malformed values are rejected here, before the cart session is
// ever created.
func withCartAddItemLegacyFormGuard(cartSessions *session.CartManager, next func(http.ResponseWriter, *http.Request, cartAddItemForm)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := parseCartForm(w, r); err != nil {
			handleCartUpdateError(w, r, err, true)
			return
		}

		rawProductID, ok := formValueOnce(r.PostForm, "product_id", true)
		if !ok {
			handleCartUpdateError(w, r, errCartFormInvalid, true)
			return
		}
		productID, valid := normalizeCartAPIUUID(rawProductID)
		if !valid {
			handleCartUpdateError(w, r, errCartFormInvalid, true)
			return
		}

		rawKey, ok := formValueOnce(r.PostForm, "idempotency_key", true)
		if !ok || !idempotencyKeyPattern.MatchString(rawKey) {
			handleCartUpdateError(w, r, errCartFormInvalid, true)
			return
		}

		form := cartAddItemForm{productID: productID, quantity: 1, idempotencyKey: rawKey, returnTo: cartFallbackReturnTo}
		withCartSession(cartSessions, func(w http.ResponseWriter, r *http.Request) {
			next(w, r, form)
		})(w, r)
	}
}

// withCartQuantityFormGuard validates product_id (from the route) and the
// quantity/return_to form fields entirely before the cart session
// middleware runs.
func withCartQuantityFormGuard(cartSessions *session.CartManager, next func(http.ResponseWriter, *http.Request, string, cartQuantityForm)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, valid := normalizeCartAPIUUID(r.PathValue("product_id"))
		if !valid {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := parseCartForm(w, r); err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		form, err := parseCartQuantityForm(r)
		if err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		withCartSession(cartSessions, func(w http.ResponseWriter, r *http.Request) {
			next(w, r, productID, form)
		})(w, r)
	}
}

// withCartRemoveFormGuard validates product_id (from the route) and
// return_to entirely before the cart session middleware runs.
func withCartRemoveFormGuard(cartSessions *session.CartManager, next func(http.ResponseWriter, *http.Request, string, cartReturnToOnlyForm)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, valid := normalizeCartAPIUUID(r.PathValue("product_id"))
		if !valid {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := parseCartForm(w, r); err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		form, err := parseCartReturnToOnlyForm(r)
		if err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		withCartSession(cartSessions, func(w http.ResponseWriter, r *http.Request) {
			next(w, r, productID, form)
		})(w, r)
	}
}

// withCartReturnToOnlyFormGuard validates return_to entirely before the
// cart session middleware runs. Used by POST /carrito/vaciar.
func withCartReturnToOnlyFormGuard(cartSessions *session.CartManager, next func(http.ResponseWriter, *http.Request, cartReturnToOnlyForm)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := parseCartForm(w, r); err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		form, err := parseCartReturnToOnlyForm(r)
		if err != nil {
			respondCartFormGuardError(w, r, err)
			return
		}
		withCartSession(cartSessions, func(w http.ResponseWriter, r *http.Request) {
			next(w, r, form)
		})(w, r)
	}
}

// withCartLegacyQuantityFormGuard validates PATCH /carrito/items' historical
// id/action/quantity form entirely before the cart session middleware runs.
func withCartLegacyQuantityFormGuard(cartSessions *session.CartManager, next func(http.ResponseWriter, *http.Request, cartLegacyQuantityForm)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := parseCartForm(w, r); err != nil {
			handleCartUpdateError(w, r, err, true)
			return
		}
		form, err := parseCartLegacyQuantityForm(r)
		if err != nil {
			handleCartUpdateError(w, r, err, true)
			return
		}
		withCartSession(cartSessions, func(w http.ResponseWriter, r *http.Request) {
			next(w, r, form)
		})(w, r)
	}
}

func respondCartFormGuardError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, errCartFormTooLarge):
		status = http.StatusRequestEntityTooLarge
	case errors.Is(err, errCartFormUnsupportedMedia):
		status = http.StatusUnsupportedMediaType
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
}

func addToCartFallbackHandler(mutations cartFormMutationService) func(http.ResponseWriter, *http.Request, cartAddItemForm) {
	return addToCartHandler(mutations)
}

// addToCartHandler is the single core implementation shared by PUT /carrito
// (legacy HTMX) and POST /carrito/items (progressive fallback and current
// HTMX button). It is the only place that calls AddItemIdempotent.
func addToCartHandler(mutations cartFormMutationService) func(http.ResponseWriter, *http.Request, cartAddItemForm) {
	return func(w http.ResponseWriter, r *http.Request, form cartAddItemForm) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			respondCartMutationHTML(w, r, "", err, "", form.returnTo)
			return
		}

		keyHash := hashCartAPIIdempotencyKey(form.idempotencyKey)
		requestHash := hashCartAPIAddItemRequest(form.productID, form.quantity)

		_, mutationErr := mutations.AddItemIdempotent(r.Context(), cartID, form.productID, form.quantity, keyHash, requestHash)
		respondCartMutationHTML(w, r, cartID, mutationErr, "added", form.returnTo)
	}
}

// updateCartQuantityFallbackHandler receives an already-validated product_id
// and cartQuantityForm from withCartQuantityFormGuard; form parsing and
// validation ran entirely before the cart session was created.
func updateCartQuantityFallbackHandler(mutations cartFormMutationService) func(http.ResponseWriter, *http.Request, string, cartQuantityForm) {
	return func(w http.ResponseWriter, r *http.Request, productID string, form cartQuantityForm) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			respondCartMutationHTML(w, r, "", err, "", form.returnTo)
			return
		}

		mutationErr := mutations.SetItemQuantity(r.Context(), cartID, productID, form.quantity)
		respondCartMutationHTML(w, r, cartID, mutationErr, "updated", form.returnTo)
	}
}

// updateCartQuantityLegacyHandler preserves PATCH /carrito/items' existing
// contract exactly: form fields id, action (increase/decrease/set), and for
// set, quantity. It resolves the current quantity via the same canonical
// GET-style read used everywhere else and delegates the actual write to
// internal/cart.Service.SetItemQuantity — the same atomic, stock-validated
// path PATCH /api/cart/items/{product_id} already uses. Unlike the JSON API,
// this route's contract predates strict quantity semantics: zero or
// negative values here still mean "remove", matching prior behavior exactly
// via DeleteItem instead of a rejected write.
// updateCartQuantityLegacyHandler receives an already-validated
// cartLegacyQuantityForm from withCartLegacyQuantityFormGuard: form parsing,
// Content-Type/size enforcement, and field allow-listing ran entirely before
// the cart session was created.
func updateCartQuantityLegacyHandler(mutations cartFormMutationService) func(http.ResponseWriter, *http.Request, cartLegacyQuantityForm) {
	return func(w http.ResponseWriter, r *http.Request, form cartLegacyQuantityForm) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handleCartUpdateError(w, r, err, true)
			return
		}

		existingCart, err := db.GetOrCreateCart(r.Context(), cartID)
		if err != nil {
			handleCartUpdateError(w, r, err, true)
			return
		}
		currentQuantity := existingCart.GetItemQuantity(form.productID)

		var newQuantity int
		switch form.action {
		case "increase":
			newQuantity = currentQuantity + 1
		case "decrease":
			newQuantity = currentQuantity - 1
		case "set":
			newQuantity = form.quantity
		default:
			handleCartUpdateError(w, r, fmt.Errorf("invalid action: %s", form.action), true)
			return
		}

		var mutationErr error
		if newQuantity <= 0 {
			mutationErr = mutations.DeleteItem(r.Context(), cartID, form.productID)
		} else {
			mutationErr = mutations.SetItemQuantity(r.Context(), cartID, form.productID, newQuantity)
		}
		respondCartMutationHTML(w, r, cartID, mutationErr, "updated", cartFallbackReturnTo)
	}
}

func parseStrictPositiveOrZero(raw string) (int, error) {
	value := 0
	if raw == "" {
		return 0, nil
	}
	_, err := fmt.Sscanf(raw, "%d", &value)
	if err != nil {
		return 0, err
	}
	return value, nil
}

// removeFromCartFallbackHandler receives an already-validated product_id and
// cartReturnToOnlyForm from withCartRemoveFormGuard.
func removeFromCartFallbackHandler(mutations cartFormMutationService) func(http.ResponseWriter, *http.Request, string, cartReturnToOnlyForm) {
	return func(w http.ResponseWriter, r *http.Request, productID string, form cartReturnToOnlyForm) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			respondCartMutationHTML(w, r, "", err, "", form.returnTo)
			return
		}

		mutationErr := mutations.DeleteItem(r.Context(), cartID, productID)
		respondCartMutationHTML(w, r, cartID, mutationErr, "removed", form.returnTo)
	}
}

func removeFromCartLegacyHandler(mutations cartFormMutationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handleCartUpdateError(w, r, err, true)
			return
		}

		productID := r.PathValue("id")
		mutationErr := mutations.DeleteItem(r.Context(), cartID, productID)
		respondCartMutationHTML(w, r, cartID, mutationErr, "removed", cartFallbackReturnTo)
	}
}

// clearCartFallbackHandler receives an already-validated cartReturnToOnlyForm
// from withCartReturnToOnlyFormGuard.
func clearCartFallbackHandler(mutations cartFormMutationService) func(http.ResponseWriter, *http.Request, cartReturnToOnlyForm) {
	return func(w http.ResponseWriter, r *http.Request, form cartReturnToOnlyForm) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			respondCartMutationHTML(w, r, "", err, "", form.returnTo)
			return
		}

		mutationErr := mutations.Clear(r.Context(), cartID)
		respondCartMutationHTML(w, r, cartID, mutationErr, "cleared", form.returnTo)
	}
}

func clearCartHandler(mutations cartFormMutationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handleCartUpdateError(w, r, err, true)
			return
		}

		mutationErr := mutations.Clear(r.Context(), cartID)
		respondCartMutationHTML(w, r, cartID, mutationErr, "cleared", cartFallbackReturnTo)
	}
}

// respondCartMutationHTML is the single point where an HTMX fragment or a
// 303 redirect is chosen and rendered. Every fallback and HTMX handler above
// funnels through it, so business errors are classified from the same typed
// internal/cart sentinels and rendered with the same fixed copy regardless
// of transport.
func respondCartMutationHTML(w http.ResponseWriter, r *http.Request, cartID string, mutationErr error, successStatus string, returnTo string) {
	if !isHTMXRequest(r) {
		var location string
		if mutationErr != nil {
			location = buildCartRedirectLocation(returnTo, "", cartErrorQueryCode(mutationErr))
		} else {
			location = buildCartRedirectLocation(returnTo, successStatus, "")
		}
		http.Redirect(w, r, location, http.StatusSeeOther)
		return
	}

	if mutationErr != nil {
		renderCartHTMXError(w, r, mutationErr)
		return
	}

	freshCart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		renderCartHTMXError(w, r, err)
		return
	}

	cartState := components.CartState{
		Cart:      freshCart,
		HasUpdate: true,
		UpdateMsg: cartStatusMessages[successStatus],
	}
	renderCart(w, r, cartState, true)
}

func renderCartHTMXError(w http.ResponseWriter, r *http.Request, mutationErr error) {
	w.WriteHeader(http.StatusInternalServerError)

	code := cartErrorQueryCode(mutationErr)
	// CartSidebar returns early when Cart is nil (see components.ErrNilCart),
	// which would otherwise silently produce an empty "cartError" fragment
	// instead of the error markup. An empty, non-nil cart satisfies the
	// guard without implying any real cart state.
	cartState := components.CartState{Cart: &db.Cart{}}
	cartState.HasError = true
	cartState.ErrorMsg = cartErrorMessages[code]

	cs := components.CartSidebar(cartState)
	comps := []*templ.Component{&cs}

	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData(cartState.ErrorMsg, components.ToastError, 3000, true, false)
	t := components.ToasterToast(toastData)
	comps = append(comps, &t)

	mixedConf := util.WithMixedFragmentsConf{
		JoinTemplates: comps,
		Fragments:     []any{"toaster-toast", "cartError"},
	}
	util.RenderMixedWithFragments(r.Context(), w, mixedConf)
	log.Printf("cart mutation failed err: %v\n", mutationErr)
}

// getCartHandler renders the HTMX sidebar fragment when called with
// HX-Request, and a full standalone HTML page otherwise — the page every
// fallback redirect above lands on, and the only way the cart is visible at
// all without JavaScript.
func getCartHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handleCartUpdateError(w, r, err, true)
			return
		}

		currentCart, err := db.GetOrCreateCart(r.Context(), cartID)
		if err != nil {
			if isHTMXRequest(r) {
				handleCartUpdateError(w, r, err, true)
				return
			}
			renderCartPage(w, r, nil, cartPageStatusFromDBError())
			return
		}

		if isHTMXRequest(r) {
			cartState := components.CartState{Cart: currentCart}
			renderCart(w, r, cartState, false)
			return
		}

		renderCartPage(w, r, currentCart, cartPageStatusFromQuery(r))
	}
}

func cartPageStatusFromDBError() *pages.CartPageStatus {
	return &pages.CartPageStatus{Message: cartErrorMessages["cart_unavailable"], IsError: true}
}

// cartPageStatusFromQuery interprets only the approved cart_status and
// cart_error codes landed on by a redirect; any other or missing value
// renders no message at all. The query string itself is never rendered.
func cartPageStatusFromQuery(r *http.Request) *pages.CartPageStatus {
	query := r.URL.Query()
	if status := query.Get("cart_status"); status != "" {
		if message, ok := cartStatusMessages[status]; ok {
			return &pages.CartPageStatus{Message: message, IsError: false}
		}
	}
	if errorCode := query.Get("cart_error"); errorCode != "" {
		if message, ok := cartErrorMessages[errorCode]; ok {
			return &pages.CartPageStatus{Message: message, IsError: true}
		}
	}
	return nil
}

func renderCartPage(w http.ResponseWriter, r *http.Request, cartValue *db.Cart, status *pages.CartPageStatus) {
	err := pages.CartPage(cartValue, status).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render cart page: %v\n", err)
	}
}

func renderCart(w http.ResponseWriter, r *http.Request, cartState components.CartState, includeToast bool) {
	comps := []*templ.Component{}

	fc := components.FloatingCart(len(cartState.Cart.Items), false)
	cs := components.CartSidebar(cartState)
	comps = append(comps, &fc, &cs)

	if includeToast {
		w.Header().Add("X-Includes-Toast", "true")
		toastData := components.NewToastData(cartState.UpdateMsg, components.ToastSuccess, 3000, true, false)
		t := components.ToasterToast(toastData)
		comps = append(comps, &t)
	}

	mixedConf := util.WithMixedFragmentsConf{
		JoinTemplates: comps,
		Fragments:     []any{"toaster-toast", "cartToggle", "cartSidebar"},
	}
	util.RenderMixedWithFragments(r.Context(), w, mixedConf)
}

func handleCartUpdateError(w http.ResponseWriter, r *http.Request, err error, includeToast bool) {
	w.WriteHeader(http.StatusInternalServerError)

	// See the identical note in renderCartHTMXError: CartSidebar requires a
	// non-nil Cart even to render its error fragment.
	cartState := components.CartState{Cart: &db.Cart{}}
	cartState.HasError = true
	cartState.ErrorMsg = "Error al actualizar el carrito"
	if r.Method == http.MethodGet {
		cartState.ErrorMsg = "Error al cargar el carrito"
	}

	cs := components.CartSidebar(cartState)
	comps := []*templ.Component{
		&cs,
	}
	if includeToast {
		w.Header().Add("X-Includes-Toast", "true")
		toastData := components.NewToastData(cartState.ErrorMsg, components.ToastError, 3000, true, false)
		t := components.ToasterToast(toastData)
		comps = append(comps, &t)
	}

	mixedConf := util.WithMixedFragmentsConf{
		JoinTemplates: comps,
		Fragments:     []any{"toaster-toast", "cartError"},
	}
	util.RenderMixedWithFragments(r.Context(), w, mixedConf)
	log.Printf("failed to process cart update err: %v\n", err)
}
