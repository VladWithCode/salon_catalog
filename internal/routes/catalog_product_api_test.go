package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vladwithcode/salon_catalog/internal/db"
)

const (
	productAPITestUUID = "01890f3a-dc02-7cb5-a4cc-451231879f0b"
	productAPITestSlug = "mesa-redonda"
)

func fakeProductAPICatalogProd(available bool) *db.CatalogProd {
	return &db.CatalogProd{
		ID:           productAPITestUUID,
		Name:         "Mesa Redonda",
		Slug:         productAPITestSlug,
		Description:  "Una mesa redonda",
		CategoryName: "Mobiliario",
		CategoryID:   "44444444-4444-4444-4444-444444444444",
		Available:    available,
		Quantity:     3,
		ImageURL:     "mesa-redonda.jpg",
		Images:       []string{"mesa-redonda-1.jpg", "mesa-redonda-2.jpg"},
	}
}

func newProductAPILoader(product *db.CatalogProd, err error) (catalogProductDetailAPILoader, *int) {
	calls := 0
	loader := func(identifier string) (*db.CatalogProd, error) {
		calls++
		if err != nil {
			return nil, err
		}
		return product, nil
	}
	return loader, &calls
}

func doProductAPIRequest(handler http.HandlerFunc, identifier string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/catalog/products/"+identifier, nil)
	request.SetPathValue("identifier", identifier)
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

func mustDecodeProductAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder, out *publicAPICatalogProductDetailResponse) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), out); err != nil {
		t.Fatalf("failed to decode response: %v (%s)", err, recorder.Body.String())
	}
}

// --- Section 20: identifier validation ---

func TestCatalogProductAPIIdentifierValidation(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		identifier string
		wantValid  bool
	}{
		{name: "valid UUID", identifier: productAPITestUUID, wantValid: true},
		{name: "valid slug", identifier: "mesa-redonda", wantValid: true},
		{name: "slug with hyphen", identifier: "silla-tiffany", wantValid: true},
		{name: "slug with numbers", identifier: "carpa-10x10", wantValid: true},
		{name: "slug with accents", identifier: "cojín-decorativo", wantValid: true},
		{name: "empty", identifier: "", wantValid: false},
		{name: "only spaces", identifier: "   ", wantValid: false},
		{name: "NUL byte", identifier: "mesa\x00redonda", wantValid: false},
		{name: "control character", identifier: "mesa\x01redonda", wantValid: false},
		{name: "slash", identifier: "mesa/redonda", wantValid: false},
		{name: "backslash", identifier: "mesa\\redonda", wantValid: false},
		{name: "malformed uuid-shaped string is still a valid slug shape", identifier: "01890f3a-dc02-7cb5-a4cc-45123187", wantValid: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isValidCatalogProductIdentifier(testCase.identifier); got != testCase.wantValid {
				t.Fatalf("isValidCatalogProductIdentifier(%q) = %v, want %v", testCase.identifier, got, testCase.wantValid)
			}
		})
	}
}

// --- Section 6B2A: 200-character Unicode length limit ---

func TestCatalogProductAPIIdentifierLengthLimit(t *testing.T) {
	// "é" is 2 bytes in UTF-8 but 1 rune — used to prove the limit is
	// measured in characters, not bytes.
	for _, testCase := range []struct {
		name       string
		identifier string
		wantValid  bool
	}{
		{name: "ASCII 199 chars accepted", identifier: strings.Repeat("a", 199), wantValid: true},
		{name: "ASCII 200 chars accepted", identifier: strings.Repeat("a", 200), wantValid: true},
		{name: "ASCII 201 chars rejected", identifier: strings.Repeat("a", 201), wantValid: false},
		{name: "unicode 200 chars accepted", identifier: strings.Repeat("é", 200), wantValid: true},
		{name: "unicode 201 chars rejected", identifier: strings.Repeat("é", 201), wantValid: false},
		{name: "valid UUID still accepted", identifier: productAPITestUUID, wantValid: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isValidCatalogProductIdentifier(testCase.identifier); got != testCase.wantValid {
				t.Fatalf("isValidCatalogProductIdentifier(len=%d runes) = %v, want %v", utf8.RuneCountInString(testCase.identifier), got, testCase.wantValid)
			}
		})
	}
}

