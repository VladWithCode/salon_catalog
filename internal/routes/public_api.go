package routes

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/vladwithcode/salon_catalog/internal/db"
	"golang.org/x/text/unicode/norm"
)

type publicAPIHealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type publicAPISocialLink struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

type publicAPIErrorResponse struct {
	Error string `json:"error"`
}

type publicAPIContactRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type publicAPIContactSuccessResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type publicAPIContactFieldErrors struct {
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
}

type publicAPIContactValidationResponse struct {
	Error  string                      `json:"error"`
	Fields publicAPIContactFieldErrors `json:"fields"`
}

type publicAPICatalogProduct struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	Description   string  `json:"description"`
	ImageFilename *string `json:"image_filename"`
}

type publicAPICatalogCategory struct {
	Name     string                    `json:"name"`
	Products []publicAPICatalogProduct `json:"products"`
}

type publicAPICatalogListingsResponse struct {
	Categories []publicAPICatalogCategory `json:"categories"`
}

type publicAPICatalogCategorySummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProductCount int    `json:"product_count"`
}

type publicAPICatalogCategoriesResponse struct {
	Categories []publicAPICatalogCategorySummary `json:"categories"`
}

type publicAPICatalogListItem struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	Description   string  `json:"description"`
	CategoryID    string  `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	ImageFilename *string `json:"image_filename"`
	Available     bool    `json:"available"`
}

type publicAPICatalogPagination struct {
	Page        int  `json:"page"`
	PageSize    int  `json:"page_size"`
	TotalItems  int  `json:"total_items"`
	TotalPages  int  `json:"total_pages"`
	HasNext     bool `json:"has_next"`
	HasPrevious bool `json:"has_previous"`
}

type publicAPICatalogFilters struct {
	Query    string  `json:"query"`
	Category *string `json:"category"`
}

type publicAPICatalogProductsResponse struct {
	Items      []publicAPICatalogListItem `json:"items"`
	Pagination publicAPICatalogPagination `json:"pagination"`
	Filters    publicAPICatalogFilters    `json:"filters"`
}

type publicAPIInvalidParametersResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields"`
}

// publicCatalogCategoriesLoader resolves the catalog categories without a
// search term. It exists so the handler can be tested without PostgreSQL.
type publicCatalogCategoriesLoader func() ([]*db.CatalogCtg, error)

// publicCatalogProductsLoader resolves a paginated page of catalog products.
// The category value is passed through untouched: the database layer already
// accepts either a category UUID or an exact category name.
type publicCatalogProductsLoader func(category string, search string, page int, limit int) (*db.CatalogProductFilterResult, error)

type publicSocialLinksLoader interface {
	GetSocialLinks() ([]db.SocialLink, error)
}

type databaseSocialLinksLoader struct{}

func (databaseSocialLinksLoader) GetSocialLinks() ([]db.SocialLink, error) {
	return db.GetSocialLinks()
}

type publicCatalogListingsLoader interface {
	FindCatalogListings() (map[string][]*db.CatalogProd, error)
}

type databaseCatalogListingsLoader struct{}

func (databaseCatalogListingsLoader) FindCatalogListings() (map[string][]*db.CatalogProd, error) {
	return db.FindCatalogListings()
}

type publicContactRequestCreator func(*db.Quote) error

const publicContactRequestMaxBodyBytes int64 = 8 * 1024

func RegisterPublicAPIRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/_health", GetPublicAPIHealth)
	router.HandleFunc("GET /api/socials", GetPublicAPISocials)
	router.HandleFunc("GET /api/catalog/listings", GetPublicAPICatalogListings)
	router.HandleFunc("GET /api/catalog/categories", GetPublicAPICatalogCategories)
	router.HandleFunc("GET /api/catalog/products", GetPublicAPICatalogProducts)
	router.HandleFunc("POST /api/contact-requests", PostPublicAPIContactRequest)
}

func GetPublicAPIHealth(w http.ResponseWriter, _ *http.Request) {
	response := publicAPIHealthResponse{
		Status:  "ok",
		Service: "go",
	}

	writePublicAPIJSON(w, http.StatusOK, response)
}

