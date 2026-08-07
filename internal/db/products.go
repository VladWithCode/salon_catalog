package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladwithcode/salon_catalog/internal"
)

var (
	ErrProductInsert = errors.New("failed to insert product")
	ErrGalleryInsert = errors.New("failed to insert gallery images")
)

type Product struct {
	ID              string   `db:"id" json:"id"`
	Name            string   `db:"name" json:"name"`
	Slug            string   `db:"slug" json:"slug"`
	Description     string   `db:"description" json:"description"`
	LongDescription string   `db:"long_description" json:"longDescription"`
	MainImg         string   `db:"main_img" json:"mainImg"`
	MainImgID       string   `db:"main_img_id" json:"mainImgId"`
	Gallery         []string `db:"gallery" json:"gallery"`
	GalleryIDs      []string `db:"gallery_ids" json:"galleryIds"`
	Category        string   `db:"category" json:"category"`
	CategoryID      string   `db:"category_id" json:"categoryId"`
	Available       bool     `db:"available" json:"available"`
	Quantity        int      `db:"quantity" json:"quantity"`
	QRCodeFilename  string   `db:"qrcode_filename" json:"qrcodeFilename"`
}

// SearchMode defines how search should behave
type SearchMode string

const (
	SearchModeFullText SearchMode = "fulltext" // Use full-text search
	SearchModeExact    SearchMode = "exact"    // Use exact matching
	SearchModeFuzzy    SearchMode = "fuzzy"    // Use LIKE matching (fallback)
)

type ProductFilterParams struct {
	IDs        []string   `json:"ids"`
	Search     string     `json:"search"`
	SearchMode SearchMode `json:"search_mode"`
	Category   string     `json:"category"`
	Sort       string     `json:"sort"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	Available  int        `json:"available"` // -1 = unavailable, 0 = all, 1 = available
	Quantity   int        `json:"quantity"`
	WithQRCode int        `json:"with_qr_code"` // -1 = unavailable, 0 = all, 1 = available
}

type ProductFilterResult struct {
	Products    []*Product `json:"products"`
	Total       int        `json:"total"`
	Page        int        `json:"page"`
	Limit       int        `json:"limit"`
	TotalPages  int        `json:"total_pages"`
	HasNext     bool       `json:"has_next"`
	HasPrevious bool       `json:"has_previous"`
	HasError    bool       `json:"has_error"`
	Error       string     `json:"error"`
}

func CreateProduct(product *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := uuid.NewV7()
	if err != nil {
		return ErrUUIDFail
	}
	product.ID = id.String()
	mainImg := uuid.NullUUID{}
	if product.MainImg != "" {
		id, err := uuid.Parse(product.MainImg)
		mainImg.UUID = id
		mainImg.Valid = err == nil
	}

	args := pgx.NamedArgs{
		"id":               product.ID,
		"name":             product.Name,
		"slug":             product.Slug,
		"description":      product.Description,
		"long_description": product.LongDescription,
		"main_img":         mainImg,
		"category":         product.CategoryID,
		"available":        product.Available,
		"quantity":         product.Quantity,
		"qrcode_filename":  product.QRCodeFilename,
	}
	_, err = tx.Exec(
		ctx,
		`INSERT INTO products 
		(id, name, slug, description, long_description, main_img, category, available, quantity, qrcode_filename)
		VALUES (@id, @name, @slug, @description, @long_description, @main_img, @category, @available, @quantity, @qrcode_filename)`,
		args,
	)
	if err != nil {
		return errors.Join(ErrProductInsert, err)
	}

	for _, img := range product.Gallery {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO images_products (image_id, product_id) VALUES ($1, $2)`,
			img,
			product.ID,
		)
		if err != nil {
			return errors.Join(ErrGalleryInsert, err)
		}
	}

	tx.Commit(ctx)
	return nil
}