func TestCatalogProductAPIIdentifierLengthCountsCharactersNotBytes(t *testing.T) {
	// 200 "é" runes is 400 bytes but exactly 200 characters: must be
	// accepted. This is the case len(identifier) would get wrong.
	identifier := strings.Repeat("é", 200)
	if byteLen := len(identifier); byteLen <= catalogProductIdentifierMaxLength {
		t.Fatalf("test setup invalid: expected byte length > %d to prove the distinction, got %d", catalogProductIdentifierMaxLength, byteLen)
	}
	if !isValidCatalogProductIdentifier(identifier) {
		t.Fatal("expected a 200-rune (400-byte) identifier to be accepted — length must be measured in characters, not bytes")
	}
}

func TestCatalogProductAPITooLongIdentifierRejectedBeforeLoader(t *testing.T) {
	loader, calls := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	tooLong := strings.Repeat("a", 201)
	recorder := doProductAPIRequest(handler, tooLong)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "invalid_identifier") {
		t.Fatalf("expected invalid_identifier, got: %s", recorder.Body.String())
	}
	if *calls != 0 {
		t.Fatalf("expected zero loader calls for a too-long identifier, got %d", *calls)
	}
}

func TestCatalogProductAPITooLongIdentifierResponseHeadersAndNoLeak(t *testing.T) {
	loader, _ := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	tooLong := strings.Repeat("a", 201)
	recorder := doProductAPIRequest(handler, tooLong)

	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: expected application/json; charset=utf-8, got %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control: expected no-store, got %q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options: expected nosniff, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header, got %q", got)
	}
	if len(recorder.Header().Values("Set-Cookie")) != 0 {
		t.Error("expected no Set-Cookie")
	}
	if strings.Contains(recorder.Body.String(), tooLong) {
		t.Fatal("the too-long identifier itself must never be reflected back in the response")
	}
}

func TestCatalogProductAPISlugURLDecodedByRouterResolves(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.Slug = "cojín-decorativo"
	loader, calls := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	// The router decodes %-escapes before PathValue returns them; this
	// test uses the already-decoded value, matching what the real mux
	// hands the handler for a request to /api/catalog/products/coj%C3%ADn-decorativo.
	recorder := doProductAPIRequest(handler, product.Slug)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if *calls != 1 {
		t.Fatalf("expected exactly one loader call, got %d", *calls)
	}
}

func TestCatalogProductAPIInvalidIdentifierNeverCallsLoader(t *testing.T) {
	loader, calls := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, "mesa/redonda")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if *calls != 0 {
		t.Fatalf("expected zero loader calls for an invalid identifier, got %d", *calls)
	}
}

func TestCatalogProductAPIValidIdentifierCallsLoaderOnce(t *testing.T) {
	loader, calls := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	doProductAPIRequest(handler, productAPITestSlug)

	if *calls != 1 {
		t.Fatalf("expected exactly one loader call, got %d", *calls)
	}
}

// --- Section 21: 200 response contract ---

func TestCatalogProductAPIByUUID(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestUUID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCatalogProductAPIBySlug(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCatalogProductAPIUUIDAndSlugSameProduct(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	byID := doProductAPIRequest(handler, productAPITestUUID)
	bySlug := doProductAPIRequest(handler, productAPITestSlug)

	if byID.Body.String() != bySlug.Body.String() {
		t.Fatalf("expected identical bodies for the same product resolved by UUID and by slug:\n%s\nvs\n%s", byID.Body.String(), bySlug.Body.String())
	}
}

func TestCatalogProductAPIAvailableTrue(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if !decoded.Product.Available {
		t.Fatal("expected available:true")
	}
}

func TestCatalogProductAPIAvailableFalseStill200(t *testing.T) {
	product := fakeProductAPICatalogProd(false)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for available=false, got %d", recorder.Code)
	}
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if decoded.Product.Available {
		t.Fatal("expected available:false to be preserved")
	}
}

func TestCatalogProductAPIQuantityZeroDoesNot404(t *testing.T) {
	// 06-product-page.md's concrete JSON contract (section I) does not
	// include quantity as a field of this endpoint at all; this test
	// confirms quantity=0 in the source data has no effect on the HTTP
	// status, consistent with that decision.
	product := fakeProductAPICatalogProd(true)
	product.Quantity = 0
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 regardless of quantity, got %d", recorder.Code)
	}
}

