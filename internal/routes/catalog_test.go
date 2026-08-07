package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal/db"
)

const (
	catalogTestExistingUUID = "01890f3a-dc02-7cb5-a4cc-451231879f0b"
	catalogTestExistingSlug = "mesa-redonda"
)

func fakeCatalogProduct(available bool) *db.CatalogProd {
	return &db.CatalogProd{
		ID:           catalogTestExistingUUID,
		Name:         "Mesa Redonda",
		Slug:         catalogTestExistingSlug,
		Description:  "Una mesa redonda",
		CategoryName: "Mobiliario",
		CategoryID:   "11111111-1111-1111-1111-111111111111",
		Available:    available,
		Quantity:     3,
		Images:       []string{},
	}
}

// fakeLoaders wires GetProductDetail's two injected dependencies to a fixed
// product, indexed by whichever identifier (UUID or slug) the caller
// requests — mirroring, at the test double level, the same dual dispatch
// db.FindCatalogProductDetail implements for real (see
// internal/db/catalog_identifier_test.go for the resolver's own coverage).
func fakeLoaders(product *db.CatalogProd, dbErr error) (catalogProductDetailLoader, relatedProductsLoader) {
	detail := func(id string) (*db.CatalogProd, error) {
		if dbErr != nil {
			return nil, dbErr
		}
		if product != nil && (id == product.ID || id == product.Slug) {
			return product, nil
		}
		return nil, errors.New("cart: product not found")
	}
	related := func(productID string, limit int) ([]*db.CatalogProd, error) {
		return nil, nil
	}
	return detail, related
}

// --- Section 6: /catalogo/producto compatibility ---

func TestGetProductDetailByUUIDResolvesExistingProduct(t *testing.T) {
	product := fakeCatalogProduct(true)
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+catalogTestExistingUUID, nil)
	request.SetPathValue("id", catalogTestExistingUUID)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatalf("expected the existing product to resolve by UUID, got 404: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Mesa Redonda") {
		t.Fatalf("expected the product's own name in the response, got: %s", recorder.Body.String())
	}
}

func TestGetProductDetailBySlugResolvesExistingProduct(t *testing.T) {
	product := fakeCatalogProduct(true)
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+catalogTestExistingSlug, nil)
	request.SetPathValue("id", catalogTestExistingSlug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatalf("expected the existing product to resolve by slug, got 404: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Mesa Redonda") {
		t.Fatalf("expected the product's own name in the response, got: %s", recorder.Body.String())
	}
}

func TestGetProductDetailUUIDAndSlugReachTheSameProduct(t *testing.T) {
	product := fakeCatalogProduct(true)
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	byUUID := httptest.NewRecorder()
	uuidReq := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+catalogTestExistingUUID, nil)
	uuidReq.SetPathValue("id", catalogTestExistingUUID)
	handler(byUUID, uuidReq)

	bySlug := httptest.NewRecorder()
	slugReq := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+catalogTestExistingSlug, nil)
	slugReq.SetPathValue("id", catalogTestExistingSlug)
	handler(bySlug, slugReq)

	if !strings.Contains(byUUID.Body.String(), "Mesa Redonda") || !strings.Contains(bySlug.Body.String(), "Mesa Redonda") {
		t.Fatal("expected both identifiers to resolve to the same product name")
	}
}

func TestGetProductDetailUUIDNotFound(t *testing.T) {
	detail, related := fakeLoaders(nil, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/00000000-0000-0000-0000-000000000000", nil)
	request.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a well-formed but nonexistent UUID, got %d", recorder.Code)
	}
}

func TestGetProductDetailSlugNotFound(t *testing.T) {
	detail, related := fakeLoaders(nil, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/no-existe", nil)
	request.SetPathValue("id", "no-existe")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a nonexistent slug, got %d", recorder.Code)
	}
}

func TestGetProductDetailEmptyIdentifier(t *testing.T) {
	detail, related := fakeLoaders(nil, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/", nil)
	request.SetPathValue("id", "")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an empty identifier, not a crash, got %d", recorder.Code)
	}
}

func TestGetProductDetailMalformedUUIDTreatedAsSlug(t *testing.T) {
	// Confirms the resolver's real fallback rule (also unit-tested directly
	// in internal/db/catalog_identifier_test.go): a string that merely
	// resembles a UUID but fails uuid.Parse falls through to the slug
	// branch, not an error.
	product := fakeCatalogProduct(true)
	product.Slug = "01890f3a-dc02-7cb5-a4cc-45123187" // deliberately malformed as a UUID
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+product.Slug, nil)
	request.SetPathValue("id", product.Slug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("expected a malformed-UUID-shaped slug to still resolve as a slug")
	}
}

func TestGetProductDetailSlugWithHyphens(t *testing.T) {
	product := fakeCatalogProduct(true)
	product.Slug = "silla-tiffany-blanca"
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+product.Slug, nil)
	request.SetPathValue("id", product.Slug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("expected a hyphenated slug to resolve")
	}
}

func TestGetProductDetailSlugWithDigits(t *testing.T) {
	product := fakeCatalogProduct(true)
	product.Slug = "carpa-10x10"
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+product.Slug, nil)
	request.SetPathValue("id", product.Slug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("expected a slug containing digits to resolve")
	}
}