func FindProductBySlug(slug string) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var product Product
	var mainImg sql.NullString
	var mainImgID sql.NullString
	var gallery pgtype.Array[*string]
	var galleryIDs pgtype.Array[*string]
	var longDescription sql.NullString
	err = conn.QueryRow(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description, prod.long_description,
			ctg.name AS category,
			ctg.id AS category_id,
			main.filename AS main_img,
			main.id AS main_img_id,
			prod.available, prod.quantity,
			prod.qrcode_filename,
			ARRAY_AGG(img.filename) AS gallery,
			ARRAY_AGG(img.id) AS gallery_ids
		FROM products prod 
			LEFT JOIN images_products img_prod ON prod.id = img_prod.product_id
			LEFT JOIN images img ON img_prod.image_id = img.id
			LEFT JOIN images main ON main.id = prod.main_img
			LEFT JOIN categories ctg ON ctg.id = prod.category
		WHERE prod.slug = $1
		GROUP BY prod.id, prod.name, prod.slug, prod.description, prod.long_description, prod.available, prod.quantity, main.filename, main.id, ctg.name, ctg.id, prod.qrcode_filename`,
		slug,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&longDescription,
		&product.Category,
		&product.CategoryID,
		&mainImg,
		&mainImgID,
		&product.Available,
		&product.Quantity,
		&product.QRCodeFilename,
		&gallery,
		&galleryIDs,
	)
	if err != nil {
		return nil, err
	}

	if mainImg.Valid {
		product.MainImg = mainImg.String
	}
	if mainImgID.Valid {
		product.MainImgID = mainImgID.String
	}
	if gallery.Valid {
		for _, img := range gallery.Elements {
			if img != nil {
				product.Gallery = append(product.Gallery, *img)
			}
		}
	}
	if galleryIDs.Valid {
		for _, img := range galleryIDs.Elements {
			if img != nil {
				product.GalleryIDs = append(product.GalleryIDs, *img)
			}
		}
	}
	if longDescription.Valid {
		product.LongDescription = longDescription.String
	} else {
		product.LongDescription = product.Description
	}

	return &product, nil
}

func FindProductByID(id string) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var product Product
	var mainImg sql.NullString
	var mainImgID sql.NullString
	var gallery pgtype.Array[*string]
	var galleryIDs pgtype.Array[*string]
	var longDescription sql.NullString
	err = conn.QueryRow(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description, prod.long_description,
			ctg.name AS category,
			ctg.id AS category_id,
			main.filename AS main_img,
			main.id AS main_img_id,
			prod.available, prod.quantity,
			prod.qrcode_filename,
			ARRAY_AGG(img.filename) AS gallery,
			ARRAY_AGG(img.id) AS gallery_ids
		FROM products prod
			LEFT JOIN images_products img_prod ON prod.id = img_prod.product_id
			LEFT JOIN images img ON img_prod.image_id = img.id
			LEFT JOIN images main ON main.id = prod.main_img
			LEFT JOIN categories ctg ON ctg.id = prod.category
		WHERE prod.id = $1
		GROUP BY prod.id, prod.name, prod.slug, prod.description, prod.long_description, prod.available, prod.quantity, main.filename, main.id, ctg.name, ctg.id, prod.qrcode_filename`,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&longDescription,
		&product.Category,
		&product.CategoryID,
		&mainImg,
		&mainImgID,
		&product.Available,
		&product.Quantity,
		&product.QRCodeFilename,
		&gallery,
		&galleryIDs,
	)
	if err != nil {
		return nil, err
	}
	if mainImg.Valid {
		product.MainImg = mainImg.String
	}
	if mainImgID.Valid {
		product.MainImgID = mainImgID.String
	}
	if gallery.Valid {
		for _, img := range gallery.Elements {
			if img != nil {
				product.Gallery = append(product.Gallery, *img)
			}
		}
	}
	if galleryIDs.Valid {
		for _, img := range galleryIDs.Elements {
			if img != nil {
				product.GalleryIDs = append(product.GalleryIDs, *img)
			}
		}
	}
	if longDescription.Valid {
		product.LongDescription = longDescription.String
	} else {
		product.LongDescription = product.Description
	}

	return &product, nil
}

