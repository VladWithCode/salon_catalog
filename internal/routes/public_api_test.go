package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

type fakePublicSocialLinksLoader struct {
	links []db.SocialLink
	err   error
	calls int
}

type fakePublicCatalogListingsLoader struct {
	listings map[string][]*db.CatalogProd
	err      error
	calls    int
}

type fakePublicContactRequestCreator struct {
	quote *db.Quote
	err   error
	calls int
}

func (creator *fakePublicContactRequestCreator) Create(quote *db.Quote) error {
	creator.calls++
	creator.quote = quote
	return creator.err
}

func (loader *fakePublicCatalogListingsLoader) FindCatalogListings() (map[string][]*db.CatalogProd, error) {
	loader.calls++
	return loader.listings, loader.err
}

func (loader *fakePublicSocialLinksLoader) GetSocialLinks() ([]db.SocialLink, error) {
	loader.calls++
	return loader.links, loader.err
}

func TestGetPublicAPIHealth(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/api/_health", nil)
	recorder := httptest.NewRecorder()

	GetPublicAPIHealth(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", got)
	}

	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}

	var body publicAPIHealthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	expected := publicAPIHealthResponse{Status: "ok", Service: "go"}
	if body != expected {
		t.Errorf("expected body %+v, got %+v", expected, body)
	}

	if got := response.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("expected no cookies, got %q", got)
	}
}

func TestGetPublicAPISocials(t *testing.T) {
	t.Parallel()

	loader := &fakePublicSocialLinksLoader{
		links: []db.SocialLink{
			{ID: "admin-id", Name: "Facebook", Link: "https://facebook.com/villa"},
			{ID: "another-admin-id", Name: "Instagram", Link: "https://instagram.com/villa"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/socials", nil)
	recorder := httptest.NewRecorder()

	getPublicAPISocialsHandler(loader)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
	if got := response.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("expected no cookies, got %q", got)
	}
	if loader.calls != 1 {
		t.Fatalf("expected one loader call, got %d", loader.calls)
	}

	var body []publicAPISocialLink
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	expected := []publicAPISocialLink{
		{Name: "Facebook", Link: "https://facebook.com/villa"},
		{Name: "Instagram", Link: "https://instagram.com/villa"},
	}
	if len(body) != len(expected) {
		t.Fatalf("expected %d social links, got %d", len(expected), len(body))
	}
	for index := range expected {
		if body[index] != expected[index] {
			t.Errorf("expected social link %+v, got %+v", expected[index], body[index])
		}
	}
}

func TestGetPublicAPISocialsHandlesDatabaseError(t *testing.T) {
	t.Parallel()

	loader := &fakePublicSocialLinksLoader{err: errors.New("database unavailable")}
	request := httptest.NewRequest(http.MethodGet, "/api/socials", nil)
	recorder := httptest.NewRecorder()

	getPublicAPISocialsHandler(loader)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
	if got := response.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("expected no cookies, got %q", got)
	}

	var body publicAPIErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error != "socials_unavailable" {
		t.Errorf("expected controlled error, got %q", body.Error)
	}
}

func TestPublicAPIHealthRouteIsGETOnly(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/_health", nil)
	_, pattern := router.Handler(getRequest)
	if pattern != "GET /api/_health" {
		t.Fatalf("expected GET-only route pattern, got %q", pattern)
	}

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d", http.StatusOK, getRecorder.Code)
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/api/_health", nil)
	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, postRequest)

	if postRecorder.Code == http.StatusOK {
		t.Fatalf("expected POST not to execute the GET handler")
	}

	if strings.Contains(postRecorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("POST response unexpectedly contains the GET handler body")
	}
}

func TestPublicAPISocialsRouteIsGETOnly(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/socials", nil)
	_, pattern := router.Handler(getRequest)
	if pattern != "GET /api/socials" {
		t.Fatalf("expected GET-only route pattern, got %q", pattern)
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/api/socials", nil)
	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, postRequest)

	if postRecorder.Code == http.StatusOK {
		t.Fatalf("expected POST not to execute the GET handler")
	}
	if strings.Contains(postRecorder.Body.String(), "Facebook") {
		t.Fatalf("POST response unexpectedly contains the GET handler body")
	}
	if got := postRecorder.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("expected POST response not to set cookies, got %q", got)
	}
}

