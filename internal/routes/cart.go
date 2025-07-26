package routes

import (
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
)

var cart []db.CartItem

func RegisterCartRoutes(router *customServeMux) {
	router.HandleFunc("GET /cart", GetCart)
	router.HandleFunc("POST /cart/add", AddToCart)
	router.HandleFunc("POST /cart/update-quantity/{id}", UpdateCartQuantity)
	router.HandleFunc("POST /cart/clear", ClearCart)
	router.HandleFunc("DELETE /cart/remove/{id}", RemoveFromCart)
}

func GetCart(w http.ResponseWriter, r *http.Request) {
	var total int
	for _, item := range cart {
		total += item.Quantity * item.Price
	}

	err := components.CartSidebar(cart, total).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render CartSidebar err: %v\n", err)
	}
}

func AddToCart(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse form"))
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	productID := r.FormValue("product_id")
	source := r.FormValue("source")
	var itemPtr *db.CartItem

	for _, item := range cart {
		if item.ProductID == productID {
			itemPtr = &item
			break
		}
	}

	if itemPtr != nil {
		itemPtr.Quantity++
	} else {
		prod, err := db.FindCatalogProductByID(productID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to find product"))
			log.Printf("failed to find product: %v\n", err)
			return
		}

		cartItem := db.CartItem{
			ProductID: productID,
			Source:    source,
			Name:      prod.Name,
			Category:  prod.Category,
			ImageURL:  prod.ImageURL,
			Price:     prod.Price,
			Quantity:  1,
		}

		cart = append(cart, cartItem)
	}

	total := 0
	for _, item := range cart {
		total += item.Quantity * item.Price
	}

	err = components.CartSidebar(cart, total).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render CartSidebar err: %v\n", err)
	}
}

func UpdateCartQuantity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	action := r.URL.Query().Get("action")
	var item *db.CartItem

	for i, cartItem := range cart {
		if cartItem.ProductID == id {
			item = &cart[i]
			break
		}
	}

	switch action {
	case "increase":
		item.Quantity++
	case "decrease":
		item.Quantity--
	}

	w.Header().Add("HX-Trigger", "cart-sidebar")
	w.Header().Add("HX-Target", "cart-sidebar")
	w.Header().Add("HX-Swap", "innerHTML")
	w.WriteHeader(http.StatusOK)
}

func RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	newCart := make([]db.CartItem, 0, len(cart))
	for _, item := range cart {
		if item.ProductID != id {
			newCart = append(newCart, item)
		}
	}
	cart = newCart

	total := 0
	for _, item := range cart {
		total += item.Quantity * item.Price
	}

	err := components.CartSidebar(cart, total).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render CartSidebar err: %v\n", err)
	}
}

func ClearCart(w http.ResponseWriter, r *http.Request) {
	cart = make([]db.CartItem, 0)
	total := 0

	err := components.CartSidebar(cart, total).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render CartSidebar err: %v\n", err)
	}
}
