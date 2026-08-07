package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vladwithcode/salon_catalog/internal"
)

type Category struct {
	ID              string `db:"id" json:"id"`
	Name            string `db:"name" json:"name"`
	Slug            string `db:"slug" json:"slug"`
	Description     string `db:"description" json:"description"`
	LongDescription string `db:"long_description" json:"longDescription"`
	HeaderImg       string `db:"header_img" json:"headerImg"`
	HeaderImgID     string `db:"header_img_id" json:"headerImgId"`
	DisplayImg      string `db:"display_img" json:"displayImg"`
	DisplayImgID    string `db:"display_img_id" json:"displayImgId"`
	ProductCount    int    `db:"product_count" json:"productCount"`
	QRCodeFilename  string `db:"qrcode_filename" json:"qrcodeFilename"`
}

type CategoryFilterParams struct {
	Search     string     `json:"search"`
	SearchMode SearchMode `json:"search_mode"`
	Sort       string     `json:"sort"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
}

type CategoryFilterResult struct {
	Categories  []*Category `json:"categories"`
	Total       int         `json:"total"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
	TotalPages  int         `json:"total_pages"`
	HasNext     bool        `json:"has_next"`
	HasPrevious bool        `json:"has_previous"`
	HasError    bool        `json:"has_error"`
	Error       string      `json:"error"`
}