func TestCatalogProductAPIWithoutMainImage(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.ImageURL = ""
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if decoded.Product.ImageFilename != nil {
		t.Fatalf("expected null image_filename, got %v", *decoded.Product.ImageFilename)
	}
}

func TestCatalogProductAPIWithMainImage(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if decoded.Product.ImageFilename == nil || *decoded.Product.ImageFilename != "mesa-redonda.jpg" {
		t.Fatalf("expected image_filename mesa-redonda.jpg, got %v", decoded.Product.ImageFilename)
	}
}

func TestCatalogProductAPIWithoutGallery(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.Images = nil
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if decoded.Product.Images == nil {
		t.Fatal("expected images to be an empty array, not null")
	}
	if len(decoded.Product.Images) != 0 {
		t.Fatalf("expected zero images, got %d", len(decoded.Product.Images))
	}
	if !strings.Contains(recorder.Body.String(), `"images":[]`) {
		t.Fatalf("expected literal empty array in JSON, got: %s", recorder.Body.String())
	}
}

func TestCatalogProductAPIGalleryWithOneImage(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.Images = []string{"unica.jpg"}
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if len(decoded.Product.Images) != 1 || decoded.Product.Images[0] != "unica.jpg" {
		t.Fatalf("expected exactly [unica.jpg], got %v", decoded.Product.Images)
	}
}

func TestCatalogProductAPIGalleryWithMultipleImagesOrderPreserved(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.Images = []string{"c.jpg", "a.jpg", "b.jpg"}
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	want := []string{"c.jpg", "a.jpg", "b.jpg"}
	if len(decoded.Product.Images) != len(want) {
		t.Fatalf("expected %d images, got %d", len(want), len(decoded.Product.Images))
	}
	for i, filename := range want {
		if decoded.Product.Images[i] != filename {
			t.Fatalf("expected order to be preserved from the loader: index %d wanted %q, got %q", i, filename, decoded.Product.Images[i])
		}
	}
}

func TestCatalogProductAPIInvalidFilenameFiltered(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.ImageURL = "../../etc/passwd"
	product.Images = []string{"valid.jpg", "../traversal.jpg", "with/slash.jpg", ""}
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)

	if decoded.Product.ImageFilename != nil {
		t.Fatalf("expected the unsafe main image filename to be dropped, got %v", *decoded.Product.ImageFilename)
	}
	if len(decoded.Product.Images) != 1 || decoded.Product.Images[0] != "valid.jpg" {
		t.Fatalf("expected only the one safe gallery filename to survive, got %v", decoded.Product.Images)
	}
	if strings.Contains(recorder.Body.String(), "..") || strings.Contains(recorder.Body.String(), "/etc/") {
		t.Fatalf("unsafe filename leaked into response: %s", recorder.Body.String())
	}
}

func TestCatalogProductAPICategoryValid(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if decoded.Product.Category == nil || decoded.Product.Category.Name != "Mobiliario" {
		t.Fatalf("expected category Mobiliario, got %v", decoded.Product.Category)
	}
}

func TestCatalogProductAPICategoryNullWhenMissing(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.CategoryID = ""
	product.CategoryName = ""
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	mustDecodeProductAPIResponse(t, recorder, &decoded)
	if decoded.Product.Category != nil {
		t.Fatalf("expected null category when category_id is empty, got %v", decoded.Product.Category)
	}
	if !strings.Contains(recorder.Body.String(), `"category":null`) {
		t.Fatalf("expected literal null in JSON, got: %s", recorder.Body.String())
	}
}

func TestCatalogProductAPILongDescriptionAlwaysString(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.LongDescription = ""
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if !strings.Contains(recorder.Body.String(), `"long_description":""`) {
		t.Fatalf("expected long_description to serialize as an empty string, never null, got: %s", recorder.Body.String())
	}
}

func TestCatalogProductAPIResponseIsValidJSON(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	var decoded publicAPICatalogProductDetailResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid JSON: %v (%s)", err, recorder.Body.String())
	}
}

