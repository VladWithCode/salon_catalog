package routes

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/qrgen"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/templates/util"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterProductsRoutes(router *customServeMux) {
	router.HandleFunc("GET /panel/qrcodes/productos", auth.ValidateAuth(DownloadAllQrCodes))
	router.HandleFunc("POST /panel/qrcodes/productos", auth.ValidateAuth(GenerateAllQrCodes))
	router.HandleFunc("GET /panel/qrcodes/productos/{id}", auth.ValidateAuth(GetProductQrCode))
	router.HandleFunc("PUT /panel/qrcodes/productos/{id}", auth.ValidateAuth(UpdateQrCode))

	router.HandleFunc("GET /panel/productos/table", auth.ValidateAuth(RenderProductsTable))
	router.HandleFunc("POST /panel/productos/nuevo", auth.ValidateAuth(CreateProductAndReturnTable))
	router.HandleFunc("GET /panel/productos/modal/nuevo", auth.ValidateAuth(RenderNewProductForm))
	router.HandleFunc("GET /panel/productos/modal/{id}", auth.ValidateAuth(RenderProduct))
	router.HandleFunc("PUT /panel/productos/{id}", auth.ValidateAuth(UpdateProductAndReturnTable))
	router.HandleFunc("DELETE /panel/productos", auth.ValidateAuth(DeleteProductsAndReturnTable))
	router.HandleFunc("DELETE /panel/productos/{id}", auth.ValidateAuth(DeleteProductAndReturnTable))

	// Image routes
	router.HandleFunc("PUT /panel/productos/{id}/main_img", auth.ValidateAuth(UpdateProductMainImg))
	router.HandleFunc("PUT /panel/productos/{id}/gallery", auth.ValidateAuth(UpdateProductGallery))
	router.HandleFunc("DELETE /panel/productos/{id}/main_img", auth.ValidateAuth(DeleteProductMainImg))
	router.HandleFunc("DELETE /panel/productos/{id}/gallery", auth.ValidateAuth(UpdateProductGallery))

	// Legacy API routes
	router.HandleFunc("GET /api/products", GetProducts)
	router.HandleFunc("GET /api/products/table", GetProductsTable)
	router.HandleFunc("GET /api/products/list", GetProductsList)
	router.HandleFunc("GET /api/products/{slug}", GetProductBySlug)

	router.HandleFunc("POST /api/products", auth.ValidateAuth(CreateProduct))
	router.HandleFunc("PUT /api/products/{id}", auth.ValidateAuth(UpdateProduct))
	// router.HandleFunc("PUT /api/products/{id}/images", auth.ValidateAuth(UpdateProductImages))
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

func GenerateAllQrCodes(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se generaron los códigos QR", components.ToastSuccess, 3000, true, false)

	// Read the QR codes directory
	filters := db.ProductFilterParams{Available: 1}
	products, err := db.FilterProducts(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar los productos"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to get products: %v\n", err)
		return
	}

	createFails := 0
	for _, product := range products.Products {
		qrData := &qrgen.QRCodeData{
			Filename: product.Slug,
			Value:    fmt.Sprintf("https://villachenacolo.com/catalogo/producto/%s", product.ID),
		}
		qrCodeFilename, err := qrgen.GenerateFromString(qrData)
		if err != nil {
			createFails++
			continue
		}
		product.QRCodeFilename = qrCodeFilename
	}

	err = db.UpdateProductBatch(products.Products)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar los productos"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to update products: %v\n", err)
		return
	}

	warnToastData := components.NewToastData(
		fmt.Sprintf("%d códigos QR no se han podido generar", createFails),
		components.ToastWarning,
		3000,
		true,
		false,
	)
	err = templ.Join(
		components.ToasterToast(warnToastData),
		components.ToasterToast(toastData),
	).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render QR code success: %v\n", err)
		return
	}
}

func DownloadAllQrCodes(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se descargaron los códigos QR", components.ToastSuccess, 3000, true, false)
	qrcodesDir := "web/static/qrcodes"

	// Read the QR codes directory
	files, err := os.ReadDir(qrcodesDir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al leer el directorio de códigos QR"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to read qrcodes directory: %v\n", err)
		return
	}

	// Filter for QR code files only
	var qrcodeFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), "-qrcode.jpg") {
			qrcodeFiles = append(qrcodeFiles, file.Name())
		}
	}

	if len(qrcodeFiles) == 0 {
		w.WriteHeader(http.StatusNotFound)
		toastData.Message = "No se encontraron códigos QR"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		return
	}

	// Set headers for file download
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("product-qrcodes_%s.zip", timestamp)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Create ZIP writer that writes directly to response
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Add each QR code file to the ZIP
	for _, qrcodeFile := range qrcodeFiles {
		filePath := filepath.Join(qrcodesDir, qrcodeFile)

		// Open the file
		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("failed to open file %s: %v\n", filePath, err)
			continue
		}

		// Create a file in the ZIP
		zipFile, err := zipWriter.Create(qrcodeFile)
		if err != nil {
			file.Close()
			log.Printf("failed to create zip entry for %s: %v\n", qrcodeFile, err)
			continue
		}

		// Copy file content to ZIP
		_, err = io.Copy(zipFile, file)
		file.Close()
		if err != nil {
			log.Printf("failed to copy file %s to zip: %v\n", qrcodeFile, err)
			continue
		}
	}

	components.ToasterToast(toastData).Render(r.Context(), w)
	log.Printf("Successfully created QR codes ZIP with %d files for admin user %s", len(qrcodeFiles), a.ID)
}