func CreateCategory(category *Category) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	id, err := uuid.NewV7()
	if err != nil {
		return ErrUUIDFail
	}

	headerImg := sql.NullString{
		String: category.HeaderImg,
		Valid:  category.HeaderImg != "",
	}
	if category.HeaderImgID != "" {
		headerImg.String = category.HeaderImgID
		headerImg.Valid = true
	}
	displayImg := sql.NullString{
		String: category.DisplayImg,
		Valid:  category.DisplayImg != "",
	}
	if category.DisplayImgID != "" {
		displayImg.String = category.DisplayImgID
		displayImg.Valid = true
	}

	if category.Slug == "" {
		category.Slug = internal.Slugify(category.Name)
	}

	args := pgx.NamedArgs{
		"id":              id.String(),
		"name":            category.Name,
		"slug":            category.Slug,
		"description":     category.Description,
		"header_img":      headerImg,
		"display_img":     displayImg,
		"qrcode_filename": category.QRCodeFilename,
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO categories (id, name, slug, description, header_img, display_img, qrcode_filename) VALUES (@id, @name, @slug, @description, @header_img, @display_img, @qrcode_filename)`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func FindCategoryBySlug(slug string) (*Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var (
		category     Category
		headerImg    sql.NullString
		headerImgID  sql.NullString
		displayImg   sql.NullString
		displayImgID sql.NullString
	)

	err = conn.QueryRow(
		ctx,
		`SELECT 
			ctg.id, ctg.name, ctg.slug, ctg.description,
			header.filename AS header_img,
			header.id AS header_img_id,
			display.filename AS display_img,
			display.id AS display_img_id,
			ctg.qrcode_filename
		FROM categories ctg
			LEFT JOIN images header ON header.id = ctg.header_img
			LEFT JOIN images display ON display.id = ctg.display_img
		WHERE slug = $1`,
		slug,
	).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&headerImg,
		&headerImgID,
		&displayImg,
		&displayImgID,
		&category.QRCodeFilename,
	)
	if err != nil {
		return nil, err
	}

	if headerImg.Valid {
		category.HeaderImg = headerImg.String
	}
	if displayImg.Valid {
		category.DisplayImg = displayImg.String
	}
	if headerImgID.Valid {
		category.HeaderImgID = headerImgID.String
	}
	if displayImgID.Valid {
		category.DisplayImgID = displayImgID.String
	}

	return &category, nil
}

func FindCategoryByID(id string) (*Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var (
		category     Category
		headerImg    sql.NullString
		headerImgID  sql.NullString
		displayImg   sql.NullString
		displayImgID sql.NullString
	)

	err = conn.QueryRow(
		ctx,
		`SELECT 
			ctg.id, ctg.name, ctg.slug, ctg.description,
			header.filename AS header_img,
			header.id AS header_img_id,
			display.filename AS display_img,
			display.id AS display_img_id,
			qrcode_filename
		FROM categories ctg
			LEFT JOIN images header ON header.id = ctg.header_img
			LEFT JOIN images display ON display.id = ctg.display_img
		WHERE ctg.id = $1`,
		id,
	).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&headerImg,
		&headerImgID,
		&displayImg,
		&displayImgID,
		&category.QRCodeFilename,
	)
	if err != nil {
		return nil, err
	}

	if headerImg.Valid {
		category.HeaderImg = headerImg.String
	}
	if headerImgID.Valid {
		category.HeaderImgID = headerImgID.String
	}
	if displayImg.Valid {
		category.DisplayImg = displayImg.String
	}
	if displayImgID.Valid {
		category.DisplayImgID = displayImgID.String
	}

	return &category, nil
}

func FindAllCategories() ([]*Category, error) {
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
			ctg.id, ctg.name, ctg.slug, ctg.description, 
			header.filename AS header_img,
			header.id AS header_img_id,
			display.filename AS display_img,
			display.id AS display_img_id,
			ctg.qrcode_filename
		FROM categories ctg
			LEFT JOIN images header ON header.id = ctg.header_img
			LEFT JOIN images display ON display.id = ctg.display_img
		ORDER BY ctg.name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		var (
			category     Category
			headerImg    sql.NullString
			headerImgID  sql.NullString
			displayImg   sql.NullString
			displayImgID sql.NullString
		)

		err = rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
			&headerImg,
			&headerImgID,
			&displayImg,
			&displayImgID,
			&category.QRCodeFilename,
		)
		if err != nil {
			return nil, err
		}

		if headerImg.Valid {
			category.HeaderImg = headerImg.String
		}
		if headerImgID.Valid {
			category.HeaderImgID = headerImgID.String
		}
		if displayImg.Valid {
			category.DisplayImg = displayImg.String
		}
		if displayImgID.Valid {
			category.DisplayImgID = displayImgID.String
		}

		categories = append(categories, &category)
	}

	return categories, nil
}

func UpdateCategory(category *Category) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	headerImg := sql.NullString{
		String: category.HeaderImg,
		Valid:  category.HeaderImg != "",
	}
	if _, err := uuid.Parse(category.HeaderImg); err != nil {
		headerImg.String = category.HeaderImgID
		headerImg.Valid = category.HeaderImgID != ""
	}

	displayImg := sql.NullString{
		String: category.DisplayImg,
		Valid:  category.DisplayImg != "",
	}
	if _, err := uuid.Parse(category.DisplayImg); err != nil {
		displayImg.String = category.DisplayImgID
		displayImg.Valid = category.DisplayImgID != ""
	}

	if category.Slug == "" {
		category.Slug = internal.Slugify(category.Name)
	}
	args := pgx.NamedArgs{
		"id":              category.ID,
		"name":            category.Name,
		"slug":            category.Slug,
		"description":     category.Description,
		"header_img":      headerImg,
		"display_img":     displayImg,
		"qrcode_filename": category.QRCodeFilename,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE categories SET
			name = @name, slug = @slug, description = @description, header_img = @header_img, display_img = @display_img, qrcode_filename = @qrcode_filename
		WHERE id = @id`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteCategory(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM categories WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}

func FilterCategories(filters CategoryFilterParams) (*CategoryFilterResult, error) {
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
	conditions, namedArgs := buildCategoryQueryConditions(filters)

	// Base query with explicit column selection
	baseQuery := `
		FROM categories ctg
		LEFT JOIN products p ON ctg.id = p.category
		LEFT JOIN images header ON header.id = ctg.header_img
		LEFT JOIN images display ON display.id = ctg.display_img
		`
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := "SELECT COUNT(DISTINCT ctg.id) " + baseQuery
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
	orderBy := buildCategoryOrderByClause(filters)
	selectQuery := fmt.Sprintf(`
		SELECT 
			ctg.id, ctg.name, ctg.slug, ctg.description, ctg.long_description,
			header.filename as header_img,
			header.id as header_img_id,
			display.filename as display_img,
			display.id as display_img_id,
			COUNT(p.id) as product_count,
			ctg.qrcode_filename,
			%s
		%s GROUP BY ctg.id, ctg.name, ctg.slug, ctg.description, ctg.long_description,
		header.filename, header.id, display.filename, display.id, ctg.qrcode_filename %s
		LIMIT @limit OFFSET @offset`,
		buildCategorySearchRankSelect(filters), baseQuery, orderBy)

	// Execute query
	rows, err := conn.Query(ctx, selectQuery, namedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan results
	categories, err := scanCategories(rows, filters.SearchMode == SearchModeFullText)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &CategoryFilterResult{
		Categories:  categories,
		Total:       total,
		Page:        filters.Page,
		Limit:       filters.Limit,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	}

	return result, nil
}

// buildCategoryQueryConditions creates WHERE conditions and named arguments
func buildCategoryQueryConditions(filters CategoryFilterParams) ([]string, pgx.NamedArgs) {
	var conditions []string
	namedArgs := make(pgx.NamedArgs)

	// Add search condition
	if filters.Search != "" {
		switch filters.SearchMode {
		case SearchModeFullText:
			// Full-text search with ranking
			conditions = append(conditions, "ctg.search_vector @@ plainto_tsquery('spanish', @search_query)")
			namedArgs["search_query"] = filters.Search

		case SearchModeExact:
			// Exact match search
			conditions = append(conditions, "(ctg.name ILIKE @exact_search OR ctg.description ILIKE @exact_search)")
			namedArgs["exact_search"] = filters.Search

		case SearchModeFuzzy:
			// Fuzzy search (LIKE with wildcards)
			conditions = append(conditions, "(ctg.name ILIKE @fuzzy_search OR ctg.description ILIKE @fuzzy_search)")
			namedArgs["fuzzy_search"] = "%" + filters.Search + "%"
		}
	}

	return conditions, namedArgs
}

// buildCategorySearchRankSelect adds search ranking column when using full-text search
func buildCategorySearchRankSelect(filters CategoryFilterParams) string {
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		return "ts_rank(ctg.search_vector, plainto_tsquery('spanish', @search_query)) as search_rank"
	}
	return "0 as search_rank"
}

// buildCategoryOrderByClause constructs the ORDER BY clause
func buildCategoryOrderByClause(filters CategoryFilterParams) string {
	// If using full-text search with a query, prioritize search ranking
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		switch strings.ToLower(filters.Sort) {
		case "relevance", "":
			return "ORDER BY search_rank DESC, ctg.name ASC"
		case "name_asc", "name":
			return "ORDER BY ctg.name ASC"
		case "name_desc":
			return "ORDER BY ctg.name DESC"
		case "product_count_asc":
			return "ORDER BY product_count ASC, search_rank DESC"
		case "product_count_desc":
			return "ORDER BY product_count DESC, search_rank DESC"
		default:
			return "ORDER BY search_rank DESC, ctg.name ASC"
		}
	}

	// Regular sorting without search ranking
	switch strings.ToLower(filters.Sort) {
	case "name_asc", "name", "":
		return "ORDER BY ctg.name ASC"
	case "name_desc":
		return "ORDER BY ctg.name DESC"
	case "product_count_asc":
		return "ORDER BY product_count ASC"
	case "product_count_desc":
		return "ORDER BY product_count DESC"
	default:
		return "ORDER BY ctg.name ASC"
	}
}