func TestGetPublicAPICatalogListings(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogListingsLoader{
		listings: map[string][]*db.CatalogProd{
			"Mantelería": {
				{ID: "product-5", Name: "Zafiro", Slug: "zafiro", Description: "Descripción cinco", ImageURL: "upload_2026-01-01T12:00:00_0.jpg"},
				{ID: "product-3", Name: "Nácar", Slug: "nacar", Description: "Descripción tres"},
				{ID: "product-1", Name: "Ámbar", Slug: "ambar", Description: "Descripción uno", ImageURL: "upload_2026-01-01T12:00:00_0.jpg"},
				nil,
				{ID: "product-4", Name: "Ónix", Slug: "onix", Description: "Descripción cuatro", ImageURL: "onix.jpg"},
				{ID: "product-2", Name: "Lino", Slug: "lino", Description: "Descripción dos", ImageURL: "lino.jpg"},
			},
			"Cristalería": {
				{ID: "glass-2", Name: "Copa vino", Slug: "copa-vino", Description: "Cristal fino"},
				{ID: "glass-1", Name: "Copa agua", Slug: "copa-agua", Description: "Cristal transparente", ImageURL: "copa.jpg"},
			},
			"   ": {
				{ID: "hidden", Name: "Oculto", Slug: "oculto"},
			},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/catalog/listings", nil)
	recorder := httptest.NewRecorder()
	getPublicAPICatalogListingsHandler(loader)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
	if got := response.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("expected no cookies, got %q", got)
	}
	if loader.calls != 1 {
		t.Fatalf("expected one loader call, got %d", loader.calls)
	}

	var body publicAPICatalogListingsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if len(body.Categories) != 2 {
		t.Fatalf("expected two named categories, got %d", len(body.Categories))
	}
	if body.Categories[0].Name != "Cristalería" || body.Categories[1].Name != "Mantelería" {
		t.Fatalf("expected alphabetically sorted categories, got %q then %q", body.Categories[0].Name, body.Categories[1].Name)
	}

	crystalProducts := body.Categories[0].Products
	if len(crystalProducts) != 2 {
		t.Fatalf("expected two crystal products, got %d", len(crystalProducts))
	}
	if crystalProducts[0].Name != "Copa agua" || crystalProducts[1].Name != "Copa vino" {
		t.Errorf("expected alphabetically sorted crystal products, got %q then %q", crystalProducts[0].Name, crystalProducts[1].Name)
	}
	if crystalProducts[0].ImageFilename == nil || *crystalProducts[0].ImageFilename != "copa.jpg" {
		t.Errorf("expected image filename copa.jpg, got %v", crystalProducts[0].ImageFilename)
	}
	if crystalProducts[1].ImageFilename != nil {
		t.Errorf("expected null image filename, got %q", *crystalProducts[1].ImageFilename)
	}

	linenProducts := body.Categories[1].Products
	if len(linenProducts) != 4 {
		t.Fatalf("expected defensive limit of four products, got %d", len(linenProducts))
	}
	expectedNames := []string{"Ámbar", "Lino", "Nácar", "Ónix"}
	for index, expectedName := range expectedNames {
		if linenProducts[index].Name != expectedName {
			t.Errorf("expected product %d to be %q, got %q", index, expectedName, linenProducts[index].Name)
		}
	}
	if linenProducts[0].ImageFilename == nil || *linenProducts[0].ImageFilename != "upload_2026-01-01T12:00:00_0.jpg" {
		t.Errorf("expected timestamp filename to be preserved, got %v", linenProducts[0].ImageFilename)
	}
}

func TestGetPublicAPICatalogListingsReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogListingsLoader{listings: nil}
	request := httptest.NewRequest(http.MethodGet, "/api/catalog/listings", nil)
	recorder := httptest.NewRecorder()

	getPublicAPICatalogListingsHandler(loader)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()
	var body publicAPICatalogListingsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Categories == nil {
		t.Fatal("expected categories to be an empty array, got null")
	}
	if len(body.Categories) != 0 {
		t.Fatalf("expected no categories, got %d", len(body.Categories))
	}
}

func TestGetPublicAPICatalogListingsHandlesLoaderError(t *testing.T) {
	t.Parallel()

	internalError := "postgres failed at 127.0.0.1 with secret details"
	loader := &fakePublicCatalogListingsLoader{err: errors.New(internalError)}
	request := httptest.NewRequest(http.MethodGet, "/api/catalog/listings", nil)
	recorder := httptest.NewRecorder()

	getPublicAPICatalogListingsHandler(loader)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
	if got := response.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("expected no cookies, got %q", got)
	}
	if loader.calls != 1 {
		t.Fatalf("expected one loader call, got %d", loader.calls)
	}

	var body publicAPIErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error != "catalog_unavailable" {
		t.Errorf("expected controlled error, got %q", body.Error)
	}
	if strings.Contains(body.Error, internalError) || strings.Contains(recorder.Body.String(), internalError) {
		t.Fatal("response leaked the internal loader error")
	}
}