func GetPublicAPISocials(w http.ResponseWriter, r *http.Request) {
	getPublicAPISocialsHandler(databaseSocialLinksLoader{})(w, r)
}

func getPublicAPISocialsHandler(loader publicSocialLinksLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		links, err := loader.GetSocialLinks()
		if err != nil {
			writePublicAPIJSON(w, http.StatusInternalServerError, publicAPIErrorResponse{
				Error: "socials_unavailable",
			})
			return
		}

		response := make([]publicAPISocialLink, 0, len(links))
		for _, link := range links {
			response = append(response, publicAPISocialLink{
				Name: link.Name,
				Link: link.Link,
			})
		}

		writePublicAPIJSON(w, http.StatusOK, response)
	}
}

func GetPublicAPICatalogListings(w http.ResponseWriter, r *http.Request) {
	getPublicAPICatalogListingsHandler(databaseCatalogListingsLoader{})(w, r)
}

func getPublicAPICatalogListingsHandler(loader publicCatalogListingsLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		listings, err := loader.FindCatalogListings()
		if err != nil {
			writePublicAPIJSON(w, http.StatusInternalServerError, publicAPIErrorResponse{
				Error: "catalog_unavailable",
			})
			return
		}

		categoryNames := make([]string, 0, len(listings))
		for name := range listings {
			if strings.TrimSpace(name) != "" {
				categoryNames = append(categoryNames, name)
			}
		}
		sort.SliceStable(categoryNames, func(i, j int) bool {
			return catalogAlphabeticalLess(categoryNames[i], categoryNames[j])
		})

		categories := make([]publicAPICatalogCategory, 0, len(categoryNames))
		for _, categoryName := range categoryNames {
			products := make([]*db.CatalogProd, 0, len(listings[categoryName]))
			for _, product := range listings[categoryName] {
				if product != nil {
					products = append(products, product)
				}
			}
			sort.SliceStable(products, func(i, j int) bool {
				return catalogAlphabeticalLess(products[i].Name, products[j].Name)
			})
			if len(products) > 4 {
				products = products[:4]
			}

			responseProducts := make([]publicAPICatalogProduct, 0, len(products))
			for _, product := range products {
				var imageFilename *string
				if product.ImageURL != "" {
					filename := product.ImageURL
					imageFilename = &filename
				}

				responseProducts = append(responseProducts, publicAPICatalogProduct{
					ID:            product.ID,
					Name:          product.Name,
					Slug:          product.Slug,
					Description:   product.Description,
					ImageFilename: imageFilename,
				})
			}

			categories = append(categories, publicAPICatalogCategory{
				Name:     categoryName,
				Products: responseProducts,
			})
		}

		writePublicAPIJSON(w, http.StatusOK, publicAPICatalogListingsResponse{
			Categories: categories,
		})
	}
}

func GetPublicAPICatalogCategories(w http.ResponseWriter, r *http.Request) {
	getPublicAPICatalogCategoriesHandler(func() ([]*db.CatalogCtg, error) {
		return db.FindCatalogCategories("")
	})(w, r)
}

func getPublicAPICatalogCategoriesHandler(loader publicCatalogCategoriesLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		categories, err := loader()
		if err != nil {
			writePublicAPICatalogJSON(w, http.StatusInternalServerError, publicAPIErrorResponse{
				Error: "catalog_unavailable",
			})
			return
		}

		response := make([]publicAPICatalogCategorySummary, 0, len(categories))
		for _, category := range categories {
			if category == nil {
				continue
			}

			name := strings.TrimSpace(category.Name)
			if category.ID == "" || name == "" {
				continue
			}

			response = append(response, publicAPICatalogCategorySummary{
				ID:           category.ID,
				Name:         name,
				ProductCount: category.ProductCount,
			})
		}

		sort.SliceStable(response, func(i, j int) bool {
			return catalogAlphabeticalLess(response[i].Name, response[j].Name)
		})

		writePublicAPICatalogJSON(w, http.StatusOK, publicAPICatalogCategoriesResponse{
			Categories: response,
		})
	}
}