func FindAllProducts() ([]*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description, prod.long_description,
			ctg.name AS category,
			ctg.id AS category_id,
			img.filename AS main_img
			prod.available, prod.quantity,
			prod.qrcode_filename
		FROM products prod
			LEFT JOIN images img ON img.id = prod.main_img
			LEFT JOIN categories ctg ON ctg.id = prod.category`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		var product Product
		var mainImg sql.NullString
		var longDescription sql.NullString
		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.Slug,
			&product.Description,
			&longDescription,
			&product.Category,
			&product.CategoryID,
			&mainImg,
			&product.Available,
			&product.Quantity,
			&product.QRCodeFilename,
		)
		if err != nil {
			return nil, err
		}
		if mainImg.Valid {
			product.MainImg = mainImg.String
		}
		if longDescription.Valid {
			product.LongDescription = longDescription.String
		} else {
			product.LongDescription = product.Description
		}
		products = append(products, &product)
	}

	return products, nil
}

func UpdateProduct(product *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	mainImg := sql.NullString{
		String: product.MainImg,
		Valid:  product.MainImg != "",
	}
	if _, err := uuid.Parse(product.MainImg); err != nil {
		mainImg.String = product.MainImgID
		mainImg.Valid = product.MainImgID != ""
	}

	args := pgx.NamedArgs{
		"id":               product.ID,
		"name":             product.Name,
		"slug":             product.Slug,
		"description":      product.Description,
		"long_description": product.LongDescription,
		"category":         product.CategoryID,
		"main_img":         mainImg,
		"available":        product.Available,
		"quantity":         product.Quantity,
		"qrcode_filename":  product.QRCodeFilename,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE products SET
			name = @name, slug = @slug, description = @description,
			long_description = @long_description, category = @category,
			main_img = @main_img, available = @available, quantity = @quantity,
			qrcode_filename = @qrcode_filename
		WHERE id = @id`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func UpdateProductBatch(products []*Product) error {
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx := context.Background()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	batch := pgx.Batch{}
	for _, product := range products {
		args := pgx.NamedArgs{
			"name":             product.Name,
			"slug":             product.Slug,
			"description":      product.Description,
			"long_description": product.LongDescription,
			"category":         product.CategoryID,
			"available":        product.Available,
			"quantity":         product.Quantity,
			"qrcode_filename":  product.QRCodeFilename,
			"id":               product.ID,
		}
		batch.Queue(
			`UPDATE products SET
				name = @name, slug = @slug, description = @description, long_description = @long_description,
				category = @category, available = @available, quantity = @quantity, qrcode_filename = @qrcode_filename
			WHERE id = @id`,
			args,
		)
	}

	results := tx.SendBatch(ctx, &batch)
	defer results.Close()

	_, err = results.Exec()
	if err != nil {
		return err
	}

	results.Close()
	return tx.Commit(ctx)
}

func UpdateProductImages(productId string, imageIds []string) error {
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx := context.Background()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete existing product-image relationships
	_, err = tx.Exec(ctx, "DELETE FROM images_products WHERE product_id = $1", productId)
	if err != nil {
		return err
	}

	// Insert new product-image relationships
	for _, imageId := range imageIds {
		_, err = tx.Exec(ctx,
			"INSERT INTO images_products (image_id, product_id) VALUES ($1, $2)",
			imageId, productId)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func DeleteProduct(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM products WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}

func FilterProducts(filters ProductFilterParams) (*ProductFilterResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Set defaults
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.SearchMode == "" {
		filters.SearchMode = SearchModeFullText
	}

	// Build query conditions and named arguments
	conditions, namedArgs := buildQueryConditions(filters)

	// Base query with explicit column selection
	baseQuery := `
		FROM products prod
		LEFT JOIN categories ctg ON prod.category = ctg.id
		LEFT JOIN images_products imgs_prod ON imgs_prod.product_id = prod.id
		LEFT JOIN images imgs ON imgs_prod.image_id = imgs.id
		LEFT JOIN images img ON prod.main_img = img.id
		`
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err = conn.QueryRow(ctx, countQuery, namedArgs).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination
	offset := (filters.Page - 1) * filters.Limit
	totalPages := int(math.Ceil(float64(total) / float64(filters.Limit)))

	// Add pagination to named args
	namedArgs["limit"] = filters.Limit
	namedArgs["offset"] = offset

	// Build final query with sorting and pagination
	orderBy := buildProductsOrderByClause(filters)
	selectQuery := fmt.Sprintf(`
		SELECT 
			prod.id, prod.name, prod.description, prod.long_description, ctg.id as category_id, ctg.name as category,
			img.filename as main_img, prod.available, prod.quantity, prod.qrcode_filename, prod.slug,
			COALESCE(ARRAY_AGG(imgs.filename) FILTER (WHERE imgs.filename IS NOT NULL), '{}') as images,
			%s
		%s GROUP BY prod.id, prod.name, prod.description, prod.long_description,
		ctg.id, ctg.name, img.filename, prod.available, prod.quantity, prod.qrcode_filename, prod.slug %s
		LIMIT @limit OFFSET @offset`,
		buildSearchRankSelect(filters), baseQuery, orderBy)

	// Execute query
	rows, err := conn.Query(ctx, selectQuery, namedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan results
	products, err := scanProducts(rows, filters.SearchMode == SearchModeFullText)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &ProductFilterResult{
		Products:    products,
		Total:       total,
		Page:        filters.Page,
		Limit:       filters.Limit,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	}

	return result, nil
}

// buildQueryConditions creates WHERE conditions and named arguments
func buildQueryConditions(filters ProductFilterParams) ([]string, pgx.NamedArgs) {
	var conditions []string
	namedArgs := make(pgx.NamedArgs)

	if len(filters.IDs) > 0 {
		conditions = append(conditions, "id = ANY(@ids)")
		namedArgs["ids"] = filters.IDs
	}

	// Add search condition
	if filters.Search != "" {
		switch filters.SearchMode {
		case SearchModeFullText:
			// Full-text search with ranking
			conditions = append(conditions, "prod.search_vector @@ plainto_tsquery('spanish', @search_query)")
			namedArgs["search_query"] = filters.Search

		case SearchModeExact:
			// Exact match search
			conditions = append(conditions, "(name ILIKE @exact_search OR description ILIKE @exact_search)")
			namedArgs["exact_search"] = filters.Search

		case SearchModeFuzzy:
			// Fuzzy search (LIKE with wildcards)
			conditions = append(conditions, "(name ILIKE @fuzzy_search OR description ILIKE @fuzzy_search OR category ILIKE @fuzzy_search)")
			namedArgs["fuzzy_search"] = "%" + filters.Search + "%"
		}
	}

	// Add category filter
	if filters.Category != "" {
		conditions = append(conditions, "category = @category_id")
		namedArgs["category_id"] = filters.Category
	}

	if filters.Available > 0 {
		conditions = append(conditions, "available")
	} else if filters.Available < 0 {
		conditions = append(conditions, "NOT available")
	}

	if filters.Quantity > 0 {
		conditions = append(conditions, "quantity = @quantity")
		namedArgs["quantity"] = filters.Quantity
	}

	if filters.WithQRCode > 0 {
		conditions = append(conditions, "qr_code_filename IS NOT NULL AND qr_code_filename != ''")
	} else if filters.WithQRCode < 0 {
		conditions = append(conditions, "qr_code_filename IS NULL OR qr_code_filename = ''")
	}

	return conditions, namedArgs
}

// buildSearchRankSelect adds search ranking column when using full-text search
func buildSearchRankSelect(filters ProductFilterParams) string {
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		return "ts_rank(prod.search_vector, plainto_tsquery('spanish', @search_query)) as search_rank"
	}
	return "0 as search_rank"
}

// buildProductsOrderByClause constructs the ORDER BY clause
func buildProductsOrderByClause(filters ProductFilterParams) string {
	// If using full-text search with a query, prioritize search ranking
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		switch strings.ToLower(filters.Sort) {
		case "relevance", "":
			return "ORDER BY search_rank DESC, name ASC"
		case "name_asc", "name":
			return "ORDER BY name ASC"
		case "name_desc":
			return "ORDER BY name DESC"
		case "price_asc":
			return "ORDER BY price ASC, search_rank DESC"
		case "price_desc":
			return "ORDER BY price DESC, search_rank DESC"
		default:
			return "ORDER BY search_rank DESC, name ASC"
		}
	}

	// Regular sorting without search ranking
	switch strings.ToLower(filters.Sort) {
	case "name_asc", "name", "":
		return "ORDER BY name ASC"
	case "name_desc":
		return "ORDER BY name DESC"
	case "price_asc":
		return "ORDER BY price ASC"
	case "price_desc":
		return "ORDER BY price DESC"
	case "newest":
		return "ORDER BY id DESC"
	case "oldest":
		return "ORDER BY id ASC"
	case "category":
		return "ORDER BY category ASC, name ASC"
	default:
		return "ORDER BY name ASC"
	}
}

// scanProducts scans the query results into CatalogProd structs
func scanProducts(rows pgx.Rows, includeRank bool) ([]*Product, error) {
	var products []*Product

	for rows.Next() {
		var product Product
		var images []string
		var searchRank float32
		var mainImg sql.NullString
		var longDescription sql.NullString

		if includeRank {
			err := rows.Scan(
				&product.ID,
				&product.Name,
				&product.Description,
				&longDescription,
				&product.CategoryID,
				&product.Category,
				&mainImg,
				&product.Available,
				&product.Quantity,
				&product.QRCodeFilename,
				&product.Slug,
				&images,
				&searchRank,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan product with rank: %w", err)
			}
		} else {
			err := rows.Scan(
				&product.ID,
				&product.Name,
				&product.Description,
				&product.CategoryID,
				&product.Category,
				&mainImg,
				&product.Available,
				&product.Quantity,
				&product.QRCodeFilename,
				&product.Slug,
				&images,
				&searchRank, // Still need to scan the rank column (will be 0)
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan product: %w", err)
			}
		}

		if mainImg.Valid {
			product.MainImg = mainImg.String
		}
		if len(images) > 0 {
			product.Gallery = images
		}
		if longDescription.Valid {
			product.LongDescription = longDescription.String
		} else {
			product.LongDescription = product.Description
		}

		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return products, nil
}

func SCreateProducts(product []*Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	ctgs, err := FindAllCategories()
	if err != nil {
		return err
	}
	ctgMap := make(map[string]string)
	for _, ctg := range ctgs {
		ctgMap[ctg.Name] = ctg.ID
	}

	for _, prod := range product {
		id, err := uuid.NewV7()
		if err != nil {
			return ErrUUIDFail
		}

		prod.ID = id.String()
		setProdMainImg(prod, conn)
		prod.Gallery = []string{prod.MainImg}

		if prod.Slug == "" {
			prod.Slug = internal.Slugify(prod.Name)
		}

		args := pgx.NamedArgs{
			"id":               prod.ID,
			"name":             prod.Name,
			"slug":             prod.Slug,
			"description":      prod.Description,
			"long_description": prod.LongDescription,
			"main_img":         prod.MainImg,
			"category":         ctgMap[prod.Category],
			"available":        prod.Available,
			"quantity":         prod.Quantity,
		}
		_, err = tx.Exec(
			ctx,
			`INSERT INTO products
				(id, name, slug, description, long_description, main_img, category, available, quantity)
				VALUES (@id, @name, @slug, @description, @long_description, @main_img, @category, @available, @quantity)`,
			args,
		)
		if err != nil {
			tx.Rollback(ctx)
			return errors.Join(ErrProductInsert, err)
		}

		for _, img := range prod.Gallery {
			_, err = tx.Exec(
				ctx,
				`INSERT INTO images_products (image_id, product_id) VALUES ($1, $2)`,
				img,
				prod.ID,
			)
			if err != nil {
				tx.Rollback(ctx)
				return errors.Join(ErrGalleryInsert, err)
			}
		}
	}

	tx.Commit(ctx)
	return nil
}

func setProdMainImg(prod *Product, conn *pgxpool.Conn) {
	row := conn.QueryRow(
		context.Background(),
		`SELECT id FROM images WHERE name = $1`,
		prod.Name,
	)
	var imgID sql.NullString
	row.Scan(&imgID)
	if imgID.Valid {
		prod.MainImg = imgID.String
	}
}