func TestPublicAPICatalogListingsRouteIsGETOnly(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/catalog/listings", nil)
	_, pattern := router.Handler(getRequest)
	if pattern != "GET /api/catalog/listings" {
		t.Fatalf("expected GET-only route pattern, got %q", pattern)
	}

	loader := &fakePublicCatalogListingsLoader{}
	testRouter := NewCustomServeMux()
	testRouter.HandleFunc("GET /api/catalog/listings", getPublicAPICatalogListingsHandler(loader))
	postRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/listings", nil)
	postRecorder := httptest.NewRecorder()
	testRouter.ServeHTTP(postRecorder, postRequest)

	if postRecorder.Code == http.StatusOK {
		t.Fatal("expected POST not to execute the GET handler")
	}
	if loader.calls != 0 {
		t.Fatalf("expected POST not to call the loader, got %d calls", loader.calls)
	}
	if strings.Contains(postRecorder.Body.String(), `"categories"`) {
		t.Fatal("POST response unexpectedly contains the GET handler body")
	}
	if got := postRecorder.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("expected POST response not to set cookies, got %q", got)
	}
}

func TestPostPublicAPIContactRequestCreatesContactQuote(t *testing.T) {
	t.Parallel()

	creator := &fakePublicContactRequestCreator{}
	recorder := performPublicAPIContactRequest(
		`{"name":"  José Álvarez  ","phone":"  +52 (618) 123-4567  "}`,
		"application/json; charset=utf-8",
		creator,
	)

	response := recorder.Result()
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
	if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected nosniff, got %q", got)
	}
	if got := response.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("expected no cookies, got %q", got)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"ok":true,"message":"Solicitud recibida"}` {
		t.Errorf("unexpected success response %s", got)
	}
	if creator.calls != 1 {
		t.Fatalf("expected creator to run once, got %d", creator.calls)
	}
	if creator.quote == nil {
		t.Fatal("expected a quote to be passed to the creator")
	}
	if creator.quote.CustomerName != "José Álvarez" {
		t.Errorf("expected trimmed customer name, got %q", creator.quote.CustomerName)
	}
	if creator.quote.CustomerPhone != "+52 (618) 123-4567" {
		t.Errorf("expected trimmed customer phone, got %q", creator.quote.CustomerPhone)
	}
	if creator.quote.RequestType != db.QuoteRequestTypeContact {
		t.Errorf("expected contact request type, got %q", creator.quote.RequestType)
	}
	if creator.quote.Status != db.QuoteStatusPending {
		t.Errorf("expected pending status, got %q", creator.quote.Status)
	}
	if creator.quote.ID != "" || creator.quote.CustomerEmail != "" {
		t.Error("expected internal and email fields to remain empty")
	}
	if creator.quote.TimeStart != nil || creator.quote.TimeEnd != nil || creator.quote.Cart != nil {
		t.Error("expected optional pointer fields to remain nil")
	}
	if creator.quote.Comments.Valid || creator.quote.CartID.Valid || creator.quote.EventKindID.Valid || creator.quote.EventKindName.Valid {
		t.Error("expected optional database fields to remain null")
	}
	if !creator.quote.CreatedAt.IsZero() || !creator.quote.UpdatedAt.IsZero() {
		t.Error("expected timestamps to remain unset before persistence")
	}
}

func TestPostPublicAPIContactRequestValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		requestBody   string
		expectedField string
		expectedError string
	}{
		{
			name:          "empty name",
			requestBody:   `{"name":"   ","phone":"6181234567"}`,
			expectedField: "name",
			expectedError: "El nombre es obligatorio.",
		},
		{
			name:          "one unicode character name",
			requestBody:   `{"name":"Á","phone":"6181234567"}`,
			expectedField: "name",
			expectedError: "El nombre debe tener al menos 2 caracteres.",
		},
		{
			name:          "name over 120 unicode characters",
			requestBody:   mustMarshalPublicAPIContactRequest(t, strings.Repeat("Á", 121), "6181234567"),
			expectedField: "name",
			expectedError: "El nombre no puede exceder 120 caracteres.",
		},
		{
			name:          "empty phone",
			requestBody:   `{"name":"Ana","phone":"   "}`,
			expectedField: "phone",
			expectedError: "El teléfono es obligatorio.",
		},
		{
			name:          "phone under ten digits",
			requestBody:   `{"name":"Ana","phone":"618123456"}`,
			expectedField: "phone",
			expectedError: "Ingresa un número de teléfono válido de entre 10 y 15 dígitos.",
		},
		{
			name:          "phone over fifteen digits",
			requestBody:   `{"name":"Ana","phone":"1234567890123456"}`,
			expectedField: "phone",
			expectedError: "Ingresa un número de teléfono válido de entre 10 y 15 dígitos.",
		},
		{
			name:          "phone with letters",
			requestBody:   `{"name":"Ana","phone":"618ABC4567"}`,
			expectedField: "phone",
			expectedError: "Ingresa un número de teléfono válido de entre 10 y 15 dígitos.",
		},
		{
			name:          "phone with unsupported symbol",
			requestBody:   `{"name":"Ana","phone":"618.123.4567"}`,
			expectedField: "phone",
			expectedError: "Ingresa un número de teléfono válido de entre 10 y 15 dígitos.",
		},
		{
			name:          "phone over 32 characters",
			requestBody:   mustMarshalPublicAPIContactRequest(t, "Ana", "6"+strings.Repeat(" ", 23)+"181234567"),
			expectedField: "phone",
			expectedError: "Ingresa un número de teléfono válido de entre 10 y 15 dígitos.",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			creator := &fakePublicContactRequestCreator{}
			recorder := performPublicAPIContactRequest(testCase.requestBody, "application/json", creator)

			response := recorder.Result()
			defer response.Body.Close()
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.StatusCode)
			}
			if creator.calls != 0 {
				t.Fatalf("expected invalid request not to call creator, got %d calls", creator.calls)
			}

			var body struct {
				Error  string            `json:"error"`
				Fields map[string]string `json:"fields"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode validation response: %v", err)
			}
			if body.Error != "validation_failed" {
				t.Errorf("expected validation_failed, got %q", body.Error)
			}
			if len(body.Fields) != 1 || body.Fields[testCase.expectedField] != testCase.expectedError {
				t.Errorf("expected only %s error %q, got %+v", testCase.expectedField, testCase.expectedError, body.Fields)
			}
		})
	}
}

