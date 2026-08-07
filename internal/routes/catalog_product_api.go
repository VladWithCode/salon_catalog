package routes

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/vladwithcode/salon_catalog/internal/db"
)

// catalogProductIdentifierMaxLength mirrors products.slug's real column
// definition — VARCHAR(200) — from
// sql/migrations/20250703200655_add_products_table.sql:4. A UUID (36
// characters) is always far under this limit, so the same cap applies to
// both identifier shapes without special-casing either.
const catalogProductIdentifierMaxLength = 200

// RegisterCatalogProductAPIRoutes registers the single, minimal, read-only
// public detail endpoint 6B2 adds. It is deliberately not folded into
// publicMiddleware (which fetches social links) or any cart-session
// middleware: this route needs neither, and adding either would create a
// cart session or issue a cookie just for a product lookup — forbidden by
// this phase.
func RegisterCatalogProductAPIRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/catalog/products/{identifier}", GetPublicAPICatalogProductDetail)
}

// publicAPICatalogProductDetail is the only shape
// GET /api/catalog/products/{identifier} ever serializes. It is a new,
// separate contract from publicProductDTO (the 6B1 DTO for the legacy
// GET /api/products/{slug}) — the two are not merged, per this phase's
// instructions, even though both exist to keep admin fields out of a
// public response.
type publicAPICatalogProductDetail struct {
	ID              string                       `json:"id"`
	Name            string                       `json:"name"`
	Slug            string                       `json:"slug"`
	Description     string                       `json:"description"`
	LongDescription string                       `json:"long_description"`
	Category        *publicAPICatalogCategoryRef `json:"category"`
	Available       bool                         `json:"available"`
	ImageFilename   *string                      `json:"image_filename"`
	Images          []string                     `json:"images"`
}

type publicAPICatalogCategoryRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type publicAPICatalogProductDetailResponse struct {
	Product publicAPICatalogProductDetail `json:"product"`
}

// catalogProductDetailAPILoader is the narrow dependency this handler needs
// from internal/db: a single lookup by an already-validated identifier
// (UUID or slug). Production always binds it to db.FindCatalogProductDetail
// — the one resolver internal/db/catalog.go implements
// (catalogProductIdentifierKind) — never a second implementation.
type catalogProductDetailAPILoader func(identifier string) (*db.CatalogProd, error)

func GetPublicAPICatalogProductDetail(w http.ResponseWriter, r *http.Request) {
	getPublicAPICatalogProductDetailHandler(db.FindCatalogProductDetail)(w, r)
}

func getPublicAPICatalogProductDetailHandler(loader catalogProductDetailAPILoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier := r.PathValue("identifier")
		if !isValidCatalogProductIdentifier(identifier) {
			writePublicAPICatalogJSON(w, http.StatusBadRequest, publicAPIErrorResponse{
				Error: "invalid_identifier",
			})
			return
		}

		product, err := loader(identifier)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writePublicAPICatalogJSON(w, http.StatusNotFound, publicAPIErrorResponse{
					Error: "product_not_found",
				})
				return
			}
			// Any other error — a real database failure — must never be
			// distinguished from "not found" by inspecting its text, and
			// must never leak driver/SQL detail. "catalog_unavailable" is
			// the same code already used by every other catalog endpoint's
			// database-failure branch (GetPublicAPICatalogListings,
			// GetPublicAPICatalogCategories, GetPublicAPICatalogProducts).
			writePublicAPICatalogJSON(w, http.StatusServiceUnavailable, publicAPIErrorResponse{
				Error: "catalog_unavailable",
			})
			return
		}

		if !isValidCatalogProductInternalState(product) {
			// A product that violates its own not-null/unique constraints
			// (empty name, empty id, empty slug) cannot be represented in
			// this public contract. This is never expected from real data —
			// products.name, .id and .slug are all NOT NULL — but if it
			// ever happened, it must not be silently "corrected" or half
			// exposed; the same generic unavailability response is used
			// rather than inventing a new error code for a state that
			// should be structurally impossible.
			writePublicAPICatalogJSON(w, http.StatusServiceUnavailable, publicAPIErrorResponse{
				Error: "catalog_unavailable",
			})
			return
		}

		writePublicAPICatalogJSON(w, http.StatusOK, publicAPICatalogProductDetailResponse{
			Product: newPublicAPICatalogProductDetail(product),
		})
	}
}

