package routes

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/util"
)

func RegisterCartRoutes(router *customServeMux) {
	router.HandleFunc("GET /carrito", publicMiddleware(GetCart))
	router.HandleFunc("PUT /carrito", publicMiddleware(AddToCart))
	router.HandleFunc("PATCH /carrito/items", publicMiddleware(UpdateCartQuantity))
	router.HandleFunc("DELETE /carrito/items", publicMiddleware(ClearCart))
	router.HandleFunc("DELETE /carrito/items/{id}", publicMiddleware(RemoveFromCart))
}

func GetCart(w http.ResponseWriter, r *http.Request) {
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handleCartUpdateError(w, r, err, true)
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		handleCartUpdateError(w, r, err, true)
		return
	}

	cartState := components.CartState{
		Cart: cart,
	}
	renderCart(w, r, cartState, false)
}

func AddToCart(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handleCartUpdateError(w, r, err, true)
		return
	}

	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handleCartUpdateError(w, r, err, true)
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handleCartUpdateError(w, r, err, true)
		return
	}

	productID := r.FormValue("product_id")
	source := r.FormValue("source")

	// Check if item already exists in cart
	existingItem := false
	for _, item := range cart.Items {
		if item.ProductID == productID {
			existingItem = true
			break
		}
	}

	if existingItem {
		// Update quantity of existing item
		newQty := cart.GetItemQuantity(productID) + 1
		cart.UpdateItemQty(productID, newQty)
	} else {
		// Add new item to cart
		prod, err := db.FindCatalogProductDetail(productID)
		if err != nil {
			handleCartUpdateError(w, r, err, true)
			return
		}

		cartItem := &db.CartItem{
			ProductID: productID,
			Source:    source,
			Name:      prod.Name,
			Category:  prod.CategoryName,
			ImageURL:  prod.ImageURL,
			Quantity:  1,
			MaxQty:    prod.Quantity,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		cart.AddItem(cartItem)
	}

	// Save cart to database
	err = cart.Save(r.Context())
	if err != nil {
		handleCartUpdateError(w, r, err, true)
		return
	}

	cartState := components.CartState{
		Cart:      cart,
		HasUpdate: true,
		UpdateMsg: "Se añadió el producto al carrito",
	}
	renderCart(w, r, cartState, true)
}

func UpdateCartQuantity(w http.ResponseWriter, r *http.Request) {
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handleCartUpdateError(w, r, err, true)
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		handleCartUpdateError(w, r, err, true)
		return
	}

	updateMsg := "Se actualizó el carrito"
	productID := r.FormValue("id")
	action := r.FormValue("action")

	currentQty := cart.GetItemQuantity(productID)
	var newQty int

	switch action {
	case "increase":
		newQty = currentQty + 1
	case "decrease":
		newQty = currentQty - 1
	case "set":
		newQty, _ = strconv.Atoi(r.FormValue("quantity"))
	default:
		handleCartUpdateError(w, r, fmt.Errorf("invalid action: %s", action), true)
		return
	}

	cart.UpdateItemQty(productID, newQty)
	if newQty > cart.GetItemMaxQty(productID) {
		updateMsg = "Ya has elegido el máximo posible para este producto"
	}

	// Save cart to database
	err = cart.Save(r.Context())
	if err != nil {
		handleCartUpdateError(w, r, err, true)
		return
	}

	cartState := components.CartState{
		Cart:      cart,
		HasUpdate: true,
		UpdateMsg: updateMsg,
	}
	renderCart(w, r, cartState, true)
}

func RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handleCartUpdateError(w, r, err, true)
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		handleCartUpdateError(w, r, err, true)
		return
	}

	productID := r.PathValue("id")
	cart.RemoveItem(productID)

	// Save cart to database
	err = cart.Save(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handleCartUpdateError(w, r, err, true)
		return
	}

	cartState := components.CartState{
		Cart:      cart,
		HasUpdate: true,
		UpdateMsg: "Se eliminó el producto del carrito",
	}
	renderCart(w, r, cartState, true)
}

func ClearCart(w http.ResponseWriter, r *http.Request) {
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handleCartUpdateError(w, r, err, true)
		return
	}

	cart, err := db.FindCartByID(r.Context(), cartID)
	if err != nil {
		handleCartUpdateError(w, r, err, true)
		return
	}

	// Delete the entire cart from database
	cart.ClearCart()

	err = cart.Save(r.Context())
	if err != nil {
		handleCartUpdateError(w, r, err, true)
		return
	}

	cartState := components.CartState{
		Cart:      cart,
		HasUpdate: true,
		UpdateMsg: "Se limpió el carrito",
	}
	renderCart(w, r, cartState, true)
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

	cartState := components.CartState{}
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
