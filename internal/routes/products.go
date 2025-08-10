package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/qrgen"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterProductsRoutes(router *customServeMux) {
	// HTMX routes that respond with templ components (for AJAX requests)
	router.HandleFunc("GET /panel/productos/table", auth.ValidateAuth(RenderProductsTable))
	router.HandleFunc("GET /panel/productos/modal/nuevo", auth.ValidateAuth(RenderNewProductForm))
	router.HandleFunc("POST /panel/productos/nuevo", auth.ValidateAuth(CreateProductAndReturnTable))
	router.HandleFunc("GET /panel/productos/modal/{id}", auth.ValidateAuth(RenderProduct))
	router.HandleFunc("PUT /panel/productos/{id}", auth.ValidateAuth(UpdateProductAndReturnTable))
	router.HandleFunc("PUT /panel/productos/{id}/qrcode", auth.ValidateAuth(UpdateProductQRCodeAndReturnTable))
	router.HandleFunc("DELETE /panel/productos", auth.ValidateAuth(DeleteProductsAndReturnTable))
	router.HandleFunc("DELETE /panel/productos/{id}", auth.ValidateAuth(DeleteProductAndReturnTable))

	// Legacy API routes
	router.HandleFunc("GET /api/products", GetProducts)
	router.HandleFunc("GET /api/products/table", GetProductsTable)
	router.HandleFunc("GET /api/products/list", GetProductsList)
	router.HandleFunc("GET /api/products/{slug}", GetProductBySlug)

	router.HandleFunc("POST /api/products", auth.ValidateAuth(CreateProduct))
	router.HandleFunc("PUT /api/products/{id}", auth.ValidateAuth(UpdateProduct))
	router.HandleFunc("PUT /api/products/{id}/images", auth.ValidateAuth(UpdateProductImages))
	router.HandleFunc("DELETE /api/products/{id}", auth.ValidateAuth(DeleteProduct))
	router.HandleFunc("DELETE /api/products/{id}/images", auth.ValidateAuth(DeleteProductImages))

	router.HandleFunc("POST /api/special/products", auth.ValidateAuth(sCreateProducts))
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
	prods, err := db.FindAllProducts()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find products"))
		log.Printf("failed to find products: %v\n", err)
		return
	}

	data, err := json.Marshal(prods)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal products"))
		log.Printf("failed to marshal products: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func GetProductsTable(w http.ResponseWriter, r *http.Request) {
	params, err := parseProductFilterParams(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	prods, err := db.FilterProducts(*params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find products"))
		log.Printf("failed to find products: %v\n", err)
		return
	}

	err = dashboard.ProductsTable(prods).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render ProductsTable err: %v\n", err)
		return
	}
}

func GetProductsList(w http.ResponseWriter, r *http.Request) {
	prods, err := db.FindAllProducts()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find products"))
		log.Printf("failed to find products: %v\n", err)
		return
	}

	data, err := json.Marshal(prods)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal products"))
		log.Printf("failed to marshal products: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	prod, err := db.FindProductBySlug(slug)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	data, err := json.Marshal(prod)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal product"))
		log.Printf("failed to marshal product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func CreateProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	var data map[string]string
	rawCtgs, _ := db.FindAllCategories()
	ctgs := internal.PtrSliceToPlainSlice(rawCtgs)

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		state := forms.NewProductFormStateFromMap("create", data)
		state.GeneralError = "Algo salió mal"
		dashboard.CreateProductForm(state, ctgs).Render(r.Context(), w)

		log.Printf("failed to parse request body: %v\n", err)
		return
	}
	defer r.Body.Close()

	formState := forms.NewProductFormStateFromMap("create", data)
	err = formState.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		formState.GeneralError = "El formulario contiene errores"
		dashboard.CreateProductForm(formState, ctgs).Render(r.Context(), w)
		log.Printf("failed to render CreateProduct err: %v\n", err)
		return
	}
	var product db.Product
	product.Name = formState.Values.Name
	product.Description = formState.Values.Description
	product.LongDescription = formState.Values.LongDescription
	product.MainImg = formState.Values.MainImg
	product.CategoryID = formState.Values.CategoryID
	product.Available = formState.Values.Available
	product.Gallery = formState.Values.Gallery

	product.Slug = internal.Slugify(product.Name)

	err = db.CreateProduct(&product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		formState.ResetFieldState()
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			formState.GeneralError = "El formulario contiene errores"
			formState.SetFieldError("name", "El nombre ya existe")
		} else {
			formState.GeneralError = "Algo salió mal"
		}
		dashboard.CreateProductForm(formState, ctgs).Render(r.Context(), w)

		log.Printf("failed to create product: %v\n", err)
		return
	}

	formState.SuccessMessage = "Producto creado exitosamente"
	dashboard.CreateProductForm(formState, ctgs).Render(r.Context(), w)
}

func UpdateProductQRCodeAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se actualizó el producto", components.ToastSuccess, 3000, true, false)
	product, _ := db.FindProductByID(id)

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductQRCode(product, true),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Update product fields
	product.QRCodeFilename = strings.TrimSpace(r.FormValue("qrcode_filename"))

	// Validate required fields
	if product.QRCodeFilename == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Nombre, slug, descripción y categoría son requeridos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Campos requeridos faltantes"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Update product in database
	err = db.UpdateProduct(product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el producto"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al actualizar el producto"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to update product: %v\n", err)
		return
	}

	// Get updated products list
	filters, _ := parseProductFilterParams(r)
	result, err := db.FilterProducts(*filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar los productos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al recuperar los productos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get products after update: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.ProductsTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func UpdateProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	var data db.Product
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	id := r.PathValue("id")
	prod, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	if data.Name != "" {
		prod.Name = data.Name
		prod.Slug = internal.Slugify(prod.Name)
	}
	if data.Description != "" {
		prod.Description = data.Description
	}
	if data.Category != "" {
		prod.Category = data.Category
	}

	if data.MainImg != "" {
		if data.MainImg == uploads.RemoveImgFlag {
			prod.MainImg = ""
		}

		prod.MainImg = data.MainImg
	}

	err = db.UpdateProduct(prod)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to update product"))
		log.Printf("failed to update product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateProductImages(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	var imgIDs []string
	err := json.NewDecoder(r.Body).Decode(&imgIDs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	prod, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	err = db.LinkImagesToProduct(imgIDs, prod.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to link images to product"))
		log.Printf("failed to link images to product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	err := db.DeleteProduct(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete product"))
		log.Printf("failed to delete product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteProductImages(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	var imgIDs []string
	err := json.NewDecoder(r.Body).Decode(&imgIDs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	prod, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	err = db.UnlinkImagesFromProduct(imgIDs, prod.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to unlink images from product"))
		log.Printf("failed to unlink images from product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func sCreateProducts(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	products := make([]*db.Product, 0)
	err := json.NewDecoder(r.Body).Decode(&products)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	err = db.SCreateProducts(products)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to create products: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Products created successfully"))
}

func parseProductFilterParams(r *http.Request) (*db.ProductFilterParams, error) {
	params := &db.ProductFilterParams{}
	params.Search = r.URL.Query().Get("search")
	params.Category = r.URL.Query().Get("category")
	params.Sort = r.URL.Query().Get("sort")
	params.Page, _ = strconv.Atoi(r.FormValue("page"))
	params.Limit, _ = strconv.Atoi(r.FormValue("limit"))

	return params, nil
}

func RenderNewProductForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	tCtgs, err := db.FindAllCategories()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find categories"))
		log.Printf("failed to find categories: %v\n", err)
		return
	}
	ctgs := internal.PtrSliceToPlainSlice(tCtgs)
	// If request is AJAX, render components
	if r.Header.Get("HX-Request") == "true" {
		component := dashboard.ProductCreateModal(ctgs, "")
		component.Render(r.Context(), w)
	} else {
		// Else render page
		// TODO: render page
	}
}

func CreateProductAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se creó el producto exitosamente", components.ToastSuccess, 3000, true, false)
	tCtgs, _ := db.FindAllCategories()
	ctgs := internal.PtrSliceToPlainSlice(tCtgs)

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductCreateModal(ctgs, "Error al procesar el formulario"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Get form values
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	longDescription := strings.TrimSpace(r.FormValue("long_description"))
	categoryID := strings.TrimSpace(r.FormValue("category"))
	quantityStr := strings.TrimSpace(r.FormValue("quantity"))
	availableStr := r.FormValue("available")

	// Validate required fields
	if name == "" || description == "" || categoryID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Nombre, descripción y categoría son requeridos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductCreateModal(ctgs, "Nombre, descripción y categoría son requeridos"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Parse quantity
	quantity := 0
	if quantityStr != "" {
		quantity, _ = strconv.Atoi(quantityStr)
	}

	// Parse availability
	available := availableStr == "on"

	// Create new product
	product := &db.Product{
		Name:            name,
		Slug:            internal.Slugify(name),
		Description:     description,
		LongDescription: longDescription,
		CategoryID:      categoryID,
		Available:       available,
		Quantity:        quantity,
	}

	qrData := &qrgen.QRCodeData{
		Filename: product.Slug,
		Value:    fmt.Sprintf("https://salon.chenacolo.com/panel/productos/%s", product.ID),
	}
	product.QRCodeFilename, _ = qrgen.GenerateFromString(qrData)

	// Create product in database
	err = db.CreateProduct(product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al crear el producto"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductCreateModal(ctgs, "Error al crear el producto"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to create product: %v\n", err)
		return
	}

	// Get updated products list
	filters, _ := parseProductFilterParams(r)
	result, err := db.FilterProducts(*filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Producto creado. Ocurrió un error al recuperar los productos, actualice la página."
		toastData.Type = components.ToastWarning
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al recuperar los productos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get products after create: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.ProductsTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func RenderProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	product, err := db.FindProductByID(id)

	// If request is AJAX, render components
	if r.Header.Get("HX-Request") == "true" {
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			dashboard.ProductModal(product, nil, "Producto no encontrado").Render(r.Context(), w)
			log.Printf("failed to find product: %v\n", err)
			return
		}
		ctgs, err := db.FindAllCategories()
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			dashboard.ProductModal(product, nil, "Algo salió mal").Render(r.Context(), w)
			log.Printf("failed to find product: %v\n", err)
			return
		}

		component := dashboard.ProductModal(product, internal.PtrSliceToPlainSlice(ctgs), "")
		component.Render(r.Context(), w)
	} else {
		// Else render page
		// TODO: render page
	}
}

func RenderProductsTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Parse query parameters
	filters, err := parseProductFilterParams(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to parse filters"))
		log.Printf("failed to parse filters: %v\n", err)
		return
	}

	// Get filtered products
	result, err := db.FilterProducts(*filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get products"))
		log.Printf("failed to filter products: %v\n", err)
		return
	}

	// Render the template with the results
	component := dashboard.ProductsTable(result)
	component.Render(r.Context(), w)
}

func UpdateProductAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se actualizó el producto", components.ToastSuccess, 3000, true, false)

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Get existing product
	product, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		toastData.Message = "Producto no encontrado"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Producto no encontrado"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to find product: %v\n", err)
		return
	}

	// Update product fields
	product.Name = strings.TrimSpace(r.FormValue("name"))
	product.Slug = strings.TrimSpace(r.FormValue("slug"))
	product.Description = strings.TrimSpace(r.FormValue("description"))
	product.LongDescription = strings.TrimSpace(r.FormValue("long_description"))
	product.CategoryID = strings.TrimSpace(r.FormValue("category"))

	quantityStr := strings.TrimSpace(r.FormValue("quantity"))
	if quantityStr != "" {
		product.Quantity, _ = strconv.Atoi(quantityStr)
	}

	product.Available = r.FormValue("available") == "on"

	// Validate required fields
	if product.Name == "" || product.Slug == "" || product.Description == "" || product.CategoryID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Nombre, slug, descripción y categoría son requeridos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Campos requeridos faltantes"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Update product in database
	err = db.UpdateProduct(product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el producto"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al actualizar el producto"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to update product: %v\n", err)
		return
	}

	// Get updated products list
	filters, _ := parseProductFilterParams(r)
	result, err := db.FilterProducts(*filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar los productos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al recuperar los productos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get products after update: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.ProductsTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func DeleteProductAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminó el producto", components.ToastSuccess, 3000, true, false)

	err := db.DeleteProduct(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al eliminar el producto"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al eliminar el producto"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to delete product: %v\n", err)
		return
	}

	filters, _ := parseProductFilterParams(r)
	result, err := db.FilterProducts(*filters)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar los productos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al recuperar los productos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get products after delete: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.ProductsTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func DeleteProductsAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	td := components.NewToastData("Se eliminaron los productos", components.ToastSuccess, 3000, true, false)

	var err error
	productIds := r.URL.Query()["ids"]
	if len(productIds) > 0 {
		for _, id := range productIds {
			err = db.DeleteProduct(id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				td.Message = "Error al eliminar los productos"
				td.Type = components.ToastError
				comp := templ.Join(
					dashboard.ProductsTable(&db.ProductFilterResult{HasError: true, Error: "Error al eliminar los productos"}),
					components.ToasterToast(td),
				)
				comp.Render(r.Context(), w)
				log.Printf("failed to delete products: %v\n", err)
				return
			}
		}
	}

	filters, _ := parseProductFilterParams(r)
	result, err := db.FilterProducts(*filters)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusInternalServerError)
		td.Message = "Error al recuperar los productos"
		td.Type = components.ToastError
		result := &db.ProductFilterResult{HasError: true, Error: "Error al recuperar los productos"}
		comp := templ.Join(
			dashboard.ProductsTable(result),
			components.ToasterToast(td),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get products after bulk delete: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.ProductsTable(result),
		components.ToasterToast(td),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}