func GetPublicAPICatalogProducts(w http.ResponseWriter, r *http.Request) {
	getPublicAPICatalogProductsHandler(db.FindCatalogProducts)(w, r)
}

func getPublicAPICatalogProductsHandler(loader publicCatalogProductsLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		search := strings.TrimSpace(query.Get("buscar"))
		category := strings.TrimSpace(query.Get("categoria"))

		fieldErrors := make(map[string]string)

		page, ok := parsePublicAPICatalogPage(query.Get("pagina"))
		if !ok {
			fieldErrors["pagina"] = "Debe ser un entero mayor o igual a 1."
		}

		pageSize, ok := parsePublicAPICatalogPageSize(query.Get("por_pagina"))
		if !ok {
			fieldErrors["por_pagina"] = "Debe ser un entero entre 1 y 100."
		}

		if len(fieldErrors) > 0 {
			writePublicAPICatalogJSON(w, http.StatusBadRequest, publicAPIInvalidParametersResponse{
				Error:  "invalid_parameters",
				Fields: fieldErrors,
			})
			return
		}

		result, err := loader(category, search, page, pageSize)
		if err != nil || result == nil {
			writePublicAPICatalogJSON(w, http.StatusInternalServerError, publicAPIErrorResponse{
				Error: "catalog_unavailable",
			})
			return
		}

		items := make([]publicAPICatalogListItem, 0, len(result.Products))
		for _, product := range result.Products {
			if product == nil {
				continue
			}

			var imageFilename *string
			if filename := strings.TrimSpace(product.ImageURL); filename != "" {
				imageFilename = &filename
			}

			items = append(items, publicAPICatalogListItem{
				ID:            product.ID,
				Name:          product.Name,
				Slug:          product.Slug,
				Description:   product.Description,
				CategoryID:    product.CategoryID,
				CategoryName:  product.CategoryName,
				ImageFilename: imageFilename,
				Available:     product.Available,
			})
		}

		var categoryFilter *string
		if category != "" {
			value := category
			categoryFilter = &value
		}

		writePublicAPICatalogJSON(w, http.StatusOK, publicAPICatalogProductsResponse{
			Items: items,
			Pagination: publicAPICatalogPagination{
				Page:        result.Page,
				PageSize:    result.Limit,
				TotalItems:  result.Total,
				TotalPages:  result.TotalPages,
				HasNext:     result.HasNext,
				HasPrevious: result.HasPrevious,
			},
			Filters: publicAPICatalogFilters{
				Query:    search,
				Category: categoryFilter,
			},
		})
	}
}

// parsePublicAPICatalogPage accepts an empty value as the first page and
// rejects anything that is not an integer greater than or equal to 1.
func parsePublicAPICatalogPage(value string) (int, bool) {
	if value == "" {
		return 1, true
	}

	page, err := strconv.Atoi(value)
	if err != nil || page < 1 {
		return 0, false
	}

	return page, true
}

// parsePublicAPICatalogPageSize accepts an empty value as the default page
// size and rejects anything outside the 1-100 range enforced by the database
// layer.
func parsePublicAPICatalogPageSize(value string) (int, bool) {
	if value == "" {
		return db.DefaultCatalogPageSize, true
	}

	pageSize, err := strconv.Atoi(value)
	if err != nil || pageSize < 1 || pageSize > 100 {
		return 0, false
	}

	return pageSize, true
}

func PostPublicAPIContactRequest(w http.ResponseWriter, r *http.Request) {
	postPublicAPIContactRequestHandler(db.CreateQuote)(w, r)
}