func TestCatalogProductAPIExactContract(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)

	var raw map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	productRaw, ok := raw["product"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level \"product\" object, got: %s", recorder.Body.String())
	}
	wantKeys := []string{"id", "name", "slug", "description", "long_description", "category", "available", "image_filename", "images"}
	if len(productRaw) != len(wantKeys) {
		t.Fatalf("expected exactly %d keys %v, got %d keys: %v", len(wantKeys), wantKeys, len(productRaw), productRaw)
	}
	for _, key := range wantKeys {
		if _, present := productRaw[key]; !present {
			t.Fatalf("missing expected key %q in: %v", key, productRaw)
		}
	}
}

// --- Section 22: excluded fields ---

func TestCatalogProductAPIExcludesAdministrativeFields(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	body := recorder.Body.String()

	forbidden := []string{
		"search_vector",
		"qrcode_filename",
		"main_img_id",
		"gallery_ids",
		"\"source\"",
		"created_at",
		"updated_at",
		"/static/uploads",
		"web/static",
		"DATABASE_URL",
		"cart_id",
		"\"token\"",
		"\"price\"",
		"SELECT",
		"pgx",
	}
	for _, needle := range forbidden {
		if strings.Contains(body, needle) {
			t.Fatalf("response must not contain %q, got: %s", needle, body)
		}
	}
}

// --- Section 23: errors ---

func TestCatalogProductAPIInvalidIdentifierReturns400(t *testing.T) {
	loader, _ := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, "mesa/redonda")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "invalid_identifier") {
		t.Fatalf("expected invalid_identifier error code, got: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "mesa/redonda") {
		t.Fatal("the invalid identifier itself must never be reflected back in the response")
	}
}

func TestCatalogProductAPINoRowsReturns404(t *testing.T) {
	loader, _ := newProductAPILoader(nil, pgx.ErrNoRows)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, "no-existe")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "product_not_found") {
		t.Fatalf("expected product_not_found error code, got: %s", recorder.Body.String())
	}
}

func TestCatalogProductAPIUnavailableIsNot404(t *testing.T) {
	loader, _ := newProductAPILoader(fakeProductAPICatalogProd(false), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if recorder.Code != http.StatusOK {
		t.Fatalf("available=false must be 200, got %d", recorder.Code)
	}
}

func TestCatalogProductAPIDatabaseErrorReturns503(t *testing.T) {
	dbErr := errors.New("dial tcp 10.0.0.5:5432: password authentication failed")
	loader, _ := newProductAPILoader(nil, dbErr)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "catalog_unavailable") {
		t.Fatalf("expected catalog_unavailable error code, got: %s", body)
	}
	for _, secret := range []string{"password", "10.0.0.5", "5432", "dial tcp"} {
		if strings.Contains(body, secret) {
			t.Fatalf("internal detail %q leaked into response: %s", secret, body)
		}
	}
}

func TestCatalogProductAPIWrappedNoRowsStillMapsTo404(t *testing.T) {
	// If a future refactor of FindCatalogProductDetail ever wraps
	// pgx.ErrNoRows (fmt.Errorf("...: %w", pgx.ErrNoRows)), errors.Is must
	// still classify it as not-found, not as a generic 503.
	loader, _ := newProductAPILoader(nil, wrapError(pgx.ErrNoRows))
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, "no-existe")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a wrapped pgx.ErrNoRows, got %d", recorder.Code)
	}
}

func wrapError(err error) error {
	return &wrappedTestError{inner: err}
}

type wrappedTestError struct{ inner error }

func (e *wrappedTestError) Error() string { return "wrapped: " + e.inner.Error() }
func (e *wrappedTestError) Unwrap() error { return e.inner }

func TestCatalogProductAPIInternalStateInvalidReturns503(t *testing.T) {
	product := fakeProductAPICatalogProd(true)
	product.Name = ""
	loader, _ := newProductAPILoader(product, nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for a structurally invalid product, got %d", recorder.Code)
	}
}

func TestCatalogProductAPINoCookieSet(t *testing.T) {
	loader, _ := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if len(recorder.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("expected no Set-Cookie header — this endpoint must never create a cart session")
	}
}

func TestCatalogProductAPINoRedirect(t *testing.T) {
	loader, _ := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	handler := getPublicAPICatalogProductDetailHandler(loader)

	recorder := doProductAPIRequest(handler, productAPITestSlug)
	if recorder.Header().Get("Location") != "" {
		t.Fatal("expected no Location header — no redirect")
	}
}

// --- Section 24: headers and methods ---