func TestPostPublicAPIContactRequestRejectsInvalidJSONContracts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"name":`},
		{name: "unknown field", body: `{"name":"Ana","phone":"6181234567","status":"procesada"}`},
		{name: "incorrect field type", body: `{"name":7,"phone":"6181234567"}`},
		{name: "additional JSON value", body: `{"name":"Ana","phone":"6181234567"} {}`},
		{name: "array root", body: `[{"name":"Ana","phone":"6181234567"}]`},
		{name: "null root", body: `null`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			creator := &fakePublicContactRequestCreator{}
			recorder := performPublicAPIContactRequest(testCase.body, "application/json", creator)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
			}
			if creator.calls != 0 {
				t.Fatalf("expected invalid request not to call creator, got %d calls", creator.calls)
			}
			if got := strings.TrimSpace(recorder.Body.String()); got != `{"error":"invalid_request"}` {
				t.Errorf("unexpected invalid request response %s", got)
			}
			assertPublicAPIContactSecurityHeaders(t, recorder)
		})
	}
}

func TestPostPublicAPIContactRequestRejectsUnsupportedMediaType(t *testing.T) {
	t.Parallel()

	creator := &fakePublicContactRequestCreator{}
	recorder := performPublicAPIContactRequest(
		`{"name":"Ana","phone":"6181234567"}`,
		"text/plain",
		creator,
	)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, recorder.Code)
	}
	if creator.calls != 0 {
		t.Fatalf("expected unsupported content type not to call creator, got %d calls", creator.calls)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"error":"unsupported_media_type"}` {
		t.Errorf("unexpected unsupported media response %s", got)
	}
	assertPublicAPIContactSecurityHeaders(t, recorder)
}

func TestPostPublicAPIContactRequestRejectsLargeBody(t *testing.T) {
	t.Parallel()

	creator := &fakePublicContactRequestCreator{}
	requestBody := `{"name":"` + strings.Repeat("a", int(publicContactRequestMaxBodyBytes)) + `","phone":"6181234567"}`
	recorder := performPublicAPIContactRequest(requestBody, "application/json", creator)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}
	if creator.calls != 0 {
		t.Fatalf("expected oversized request not to call creator, got %d calls", creator.calls)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"error":"request_too_large"}` {
		t.Errorf("unexpected oversized response %s", got)
	}
	assertPublicAPIContactSecurityHeaders(t, recorder)
}

func TestPostPublicAPIContactRequestHandlesCreatorError(t *testing.T) {
	t.Parallel()

	internalError := "insert into quotes failed at 127.0.0.1 with secret details"
	creator := &fakePublicContactRequestCreator{err: errors.New(internalError)}
	recorder := performPublicAPIContactRequest(
		`{"name":"Ana Pérez","phone":"6181234567"}`,
		"application/json",
		creator,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if creator.calls != 1 {
		t.Fatalf("expected creator to run once, got %d", creator.calls)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"error":"contact_unavailable"}` {
		t.Errorf("unexpected creator error response %s", got)
	}
	if strings.Contains(recorder.Body.String(), internalError) || strings.Contains(recorder.Body.String(), "quotes") {
		t.Fatal("response leaked internal persistence details")
	}
	assertPublicAPIContactSecurityHeaders(t, recorder)
}