// scanCategories scans the query results into Category structs
func scanCategories(rows pgx.Rows, includeRank bool) ([]*Category, error) {
	var categories []*Category

	for rows.Next() {
		var category Category
		var searchRank float32
		var headerImg sql.NullString
		var headerImgID sql.NullString
		var displayImg sql.NullString
		var displayImgID sql.NullString
		var longDescription sql.NullString

		if includeRank {
			err := rows.Scan(
				&category.ID,
				&category.Name,
				&category.Slug,
				&category.Description,
				&longDescription,
				&headerImg,
				&headerImgID,
				&displayImg,
				&displayImgID,
				&category.ProductCount,
				&category.QRCodeFilename,
				&searchRank,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan category with rank: %w", err)
			}
		} else {
			err := rows.Scan(
				&category.ID,
				&category.Name,
				&category.Slug,
				&category.Description,
				&longDescription,
				&headerImg,
				&headerImgID,
				&displayImg,
				&displayImgID,
				&category.ProductCount,
				&category.QRCodeFilename,
				&searchRank, // Still need to scan the rank column (will be 0)
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan category: %w", err)
			}
		}

		if headerImg.Valid {
			category.HeaderImg = headerImg.String
		}
		if headerImgID.Valid {
			category.HeaderImgID = headerImgID.String
		}
		if displayImg.Valid {
			category.DisplayImg = displayImg.String
		}
		if displayImgID.Valid {
			category.DisplayImgID = displayImgID.String
		}
		if longDescription.Valid {
			category.LongDescription = longDescription.String
		}

		categories = append(categories, &category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return categories, nil
}

func UpdateCategoryHeaderImg(categoryId, imageId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	headerImg := sql.NullString{
		String: imageId,
		Valid:  imageId != "",
	}

	_, err = conn.Exec(
		ctx,
		`UPDATE categories SET header_img = $1 WHERE id = $2`,
		headerImg,
		categoryId,
	)
	if err != nil {
		return err
	}

	return nil
}

func UpdateCategoryDisplayImg(categoryId, imageId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	displayImg := sql.NullString{
		String: imageId,
		Valid:  imageId != "",
	}

	_, err = conn.Exec(
		ctx,
		`UPDATE categories SET display_img = $1 WHERE id = $2`,
		displayImg,
		categoryId,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteCategoryHeaderImg(categoryId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`UPDATE categories SET header_img = NULL WHERE id = $1`,
		categoryId,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteCategoryDisplayImg(categoryId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`UPDATE categories SET display_img = NULL WHERE id = $1`,
		categoryId,
	)
	if err != nil {
		return err
	}

	return nil
}
