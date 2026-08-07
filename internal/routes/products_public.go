package routes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/vladwithcode/salon_catalog/internal/db"
)

// publicProductDTO is the only shape GET /api/products/{slug} — a public,
// unauthenticated legacy route (internal/routes/products.go) — is allowed
// to serialize. It exists so this handler never marshals db.Product
// directly: that struct carries admin-only fields (MainImgID, GalleryIDs,
// QRCodeFilename) that must never reach an unauthenticated caller. The
// field list here matches the "público: sí" column of 06-product-page.md.
type publicProductDTO struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	Description     string            `json:"description"`
	LongDescription string            `json:"long_description"`
	Category        publicCategoryDTO `json:"category"`
	Available       bool              `json:"available"`
	Quantity        int               `json:"quantity"`
	ImageFilename   string            `json:"image_filename"`
	Images          []string          `json:"images"`
}

type publicCategoryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// newPublicProductDTO copies only the public fields out of db.Product.
// product.MainImg and product.Gallery are already plain image filenames
// (see internal/db/products.go's FindProductBySlug query, which aliases
// main.filename AS main_img) — not the UUID columns, which live in the
// separate MainImgID/GalleryIDs fields this DTO never reads.
func newPublicProductDTO(product *db.Product) publicProductDTO {
	images := product.Gallery
	if images == nil {
		images = []string{}
	}
	return publicProductDTO{
		ID:              product.ID,
		Name:            product.Name,
		Slug:            product.Slug,
		Description:     product.Description,
		LongDescription: product.LongDescription,
		Category:        publicCategoryDTO{ID: product.CategoryID, Name: product.Category},
		Available:       product.Available,
		Quantity:        product.Quantity,
		ImageFilename:   product.MainImg,
		Images:          images,
	}
}

// productBySlugLoader is the narrow dependency GetProductBySlug needs from
// internal/db: a single lookup by slug. It exists so tests can substitute a
// fake without a database — the same pattern internal/routes/cart_forms.go
// already uses for cartFormMutationService.
type productBySlugLoader func(slug string) (*db.Product, error)

// GetProductBySlug is the heavily legacy, unauthenticated
// GET /api/products/{slug} route. It is not the new 06B2 public-detail
// contract — it is kept alive only because no internal consumer of it was
// found (see 6B1's delivery report), and removing it outright was out of
// scope for this phase. Its response now goes through publicProductDTO
// instead of serializing db.Product directly.
func GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	getProductBySlugHandler(db.FindProductBySlug)(w, r)
}

func getProductBySlugHandler(loadProduct productBySlugLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		product, err := loadProduct(slug)
		if err != nil {
			writePublicProductError(w, slug, err)
			return
		}

		data, err := json.Marshal(newPublicProductDTO(product))
		if err != nil {
			log.Printf("failed to marshal public product dto for slug %q: %v\n", slug, err)
			writePublicProductJSONError(w, http.StatusInternalServerError, "product_unavailable")
			return
		}

		setPublicProductJSONHeaders(w)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

func writePublicProductError(w http.ResponseWriter, slug string, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writePublicProductJSONError(w, http.StatusNotFound, "product_not_found")
		return
	}
	log.Printf("failed to load product by slug %q: %v\n", slug, err)
	writePublicProductJSONError(w, http.StatusInternalServerError, "product_unavailable")
}

func setPublicProductJSONHeaders(w http.ResponseWriter) {
	header := w.Header()
	header.Set("Content-Type", "application/json; charset=utf-8")
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
}

func writePublicProductJSONError(w http.ResponseWriter, status int, code string) {
	setPublicProductJSONHeaders(w)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
