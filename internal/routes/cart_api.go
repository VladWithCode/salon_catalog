package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

type cartAPIItem struct {
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	ImageFilename string `json:"image_filename"`
	Quantity      int    `json:"quantity"`
	MaxQuantity   int    `json:"max_quantity"`
	Available     bool   `json:"available"`
}

type cartAPIState struct {
	Items      []cartAPIItem `json:"items"`
	TotalItems int           `json:"total_items"`
}

type cartAPIResponse struct {
	Cart cartAPIState `json:"cart"`
}

type cartAPIErrorResponse struct {
	Error string `json:"error"`
}

type cartAPIDataLoader interface {
	FindCartByID(context.Context, string) (*db.Cart, error)
	FindCatalogProductDetail(string) (*db.CatalogProd, error)
}

type databaseCartAPIDataLoader struct{}

func (databaseCartAPIDataLoader) FindCartByID(ctx context.Context, cartID string) (*db.Cart, error) {
	return db.FindCartByID(ctx, cartID)
}

func (databaseCartAPIDataLoader) FindCatalogProductDetail(productID string) (*db.CatalogProd, error) {
	return db.FindCatalogProductDetail(productID)
}

func RegisterCartAPIRoutes(router *customServeMux, cartSessions *session.CartManager) {
	registerCartAPIRoutes(router, cartSessions, databaseCartAPIDataLoader{})
}

func registerCartAPIRoutes(router *customServeMux, cartSessions *session.CartManager, loader cartAPIDataLoader) {
	router.HandleFunc("GET /api/cart", withCartSession(cartSessions, getCartAPIHandler(loader)))
}

func getCartAPIHandler(loader cartAPIDataLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cartID, err := cartIDFromRequestContext(r)
		if err != nil {
			writeCartAPIJSON(w, http.StatusInternalServerError, cartAPIErrorResponse{Error: "cart_unavailable"})
			return
		}

		response, err := loadCartAPIResponse(r.Context(), loader, cartID)
		if err != nil {
			writeCartAPIJSON(w, http.StatusServiceUnavailable, cartAPIErrorResponse{Error: "cart_unavailable"})
			return
		}
		writeCartAPIJSON(w, http.StatusOK, response)
	}
}

func loadCartAPIResponse(ctx context.Context, loader cartAPIDataLoader, cartID string) (cartAPIResponse, error) {
	cart, err := loader.FindCartByID(ctx, cartID)
	if errors.Is(err, db.ErrCartNotFound) {
		return emptyCartAPIResponse(), nil
	}
	if err != nil || cart == nil {
		return cartAPIResponse{}, errCartSessionUnavailable
	}

	items := make([]cartAPIItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if item == nil {
			continue
		}

		product, productErr := loader.FindCatalogProductDetail(item.ProductID)
		if productErr != nil || product == nil {
			return cartAPIResponse{}, errCartSessionUnavailable
		}

		items = append(items, cartAPIItem{
			ProductID:     product.ID,
			Name:          product.Name,
			Slug:          product.Slug,
			ImageFilename: strings.TrimSpace(product.ImageURL),
			Quantity:      item.Quantity,
			MaxQuantity:   product.Quantity,
			Available:     product.Available && product.Quantity > 0,
		})
	}

	return cartAPIResponse{
		Cart: cartAPIState{
			Items:      items,
			TotalItems: len(items),
		},
	}, nil
}

func emptyCartAPIResponse() cartAPIResponse {
	return cartAPIResponse{
		Cart: cartAPIState{
			Items:      make([]cartAPIItem, 0),
			TotalItems: 0,
		},
	}
}

func writeCartAPIJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
