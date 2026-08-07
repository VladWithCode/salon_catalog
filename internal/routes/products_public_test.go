package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/vladwithcode/salon_catalog/internal/db"
)

func fakeAdminProduct(available bool) *db.Product {
	return &db.Product{
		ID:              "01890f3a-dc02-7cb5-a4cc-451231879f0b",
		Name:            "Mesa Redonda",
		Slug:            "mesa-redonda",
		Description:     "Una mesa redonda",
		LongDescription: "Una mesa redonda de madera",
		MainImg:         "mesa-redonda.jpg",
		MainImgID:       "22222222-2222-2222-2222-222222222222",
		Gallery:         []string{"mesa-redonda-1.jpg", "mesa-redonda-2.jpg"},
		GalleryIDs:      []string{"33333333-3333-3333-3333-333333333333"},
		Category:        "Mobiliario",
		CategoryID:      "44444444-4444-4444-4444-444444444444",
		Available:       available,
		Quantity:        3,
		QRCodeFilename:  "mesa-redonda-qrcode.jpeg",
	}
}

func TestGetProductBySlugValidProduct(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return fakeAdminProduct(true), nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetProductBySlugNotFound(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return nil, pgx.ErrNoRows }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/no-existe", nil)
	request.SetPathValue("slug", "no-existe")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestGetProductBySlugUnavailableProductStillReturned(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return fakeAdminProduct(false), nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for an unavailable-but-existing product, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"available":false`) {
		t.Fatalf("expected available:false in the response, got: %s", recorder.Body.String())
	}
}

func TestGetProductBySlugWithoutImage(t *testing.T) {
	product := fakeAdminProduct(true)
	product.MainImg = ""
	product.Gallery = nil
	loader := func(slug string) (*db.Product, error) { return product, nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"image_filename":""`) {
		t.Fatalf("expected empty image_filename, got: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"images":[]`) {
		t.Fatalf("expected an empty images array, not null, got: %s", recorder.Body.String())
	}
}

func TestGetProductBySlugMainImage(t *testing.T) {
	product := fakeAdminProduct(true)
	loader := func(slug string) (*db.Product, error) { return product, nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if !strings.Contains(recorder.Body.String(), `"image_filename":"mesa-redonda.jpg"`) {
		t.Fatalf("expected the main image filename in the response, got: %s", recorder.Body.String())
	}
}

func TestGetProductBySlugGalleryFilenamesOnly(t *testing.T) {
	product := fakeAdminProduct(true)
	loader := func(slug string) (*db.Product, error) { return product, nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "mesa-redonda-1.jpg") || !strings.Contains(body, "mesa-redonda-2.jpg") {
		t.Fatalf("expected gallery filenames in the response, got: %s", body)
	}
	if strings.Contains(body, product.GalleryIDs[0]) {
		t.Fatalf("gallery image UUIDs must never appear in the public response, got: %s", body)
	}
}

func TestGetProductBySlugDatabaseErrorDoesNotLeakDetails(t *testing.T) {
	loader := func(slug string) (*db.Product, error) {
		return nil, errors.New("dial tcp 10.0.0.5:5432: password authentication failed")
	}
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, secret := range []string{"password", "10.0.0.5", "5432", "dial tcp"} {
		if strings.Contains(body, secret) {
			t.Fatalf("internal detail %q leaked into the response: %s", secret, body)
		}
	}
}

func TestGetProductBySlugResponseIsValidJSON(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return fakeAdminProduct(true), nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	var decoded publicProductDTO
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid JSON matching publicProductDTO: %v (%s)", err, recorder.Body.String())
	}
	if decoded.Slug != "mesa-redonda" {
		t.Fatalf("expected slug to round-trip, got %q", decoded.Slug)
	}
}

func TestGetProductBySlugContentType(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return fakeAdminProduct(true), nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("expected application/json; charset=utf-8, got %q", got)
	}
}

func TestGetProductBySlugCacheControl(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return fakeAdminProduct(true), nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %q", got)
	}
}

func TestGetProductBySlugNosniff(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return fakeAdminProduct(true), nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff, got %q", got)
	}
}

func TestGetProductBySlugExcludesAdministrativeFields(t *testing.T) {
	loader := func(slug string) (*db.Product, error) { return fakeAdminProduct(true), nil }
	handler := getProductBySlugHandler(loader)

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	request.SetPathValue("slug", "mesa-redonda")
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	body := recorder.Body.String()

	forbidden := map[string]string{
		"search_vector field":     "search_vector",
		"qrcode_filename field":   "qrcode_filename",
		"qrcode_filename value":   "mesa-redonda-qrcode.jpeg",
		"main image UUID field":   "mainImgId",
		"main image UUID value":   "22222222-2222-2222-2222-222222222222",
		"gallery UUID field":      "galleryIds",
		"gallery UUID value":      "33333333-3333-3333-3333-333333333333",
		"filesystem/static path":  "/static/uploads",
		"web root path":           "web/static",
		"internal category field": "categoryId", // public field is "category":{"id":...}, not this camelCase admin key
	}
	for label, needle := range forbidden {
		if strings.Contains(body, needle) {
			t.Fatalf("response must not contain %s (%q), got: %s", label, needle, body)
		}
	}
}

func TestGetProductBySlugOnlyGETReachesHandler(t *testing.T) {
	router := NewCustomServeMux()
	router.HandleFunc("GET /api/products/{slug}", getProductBySlugHandler(func(slug string) (*db.Product, error) {
		return fakeAdminProduct(true), nil
	}))

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/api/products/mesa-redonda", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code == http.StatusOK {
				t.Fatalf("%s must not reach the GET-only handler, got 200", method)
			}
		})
	}
}

func TestGetProductBySlugGETIsRegistered(t *testing.T) {
	router := NewCustomServeMux()
	router.HandleFunc("GET /api/products/{slug}", getProductBySlugHandler(func(slug string) (*db.Product, error) {
		return fakeAdminProduct(true), nil
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/products/mesa-redonda", nil)
	_, pattern := router.Handler(request)
	if pattern != "GET /api/products/{slug}" {
		t.Fatalf("expected GET /api/products/{slug} to match, got %q", pattern)
	}
}