func postPublicAPIContactRequestHandler(creator publicContactRequestCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || !strings.EqualFold(mediaType, "application/json") {
			writePublicAPIContactJSON(w, http.StatusUnsupportedMediaType, publicAPIErrorResponse{
				Error: "unsupported_media_type",
			})
			return
		}

		if r.ContentLength > publicContactRequestMaxBodyBytes {
			writePublicAPIContactJSON(w, http.StatusRequestEntityTooLarge, publicAPIErrorResponse{
				Error: "request_too_large",
			})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, publicContactRequestMaxBodyBytes)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		var request *publicAPIContactRequest
		if err = decoder.Decode(&request); err != nil {
			writePublicAPIContactDecodeError(w, err)
			return
		}
		if request == nil {
			writePublicAPIContactJSON(w, http.StatusBadRequest, publicAPIErrorResponse{
				Error: "invalid_request",
			})
			return
		}
		if err = decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writePublicAPIContactDecodeError(w, err)
			return
		}

		request.Name = strings.TrimSpace(request.Name)
		request.Phone = strings.TrimSpace(request.Phone)
		fieldErrors := validatePublicAPIContactRequest(*request)
		if fieldErrors.Name != "" || fieldErrors.Phone != "" {
			writePublicAPIContactJSON(w, http.StatusBadRequest, publicAPIContactValidationResponse{
				Error:  "validation_failed",
				Fields: fieldErrors,
			})
			return
		}

		quote := &db.Quote{
			CustomerName:  request.Name,
			CustomerPhone: request.Phone,
			RequestType:   db.QuoteRequestTypeContact,
			Status:        db.QuoteStatusPending,
		}
		if err = creator(quote); err != nil {
			writePublicAPIContactJSON(w, http.StatusInternalServerError, publicAPIErrorResponse{
				Error: "contact_unavailable",
			})
			return
		}

		writePublicAPIContactJSON(w, http.StatusCreated, publicAPIContactSuccessResponse{
			OK:      true,
			Message: "Solicitud recibida",
		})
	}
}

func writePublicAPIContactDecodeError(w http.ResponseWriter, err error) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		writePublicAPIContactJSON(w, http.StatusRequestEntityTooLarge, publicAPIErrorResponse{
			Error: "request_too_large",
		})
		return
	}

	writePublicAPIContactJSON(w, http.StatusBadRequest, publicAPIErrorResponse{
		Error: "invalid_request",
	})
}

func validatePublicAPIContactRequest(request publicAPIContactRequest) publicAPIContactFieldErrors {
	fieldErrors := publicAPIContactFieldErrors{}
	nameLength := utf8.RuneCountInString(request.Name)
	if nameLength == 0 {
		fieldErrors.Name = "El nombre es obligatorio."
	} else if nameLength < 2 {
		fieldErrors.Name = "El nombre debe tener al menos 2 caracteres."
	} else if nameLength > 120 {
		fieldErrors.Name = "El nombre no puede exceder 120 caracteres."
	}

	if request.Phone == "" {
		fieldErrors.Phone = "El teléfono es obligatorio."
		return fieldErrors
	}

	digitCount := 0
	validCharacters := utf8.RuneCountInString(request.Phone) <= 32
	for _, character := range request.Phone {
		if character >= '0' && character <= '9' {
			digitCount++
			continue
		}
		if character != ' ' && character != '+' && character != '-' && character != '(' && character != ')' {
			validCharacters = false
		}
	}
	if !validCharacters || digitCount < 10 || digitCount > 15 {
		fieldErrors.Phone = "Ingresa un número de teléfono válido de entre 10 y 15 dígitos."
	}

	return fieldErrors
}

func catalogAlphabeticalLess(left string, right string) bool {
	leftKey := catalogAlphabeticalKey(left)
	rightKey := catalogAlphabeticalKey(right)
	if leftKey == rightKey {
		return left < right
	}
	return leftKey < rightKey
}

func catalogAlphabeticalKey(value string) string {
	decomposed := norm.NFD.String(strings.ToLower(value))
	return strings.Map(func(character rune) rune {
		if unicode.Is(unicode.Mn, character) {
			return -1
		}
		return character
	}, decomposed)
}

func writePublicAPIJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writePublicAPICatalogJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	writePublicAPIJSON(w, status, value)
}

func writePublicAPIContactJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	writePublicAPIJSON(w, status, value)
}