func GetProductQrCode(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
}

func UpdateQrCode(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se ha generado el código QR", components.ToastSuccess, 3000, true, false)

	product, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar el producto"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductQRCode(product, dashboard.ProductQRCodeState{HasError: true, ErrorMsg: "Error al recuperar el producto"}),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "qrcodeTabError", "toaster-toast")
		log.Printf("failed to get product: %v\n", err)
		return
	}

	qrData := &qrgen.QRCodeData{
		Filename: product.Slug,
		Value:    fmt.Sprintf("https://villachenacolo.com/catalogo/producto/%s", product.ID),
	}
	qrCodeFilename, err := qrgen.GenerateFromString(qrData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al generar el código QR"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductQRCode(product, dashboard.ProductQRCodeState{HasError: true, ErrorMsg: "Error al generar el código QR"}),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "qrcodeTabError", "toaster-toast")
		log.Printf("failed to generate QR code: %v\n", err)
		return
	}

	product.QRCodeFilename = qrCodeFilename
	product.MainImg = product.MainImgID
	product.Gallery = product.GalleryIDs
	err = db.UpdateProduct(product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el producto"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ProductQRCode(product, dashboard.ProductQRCodeState{HasError: true, ErrorMsg: "Error al actualizar el producto"}),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "qrcodeTabError", "toaster-toast")
		log.Printf("failed to update product: %v\n", err)
		return
	}

	err = templ.Join(
		dashboard.ProductQRCode(product, dashboard.ProductQRCodeState{HasSuccess: true, SuccessMsg: "Código QR actualizado exitosamente"}),
		components.ToasterToast(toastData),
	).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render QR code success: %v\n", err)
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
	params.Available, _ = strconv.Atoi(r.FormValue("available"))
	params.WithQRCode, _ = strconv.Atoi(r.FormValue("with_qr_code"))

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