// isValidCatalogProductIdentifier rejects, before the loader is ever
// called, anything that cannot possibly be a legitimate UUID or slug: an
// empty value (after trimming only to detect emptiness — the original,
// untrimmed value is still what reaches the loader on success), NUL, other
// control bytes, path separators that could never appear in a
// products.slug value generated by the admin panel's plain text field, and
// anything longer than products.slug's own VARCHAR(200) column definition
// (sql/migrations/20250703200655_add_products_table.sql:4). The length is
// measured in Unicode characters via utf8.RuneCountInString, not
// len(identifier) (which counts bytes and would undercount the limit for
// any multi-byte character, rejecting valid 200-character slugs too early
// or accepting byte-equivalent strings the database would reject).
func isValidCatalogProductIdentifier(identifier string) bool {
	if strings.TrimSpace(identifier) == "" {
		return false
	}
	if strings.ContainsRune(identifier, '/') || strings.ContainsRune(identifier, '\\') {
		return false
	}
	for _, r := range identifier {
		if r == 0 || r < 0x20 || r == 0x7f {
			return false
		}
	}
	if utf8.RuneCountInString(identifier) > catalogProductIdentifierMaxLength {
		return false
	}
	return true
}

// isValidCatalogProductInternalState guards the structurally-impossible
// case described above the call site.
func isValidCatalogProductInternalState(product *db.CatalogProd) bool {
	return product != nil && product.ID != "" && product.Name != "" && product.Slug != ""
}

// catalogProductImageFilenamePattern mirrors the safe-filename contract
// this codebase already applies at the edge for served images
// (frontend/app/api/catalog/media/[filename]/route.ts uses
// ^[\p{L}\p{N}._:-]+$; this Go-side check uses the same ASCII-only shape
// already established for other generated identifiers in this codebase).
// It exists purely to decide whether a filename is safe to expose in this
// JSON response — it never touches the filesystem, never opens a file, and
// never constructs a path or URL.
var catalogProductImageFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

func isSafeCatalogProductImageFilename(filename string) bool {
	if filename == "" || filename == "." || filename == ".." {
		return false
	}
	return catalogProductImageFilenamePattern.MatchString(filename)
}

// newPublicAPICatalogProductDetail builds the public response from
// db.CatalogProd (internal/db/catalog.go), the same struct
// FindCatalogProductDetail already returns — no second query, no admin
// struct involved. product.ImageURL and product.Images are already plain
// filenames (the catalog_products view resolves them from images.filename,
// never a UUID or a path), so this function only ever validates and
// filters, never derives a path or host.
func newPublicAPICatalogProductDetail(product *db.CatalogProd) publicAPICatalogProductDetail {
	var category *publicAPICatalogCategoryRef
	if product.CategoryID != "" {
		category = &publicAPICatalogCategoryRef{ID: product.CategoryID, Name: product.CategoryName}
	}

	var imageFilename *string
	if isSafeCatalogProductImageFilename(product.ImageURL) {
		filename := product.ImageURL
		imageFilename = &filename
	}

	images := make([]string, 0, len(product.Images))
	for _, filename := range product.Images {
		if isSafeCatalogProductImageFilename(filename) {
			images = append(images, filename)
		}
	}

	return publicAPICatalogProductDetail{
		ID:              product.ID,
		Name:            product.Name,
		Slug:            product.Slug,
		Description:     product.Description,
		LongDescription: product.LongDescription,
		Category:        category,
		Available:       product.Available,
		ImageFilename:   imageFilename,
		Images:          images,
	}
}