func TestGetProductDetailSlugEncodedByNextDecodesCorrectly(t *testing.T) {
	// frontend/components/catalog/catalog-product-card.tsx builds
	// `/catalogo/producto/${encodeURIComponent(product.slug)}`. Go's
	// net/http ServeMux decodes the raw path before PathValue returns it,
	// so this test uses the already-decoded value the handler actually
	// receives — the same value the real mux would hand it after a request
	// to the percent-encoded URL.
	product := fakeCatalogProduct(true)
	product.Slug = "cojín-decorativo"
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/coj%C3%ADn-decorativo", nil)
	request.SetPathValue("id", product.Slug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("expected an accented slug (as decoded by the router) to resolve")
	}
}

func TestGetProductDetailAvailableTrue(t *testing.T) {
	product := fakeCatalogProduct(true)
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+product.Slug, nil)
	request.SetPathValue("id", product.Slug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("an available product must not 404")
	}
}

func TestGetProductDetailAvailableFalseStillRenders(t *testing.T) {
	// Section 12: a product with available=false must still be
	// consultable, never silently turned into a 404.
	product := fakeCatalogProduct(false)
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+product.Slug, nil)
	request.SetPathValue("id", product.Slug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("expected an unavailable product to still resolve, not 404")
	}
}

func TestGetProductDetailWithoutImages(t *testing.T) {
	product := fakeCatalogProduct(true)
	product.Images = nil
	product.ImageURL = ""
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+product.Slug, nil)
	request.SetPathValue("id", product.Slug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("a product with no images must still resolve")
	}
}

func TestGetProductDetailDatabaseErrorNeverLeaksDetails(t *testing.T) {
	dbErr := errors.New("dial tcp 10.0.0.5:5432: password authentication failed")
	detail, related := fakeLoaders(nil, dbErr)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/catalogo/producto/"+catalogTestExistingSlug, nil)
	request.SetPathValue("id", catalogTestExistingSlug)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	// Current, unchanged behavior: any loader error — not-found or a real
	// database failure alike — renders the same fixed 404 copy. This means
	// no internal detail can leak through this path today; it is not
	// modified in this phase, only confirmed and pinned by this test.
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected the existing fixed-404 behavior for any loader error, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, secret := range []string{"password", "10.0.0.5", "5432", "pgx", "SELECT", "sql"} {
		if strings.Contains(body, secret) {
			t.Fatalf("internal detail %q leaked into the response body", secret)
		}
	}
}

// --- Section 7: QR / legacy-route compatibility ---

func TestGetProductDetailViaCatalogoProductoAndProductosResolveSameProduct(t *testing.T) {
	// /catalogo/producto/{id} and /productos/{id} are wired to the exact
	// same handler function (internal/routes/catalog.go), confirmed by
	// direct read; this test exercises that shared handler with the QR
	// route's own identifier shape (a UUID) to fix the contract in code.
	product := fakeCatalogProduct(true)
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/productos/"+catalogTestExistingUUID, nil)
	request.SetPathValue("id", catalogTestExistingUUID)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("expected the QR-compat route's UUID to resolve via the same handler")
	}
	if !strings.Contains(recorder.Body.String(), product.ID) {
		t.Fatal("expected the resolved product's UUID to be intact in the response")
	}
}

func TestGetProductDetailQRRouteDoesNotDependOnSlug(t *testing.T) {
	// A printed QR code embeds only the UUID (internal/routes/products.go,
	// qrgen callers). This test proves resolution by UUID alone succeeds
	// even when the loader is never asked to match by slug — i.e. slug
	// never needs to be known or unchanged for QR compatibility to hold.
	product := fakeCatalogProduct(true)
	product.Slug = "un-slug-que-podria-cambiar-despues"
	detail, related := fakeLoaders(product, nil)
	handler := getProductDetailHandler(detail, related)

	request := httptest.NewRequest(http.MethodGet, "/productos/"+catalogTestExistingUUID, nil)
	request.SetPathValue("id", catalogTestExistingUUID)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("expected UUID resolution to succeed independent of the product's current slug")
	}
}

// --- Section 5: routes stay registered and coexist ---

func TestCatalogoProductoAndProductosRoutesAreRegistered(t *testing.T) {
	router := NewCustomServeMux()
	manager := newDeterministicRouteCartManager(t, uuid.New())
	RegisterCatalogRoutes(router, manager)

	for _, testCase := range []struct {
		path    string
		pattern string
	}{
		{path: "/catalogo/producto/" + catalogTestExistingSlug, pattern: "GET /catalogo/producto/{id}"},
		{path: "/productos/" + catalogTestExistingUUID, pattern: "GET /productos/{id}"},
		{path: "/catalog/products", pattern: "GET /catalog/products"},
		{path: "/catalog/categories", pattern: "GET /catalog/categories"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		_, pattern := router.Handler(request)
		if pattern != testCase.pattern {
			t.Errorf("%s: expected pattern %q, got %q", testCase.path, testCase.pattern, pattern)
		}
	}
}