func UpdateProductMainImg(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	productId := r.PathValue("id")
	toastData := components.NewToastData("Se actualizó la imagen principal", components.ToastSuccess, 3000, true, false)

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImageSelectorModal(
				dashboard.ImageSelectorConfig{},
				&db.ImageFilterResult{HasError: true, Error: "Error al procesar el formulario"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "imagesGrid", "toaster-toast")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	var config dashboard.ImageSelectorConfig
	rawConf := r.FormValue("selectorConfig")
	if rawConf != "" {
		err = config.FromJSONString(rawConf)
		if err != nil {
			config = dashboard.ImageSelectorConfig{
				Mode:           "single",
				UpdateEndpoint: "/panel/productos/" + productId + "/main_img",
				MaxSelection:   1,
			}
		}
	}

	product, err := db.FindProductByID(productId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar el producto"
		toastData.Type = components.ToastError
		combined := templ.Join(
			dashboard.ImageSelectorModal(
				config,
				&db.ImageFilterResult{HasError: true, Error: "Error al recuperar el producto"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, combined, "imagesGrid", "toaster-toast")
		log.Printf("failed to get product: %v\n", err)
		return
	}

	selectedImg := r.FormValue("selected")
	findFn := db.FindImageByID
	if _, err = uuid.Parse(selectedImg); err != nil {
		findFn = db.FindImageByFilename
	}

	img, err := findFn(selectedImg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar la imagen"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImageSelectorModal(
				dashboard.ImageSelectorConfig{},
				&db.ImageFilterResult{HasError: true, Error: "Error al recuperar la imagen"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "imagesGrid", "toaster-toast")
		log.Printf("failed to get image: %v\n", err)
		return
	}

	product.MainImg = img.ID
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

	updateData := &dashboard.ProductImagesTabUpdateData{
		MainImgFilename: img.Filename,
	}
	combined := templ.Join(
		dashboard.ProductImagesTab(product, updateData),
		components.ToasterToast(toastData),
	)
	err = templ.RenderFragments(r.Context(), w, combined, "mainImgSection", "toaster-toast")
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render images tab: %v\n", err)
		return
	}
}

func DeleteProductMainImg(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	productId := r.PathValue("id")
	toastData := components.NewToastData("Se eliminó la imagen principal", components.ToastSuccess, 3000, true, false)
	updateData := &dashboard.ProductImagesTabUpdateData{
		MainImgFilename: "",
		MainImgID:       "",
	}
	mixedFragmentsConf := util.WithMixedFragmentsConf{}

	product, err := db.FindProductByID(productId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		toastData.Message = "Error al recuperar el producto"
		toastData.Type = components.ToastError
		mixedFragmentsConf.
			AppendJoinTemplates(components.ToasterToast(toastData)).
			SetFragments("toaster-toast")
		util.RenderMixedWithFragments(r.Context(), w, mixedFragmentsConf)
		log.Printf("failed to find product: %v\n", err)
		return
	}

	product.MainImg = ""
	product.MainImgID = ""
	err = db.UpdateProduct(product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		toastData.Message = "Error al eliminar la imagen principal"
		toastData.Type = components.ToastError
		mixedFragmentsConf.
			AppendJoinTemplates(components.ToasterToast(toastData)).
			SetFragments("toaster-toast")
		util.RenderMixedWithFragments(r.Context(), w, mixedFragmentsConf)
		log.Printf("failed to unlink images from product: %v\n", err)
		return
	}

	mixedFragmentsConf.
		AppendJoinTemplates(components.ToasterToast(toastData), dashboard.ProductImagesTab(product, updateData)).
		SetFragments("toaster-toast", "mainImgSection")
	util.RenderMixedWithFragments(r.Context(), w, mixedFragmentsConf)
}

func UpdateProductGallery(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	productId := r.PathValue("id")
	toastData := components.NewToastData("Se actualizó la galería de imágenes", components.ToastSuccess, 3000, true, false)
	selectorConfig := dashboard.ImageSelectorConfig{
		Mode:           "multiple",
		UpdateEndpoint: "/panel/productos/" + productId + "/gallery",
		MaxSelection:   10,
		SuccessTarget:  "#products-modal-images-gallery",
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImageSelectorModal(
				selectorConfig,
				&db.ImageFilterResult{HasError: true, Error: "Error al procesar el formulario"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "imagesGridError", "toaster-toast")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	var config dashboard.ImageSelectorConfig
	rawConf := r.FormValue("selectorConfig")
	if rawConf != "" {
		err = config.FromJSONString(rawConf)
		if err != nil {
			config = dashboard.ImageSelectorConfig{
				Mode:           "multiple",
				UpdateEndpoint: "/panel/productos/" + productId + "/gallery",
				MaxSelection:   10,
			}
		}
	}

	product, err := db.FindProductByID(productId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar el producto"
		toastData.Type = components.ToastError
		combined := templ.Join(
			dashboard.ImageSelectorModal(
				config,
				&db.ImageFilterResult{HasError: true, Error: "Error al recuperar el producto"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, combined, "imagesGridError", "toaster-toast")
		log.Printf("failed to get product: %v\n", err)
		return
	}

	selectedImgs := r.FormValue("selected")
	if len(selectedImgs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "No se encontraron imágenes"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImageSelectorModal(
				config,
				&db.ImageFilterResult{HasError: true, Error: "No se encontraron imágenes"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "imagesGridError", "toaster-toast")
		log.Printf("failed to get selected images: %v\n", err)
		return
	}

	imgIds := strings.Split(selectedImgs, ",")
	imgs, err := db.FindAllImages(imgIds)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar la imagen"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImageSelectorModal(
				dashboard.ImageSelectorConfig{},
				&db.ImageFilterResult{HasError: true, Error: "Error al recuperar la imagen"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "imagesGridError", "toaster-toast")
		log.Printf("failed to get image: %v\n", err)
		return
	}

	product.Gallery = []string{}
	product.GalleryIDs = []string{}
	for _, img := range imgs {
		product.Gallery = append(product.Gallery, img.Filename)
		product.GalleryIDs = append(product.GalleryIDs, img.ID)
	}

	updateData := &dashboard.ProductImagesTabUpdateData{
		GalleryIDs: product.GalleryIDs,
		Gallery:    product.Gallery,
	}
	err = db.UpdateProductImages(product.ID, product.GalleryIDs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el producto"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImageSelectorModal(
				dashboard.ImageSelectorConfig{},
				&db.ImageFilterResult{HasError: true, Error: "Error al recuperar la imagen"},
			),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comp, "imagesGridError", "toaster-toast")
		log.Printf("failed to update product: %v\n", err)
		return
	}

	combined := templ.Join(
		dashboard.ProductImagesTab(product, updateData),
		components.ToasterToast(toastData),
	)
	err = templ.RenderFragments(r.Context(), w, combined, "gallerySection", "toaster-toast")
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render images tab: %v\n", err)
		return
	}
}