func TestPublicAPIContactRequestRouteIsPOSTOnly(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	postRequest := httptest.NewRequest(http.MethodPost, "/api/contact-requests", nil)
	_, pattern := router.Handler(postRequest)
	if pattern != "POST /api/contact-requests" {
		t.Fatalf("expected POST-only route pattern, got %q", pattern)
	}

	creator := &fakePublicContactRequestCreator{}
	testRouter := NewCustomServeMux()
	testRouter.HandleFunc("POST /api/contact-requests", postPublicAPIContactRequestHandler(creator.Create))
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		request := httptest.NewRequest(method, "/api/contact-requests", nil)
		recorder := httptest.NewRecorder()
		testRouter.ServeHTTP(recorder, request)
		if recorder.Code == http.StatusCreated {
			t.Errorf("expected %s not to execute POST handler", method)
		}
		if got := recorder.Header().Get("Set-Cookie"); got != "" {
			t.Errorf("expected %s not to create cookies, got %q", method, got)
		}
	}
	if creator.calls != 0 {
		t.Fatalf("expected non-POST methods not to call creator, got %d calls", creator.calls)
	}
}

func performPublicAPIContactRequest(
	requestBody string,
	contentType string,
	creator *fakePublicContactRequestCreator,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/contact-requests", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	postPublicAPIContactRequestHandler(creator.Create)(recorder, request)
	return recorder
}

func mustMarshalPublicAPIContactRequest(t *testing.T, name string, phone string) string {
	t.Helper()
	body, err := json.Marshal(publicAPIContactRequest{Name: name, Phone: phone})
	if err != nil {
		t.Fatalf("marshal contact request: %v", err)
	}
	return string(body)
}

func assertPublicAPIContactSecurityHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected nosniff, got %q", got)
	}
	if got := recorder.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("expected no cookies, got %q", got)
	}
}

type fakePublicCatalogCategoriesLoader struct {
	categories []*db.CatalogCtg
	err        error
	calls      int
}

func (loader *fakePublicCatalogCategoriesLoader) load() ([]*db.CatalogCtg, error) {
	loader.calls++
	return loader.categories, loader.err
}

type fakePublicCatalogProductsLoader struct {
	result   *db.CatalogProductFilterResult
	err      error
	calls    int
	category string
	search   string
	page     int
	limit    int
}

func (loader *fakePublicCatalogProductsLoader) load(
	category string,
	search string,
	page int,
	limit int,
) (*db.CatalogProductFilterResult, error) {
	loader.calls++
	loader.category = category
	loader.search = search
	loader.page = page
	loader.limit = limit
	return loader.result, loader.err
}

func requestPublicAPICatalogCategories(
	t *testing.T,
	loader *fakePublicCatalogCategoriesLoader,
) (*httptest.ResponseRecorder, publicAPICatalogCategoriesResponse) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/api/catalog/categories", nil)
	recorder := httptest.NewRecorder()
	getPublicAPICatalogCategoriesHandler(loader.load)(recorder, request)

	var body publicAPICatalogCategoriesResponse
	if recorder.Code == http.StatusOK {
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
	}

	return recorder, body
}

func requestPublicAPICatalogProducts(
	t *testing.T,
	loader *fakePublicCatalogProductsLoader,
	target string,
) (*httptest.ResponseRecorder, publicAPICatalogProductsResponse) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	getPublicAPICatalogProductsHandler(loader.load)(recorder, request)

	var body publicAPICatalogProductsResponse
	if recorder.Code == http.StatusOK {
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
	}

	return recorder, body
}

func TestGetPublicAPICatalogCategories(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogCategoriesLoader{
		categories: []*db.CatalogCtg{
			{ID: "ctg-mesas", Name: "  Mesas  ", ProductCount: 12},
			nil,
			{ID: "", Name: "Sin identificador", ProductCount: 3},
			{ID: "ctg-vacia", Name: "   ", ProductCount: 0},
			{ID: "ctg-cristaleria", Name: "Cristalería", ProductCount: 7},
			{ID: "ctg-blancos", Name: "Blancos", ProductCount: 0},
		},
	}

	recorder, body := requestPublicAPICatalogCategories(t, loader)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	assertPublicAPIContactSecurityHeaders(t, recorder)

	if loader.calls != 1 {
		t.Fatalf("expected one loader call, got %d", loader.calls)
	}

	expected := []publicAPICatalogCategorySummary{
		{ID: "ctg-blancos", Name: "Blancos", ProductCount: 0},
		{ID: "ctg-cristaleria", Name: "Cristalería", ProductCount: 7},
		{ID: "ctg-mesas", Name: "Mesas", ProductCount: 12},
	}
	if len(body.Categories) != len(expected) {
		t.Fatalf("expected %d categories, got %d (%+v)", len(expected), len(body.Categories), body.Categories)
	}
	for index, category := range expected {
		if body.Categories[index] != category {
			t.Errorf("category %d: expected %+v, got %+v", index, category, body.Categories[index])
		}
	}
}

func TestGetPublicAPICatalogCategoriesReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogCategoriesLoader{}
	recorder, body := requestPublicAPICatalogCategories(t, loader)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Categories == nil {
		t.Fatal("expected categories to be an empty array, got null")
	}
	if len(body.Categories) != 0 {
		t.Fatalf("expected no categories, got %+v", body.Categories)
	}
	if !strings.Contains(recorder.Body.String(), `"categories":[]`) {
		t.Fatalf("expected an empty JSON array, got %s", recorder.Body.String())
	}
}

func TestGetPublicAPICatalogCategoriesHandlesLoaderError(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogCategoriesLoader{
		err: errors.New(`relation "catalog_categories" does not exist`),
	}
	recorder, _ := requestPublicAPICatalogCategories(t, loader)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	assertPublicAPIContactSecurityHeaders(t, recorder)

	var body publicAPIErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error != "catalog_unavailable" {
		t.Fatalf("expected catalog_unavailable, got %q", body.Error)
	}
	if strings.Contains(recorder.Body.String(), "catalog_categories") {
		t.Fatalf("internal error leaked into the response: %s", recorder.Body.String())
	}
}

func TestPublicAPICatalogCategoriesRouteIsGETOnly(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/catalog/categories", nil)
	_, pattern := router.Handler(getRequest)
	if pattern != "GET /api/catalog/categories" {
		t.Fatalf("expected GET-only route pattern, got %q", pattern)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		request := httptest.NewRequest(method, "/api/catalog/categories", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code == http.StatusOK {
			t.Fatalf("expected %s not to execute the GET handler", method)
		}
		if strings.Contains(recorder.Body.String(), `"categories"`) {
			t.Fatalf("%s response unexpectedly contains the GET handler body", method)
		}
	}
}

func TestGetPublicAPICatalogProducts(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogProductsLoader{
		result: &db.CatalogProductFilterResult{
			Products: []*db.CatalogProd{
				{
					ID:              "prod-2",
					Name:            "Mesa redonda",
					Slug:            "mesa-redonda",
					Description:     "Descripción corta",
					LongDescription: "Descripción larga que no debe exponerse",
					CategoryID:      "ctg-mesas",
					CategoryName:    "Mesas",
					ImageURL:        "  upload_mesa.jpg  ",
					Images:          []string{"upload_mesa.jpg", "upload_mesa_2.jpg"},
					Available:       true,
					Quantity:        9,
				},
				nil,
				{
					ID:           "prod-1",
					Name:         "Copa alta",
					Slug:         "copa-alta",
					Description:  "Copa de cristal",
					CategoryID:   "ctg-cristaleria",
					CategoryName: "Cristalería",
					ImageURL:     "   ",
					Available:    false,
					Quantity:     0,
				},
			},
			Total:       25,
			Page:        2,
			Limit:       16,
			TotalPages:  2,
			HasNext:     false,
			HasPrevious: true,
		},
	}

	recorder, body := requestPublicAPICatalogProducts(
		t,
		loader,
		"/api/catalog/products?buscar=%20mesa%20&categoria=%20Mesas%20&pagina=2&por_pagina=16",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	assertPublicAPIContactSecurityHeaders(t, recorder)

	if loader.calls != 1 {
		t.Fatalf("expected one loader call, got %d", loader.calls)
	}
	if loader.search != "mesa" {
		t.Errorf("expected the loader to receive a trimmed search term, got %q", loader.search)
	}
	if loader.category != "Mesas" {
		t.Errorf("expected the loader to receive a trimmed category, got %q", loader.category)
	}
	if loader.page != 2 || loader.limit != 16 {
		t.Errorf("expected page 2 and limit 16, got page %d and limit %d", loader.page, loader.limit)
	}

	if len(body.Items) != 2 {
		t.Fatalf("expected two items, got %+v", body.Items)
	}
	if body.Items[0].ID != "prod-2" || body.Items[1].ID != "prod-1" {
		t.Errorf("expected the database order to be preserved, got %q then %q", body.Items[0].ID, body.Items[1].ID)
	}
	if body.Items[0].ImageFilename == nil || *body.Items[0].ImageFilename != "upload_mesa.jpg" {
		t.Errorf("expected a trimmed image filename, got %v", body.Items[0].ImageFilename)
	}
	if body.Items[1].ImageFilename != nil {
		t.Errorf("expected a blank image filename to become null, got %v", *body.Items[1].ImageFilename)
	}
	if body.Items[1].Available {
		t.Error("expected the unavailable product to keep available=false")
	}
	if body.Items[1].CategoryID != "ctg-cristaleria" || body.Items[1].CategoryName != "Cristalería" {
		t.Errorf("expected category fields to be preserved, got %+v", body.Items[1])
	}

	expectedPagination := publicAPICatalogPagination{
		Page:        2,
		PageSize:    16,
		TotalItems:  25,
		TotalPages:  2,
		HasNext:     false,
		HasPrevious: true,
	}
	if body.Pagination != expectedPagination {
		t.Errorf("expected pagination %+v, got %+v", expectedPagination, body.Pagination)
	}

	if body.Filters.Query != "mesa" {
		t.Errorf("expected the trimmed query in filters, got %q", body.Filters.Query)
	}
	if body.Filters.Category == nil || *body.Filters.Category != "Mesas" {
		t.Errorf("expected the trimmed category in filters, got %v", body.Filters.Category)
	}

	raw := recorder.Body.String()
	for _, forbidden := range []string{
		"long_description",
		"Descripción larga",
		`"images"`,
		"quantity",
		"search_vector",
		"search_rank",
		"main_img_id",
		"gallery_ids",
		"qrcode_filename",
		"has_error",
	} {
		if strings.Contains(raw, forbidden) {
			t.Errorf("response unexpectedly exposes %q: %s", forbidden, raw)
		}
	}
}

func TestGetPublicAPICatalogProductsUsesDefaults(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogProductsLoader{
		result: &db.CatalogProductFilterResult{
			Page:  1,
			Limit: db.DefaultCatalogPageSize,
		},
	}

	recorder, body := requestPublicAPICatalogProducts(t, loader, "/api/catalog/products")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if loader.page != 1 {
		t.Errorf("expected the default page 1, got %d", loader.page)
	}
	if loader.limit != db.DefaultCatalogPageSize {
		t.Errorf("expected the default page size %d, got %d", db.DefaultCatalogPageSize, loader.limit)
	}
	if loader.search != "" || loader.category != "" {
		t.Errorf("expected empty filters, got search %q and category %q", loader.search, loader.category)
	}
	if body.Filters.Query != "" {
		t.Errorf("expected an empty query filter, got %q", body.Filters.Query)
	}
	if body.Filters.Category != nil {
		t.Errorf("expected a null category filter, got %q", *body.Filters.Category)
	}
	if body.Items == nil {
		t.Fatal("expected items to be an empty array, got null")
	}
	if !strings.Contains(recorder.Body.String(), `"items":[]`) {
		t.Fatalf("expected an empty JSON array, got %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"category":null`) {
		t.Fatalf("expected a null category filter in the payload, got %s", recorder.Body.String())
	}
}

func TestGetPublicAPICatalogProductsPassesCategoryUUIDUnchanged(t *testing.T) {
	t.Parallel()

	const categoryID = "11111111-1111-1111-1111-111111111111"
	loader := &fakePublicCatalogProductsLoader{
		result: &db.CatalogProductFilterResult{Page: 1, Limit: 16},
	}

	recorder, body := requestPublicAPICatalogProducts(
		t,
		loader,
		"/api/catalog/products?categoria="+categoryID,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if loader.category != categoryID {
		t.Errorf("expected the category UUID to be passed through, got %q", loader.category)
	}
	if body.Filters.Category == nil || *body.Filters.Category != categoryID {
		t.Errorf("expected the category UUID in filters, got %v", body.Filters.Category)
	}
}

func TestGetPublicAPICatalogProductsRejectsInvalidParameters(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		target      string
		wantFields  []string
		otherFields []string
	}{
		{name: "page zero", target: "/api/catalog/products?pagina=0", wantFields: []string{"pagina"}, otherFields: []string{"por_pagina"}},
		{name: "page negative", target: "/api/catalog/products?pagina=-2", wantFields: []string{"pagina"}, otherFields: []string{"por_pagina"}},
		{name: "page decimal", target: "/api/catalog/products?pagina=1.5", wantFields: []string{"pagina"}, otherFields: []string{"por_pagina"}},
		{name: "page text", target: "/api/catalog/products?pagina=abc", wantFields: []string{"pagina"}, otherFields: []string{"por_pagina"}},
		{name: "page size zero", target: "/api/catalog/products?por_pagina=0", wantFields: []string{"por_pagina"}, otherFields: []string{"pagina"}},
		{name: "page size negative", target: "/api/catalog/products?por_pagina=-5", wantFields: []string{"por_pagina"}, otherFields: []string{"pagina"}},
		{name: "page size above maximum", target: "/api/catalog/products?por_pagina=101", wantFields: []string{"por_pagina"}, otherFields: []string{"pagina"}},
		{name: "page size decimal", target: "/api/catalog/products?por_pagina=16.5", wantFields: []string{"por_pagina"}, otherFields: []string{"pagina"}},
		{name: "page size text", target: "/api/catalog/products?por_pagina=muchos", wantFields: []string{"por_pagina"}, otherFields: []string{"pagina"}},
		{name: "both invalid", target: "/api/catalog/products?pagina=0&por_pagina=999", wantFields: []string{"pagina", "por_pagina"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			loader := &fakePublicCatalogProductsLoader{result: &db.CatalogProductFilterResult{}}
			request := httptest.NewRequest(http.MethodGet, testCase.target, nil)
			recorder := httptest.NewRecorder()

			getPublicAPICatalogProductsHandler(loader.load)(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
			}
			assertPublicAPIContactSecurityHeaders(t, recorder)
			if loader.calls != 0 {
				t.Fatalf("expected the loader not to run, got %d calls", loader.calls)
			}

			var body publicAPIInvalidParametersResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}
			if body.Error != "invalid_parameters" {
				t.Fatalf("expected invalid_parameters, got %q", body.Error)
			}
			if len(body.Fields) != len(testCase.wantFields) {
				t.Fatalf("expected %d field errors, got %+v", len(testCase.wantFields), body.Fields)
			}
			for _, field := range testCase.wantFields {
				if body.Fields[field] == "" {
					t.Errorf("expected a %q field error, got %+v", field, body.Fields)
				}
			}
			for _, field := range testCase.otherFields {
				if _, ok := body.Fields[field]; ok {
					t.Errorf("expected no %q field error, got %+v", field, body.Fields)
				}
			}
		})
	}
}

func TestGetPublicAPICatalogProductsKeepsOutOfRangePage(t *testing.T) {
	t.Parallel()

	loader := &fakePublicCatalogProductsLoader{
		result: &db.CatalogProductFilterResult{
			Products:    nil,
			Total:       25,
			Page:        99,
			Limit:       16,
			TotalPages:  2,
			HasNext:     false,
			HasPrevious: true,
		},
	}

	recorder, body := requestPublicAPICatalogProducts(t, loader, "/api/catalog/products?pagina=99")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Items == nil || len(body.Items) != 0 {
		t.Fatalf("expected an empty item array, got %+v", body.Items)
	}
	expected := publicAPICatalogPagination{
		Page:        99,
		PageSize:    16,
		TotalItems:  25,
		TotalPages:  2,
		HasNext:     false,
		HasPrevious: true,
	}
	if body.Pagination != expected {
		t.Errorf("expected the database pagination to be preserved, got %+v", body.Pagination)
	}
}

func TestGetPublicAPICatalogProductsHandlesLoaderFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		loader *fakePublicCatalogProductsLoader
	}{
		{
			name: "loader error",
			loader: &fakePublicCatalogProductsLoader{
				err: errors.New(`column "spanish" does not exist (SQLSTATE 42703)`),
			},
		},
		{
			name:   "nil result",
			loader: &fakePublicCatalogProductsLoader{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/api/catalog/products?buscar=mesa", nil)
			recorder := httptest.NewRecorder()

			getPublicAPICatalogProductsHandler(testCase.loader.load)(recorder, request)

			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
			}
			assertPublicAPIContactSecurityHeaders(t, recorder)

			var body publicAPIErrorResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}
			if body.Error != "catalog_unavailable" {
				t.Fatalf("expected catalog_unavailable, got %q", body.Error)
			}
			for _, leak := range []string{"SQLSTATE", "spanish", "catalog_products", "DATABASE_URL"} {
				if strings.Contains(recorder.Body.String(), leak) {
					t.Errorf("internal detail %q leaked into the response: %s", leak, recorder.Body.String())
				}
			}
		})
	}
}

func TestPublicAPICatalogProductsRouteIsGETOnly(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/catalog/products", nil)
	_, pattern := router.Handler(getRequest)
	if pattern != "GET /api/catalog/products" {
		t.Fatalf("expected GET-only route pattern, got %q", pattern)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		request := httptest.NewRequest(method, "/api/catalog/products", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code == http.StatusOK {
			t.Fatalf("expected %s not to execute the GET handler", method)
		}
		if strings.Contains(recorder.Body.String(), `"pagination"`) {
			t.Fatalf("%s response unexpectedly contains the GET handler body", method)
		}
	}
}

func TestPublicAPICatalogListingsRouteIsUnchanged(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/api/catalog/listings", nil)
	_, pattern := router.Handler(request)
	if pattern != "GET /api/catalog/listings" {
		t.Fatalf("expected the Home listings route to stay registered, got %q", pattern)
	}
}