func TestCatalogProductAPIHeaders(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		loader     catalogProductDetailAPILoader
		identifier string
		wantCode   int
	}{
		{name: "200", loader: mustLoader(fakeProductAPICatalogProd(true), nil), identifier: productAPITestSlug, wantCode: http.StatusOK},
		{name: "400", loader: mustLoader(nil, nil), identifier: "mesa/redonda", wantCode: http.StatusBadRequest},
		{name: "404", loader: mustLoader(nil, pgx.ErrNoRows), identifier: "no-existe", wantCode: http.StatusNotFound},
		{name: "503", loader: mustLoader(nil, errors.New("boom")), identifier: productAPITestSlug, wantCode: http.StatusServiceUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler := getPublicAPICatalogProductDetailHandler(testCase.loader)
			recorder := doProductAPIRequest(handler, testCase.identifier)

			if recorder.Code != testCase.wantCode {
				t.Fatalf("expected %d, got %d", testCase.wantCode, recorder.Code)
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type: expected application/json; charset=utf-8, got %q", got)
			}
			if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control: expected no-store, got %q", got)
			}
			if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options: expected nosniff, got %q", got)
			}
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("expected no CORS header, got %q", got)
			}
			if len(recorder.Header().Values("Set-Cookie")) != 0 {
				t.Error("expected no Set-Cookie")
			}
		})
	}
}

func mustLoader(product *db.CatalogProd, err error) catalogProductDetailAPILoader {
	loader, _ := newProductAPILoader(product, err)
	return loader
}

func TestCatalogProductAPIOnlyGETReachesHandler(t *testing.T) {
	router := NewCustomServeMux()
	loader, calls := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	router.HandleFunc("GET /api/catalog/products/{identifier}", getPublicAPICatalogProductDetailHandler(loader))

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/api/catalog/products/"+productAPITestSlug, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code == http.StatusOK {
				t.Fatalf("%s must not reach the GET-only handler", method)
			}
		})
	}
	if *calls != 0 {
		t.Fatalf("expected zero loader calls across all non-GET methods, got %d", *calls)
	}
}

// --- Section 25: router-level coexistence ---

func TestCatalogProductAPIRouteRegisteredAndCoexists(t *testing.T) {
	router := NewCustomServeMux()
	RegisterCatalogProductAPIRoutes(router)
	RegisterPublicAPIRoutes(router)
	RegisterProductsRoutes(router)
	manager := newDeterministicRouteCartManager(t, uuid.New())
	RegisterCatalogRoutes(router, manager)

	for _, testCase := range []struct {
		method  string
		path    string
		pattern string
	}{
		{method: http.MethodGet, path: "/api/catalog/products/" + productAPITestSlug, pattern: "GET /api/catalog/products/{identifier}"},
		{method: http.MethodGet, path: "/api/catalog/products/" + productAPITestUUID, pattern: "GET /api/catalog/products/{identifier}"},
		{method: http.MethodGet, path: "/api/catalog/products", pattern: "GET /api/catalog/products"},
		{method: http.MethodGet, path: "/api/catalog/categories", pattern: "GET /api/catalog/categories"},
		{method: http.MethodGet, path: "/api/catalog/listings", pattern: "GET /api/catalog/listings"},
		{method: http.MethodGet, path: "/api/products/" + productAPITestSlug, pattern: "GET /api/products/{slug}"},
		{method: http.MethodGet, path: "/catalogo/producto/" + productAPITestSlug, pattern: "GET /catalogo/producto/{id}"},
		{method: http.MethodGet, path: "/productos/" + productAPITestUUID, pattern: "GET /productos/{id}"},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		_, pattern := router.Handler(request)
		if pattern != testCase.pattern {
			t.Errorf("%s %s: expected pattern %q, got %q", testCase.method, testCase.path, testCase.pattern, pattern)
		}
	}
}

func TestCatalogProductAPINoOriginRequired(t *testing.T) {
	router := NewCustomServeMux()
	loader, _ := newProductAPILoader(fakeProductAPICatalogProd(true), nil)
	router.HandleFunc("GET /api/catalog/products/{identifier}", getPublicAPICatalogProductDetailHandler(loader))

	request := httptest.NewRequest(http.MethodGet, "/api/catalog/products/"+productAPITestSlug, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusForbidden {
		t.Fatal("expected no Origin/Referer requirement on this read-only endpoint")
	}
}
