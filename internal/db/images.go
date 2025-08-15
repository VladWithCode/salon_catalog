package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	DefaultImageSelectorLimit = 20
)

var (
	ErrImageInsert                = errors.New("failed to insert image")
	ErrDeleteImageProductRelation = errors.New("failed to delete image product relation")
)

type Image struct {
	ID         string    `db:"id" json:"id"`
	Filename   string    `db:"filename" json:"filename"`
	Name       string    `db:"name" json:"name"`
	NoOptimize bool      `db:"no_optimize" json:"noOptimize"`
	Size       int       `db:"size" json:"size"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`

	// Not from schema
	Pinned bool `db:"pinned" json:"pinned"`
}

type ImageFilterParams struct {
	Name       string    `json:"name"`
	ExactDate  time.Time `json:"exact_date"`
	DateAfter  time.Time `json:"date_after"`
	DateBefore time.Time `json:"date_before"`
	SortBy     string    `json:"sort_by"`
	SortOrder  string    `json:"sort_order"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`

	Pinned []string `json:"pinned"`
}

type ImageFilterResult struct {
	Images      []*Image `json:"images"`
	Total       int      `json:"total"`
	Page        int      `json:"page"`
	Limit       int      `json:"limit"`
	TotalPages  int      `json:"total_pages"`
	HasNext     bool     `json:"has_next"`
	HasPrevious bool     `json:"has_previous"`
	HasError    bool     `json:"has_error"`
	Error       string   `json:"error"`
}

func CreateImages(imgs []*Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	for _, img := range imgs {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO images (id, filename, name, no_optimize, size)
				VALUES ($1, $2, $3, $4, $5)`,
			img.ID,
			img.Filename,
			img.Name,
			img.NoOptimize,
			img.Size,
		)

		if err != nil {
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateImage(img *Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`INSERT INTO images (id, filename, name, no_optimize, size) VALUES ($1, $2, $3, $4, $5)`,
		img.ID,
		img.Filename,
		img.Name,
		img.NoOptimize,
		img.Size,
	)
	if err != nil {
		return err
	}

	return nil
}

func LinkImagesToProduct(imgIDs []string, prodID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	for _, id := range imgIDs {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO images_products (image_id, product_id)
				VALUES ($1, $2)`,
			id,
			prodID,
		)

		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(
		ctx,
		`DELETE FROM images WHERE id NOT IN @ids::uuid[]`,
		pgx.NamedArgs{"ids": imgIDs},
	)
	if err != nil {
		return ErrDeleteImageProductRelation
	}

	return tx.Commit(ctx)
}

func UnlinkImagesFromProduct(imgIDs []string, prodID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	for _, id := range imgIDs {
		_, err = tx.Exec(
			ctx,
			`DELETE FROM images_products WHERE image_id = $1 AND product_id = $2`,
			id,
			prodID,
		)

		if err != nil {
			return ErrDeleteImageProductRelation
		}
	}

	return tx.Commit(ctx)
}

func FindImageByID(id string) (*Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var image Image
	err = conn.QueryRow(
		ctx,
		`SELECT id, filename, name, no_optimize, size, created_at FROM images WHERE id = $1`,
		id,
	).Scan(
		&image.ID,
		&image.Filename,
		&image.Name,
		&image.NoOptimize,
		&image.Size,
		&image.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &image, nil
}

func FindImageByFilename(filename string) (*Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var image Image
	err = conn.QueryRow(
		ctx,
		`SELECT id, filename, name, no_optimize, size, created_at FROM images WHERE filename = $1`,
		filename,
	).Scan(
		&image.ID,
		&image.Filename,
		&image.Name,
		&image.NoOptimize,
		&image.Size,
		&image.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &image, nil
}

func FindAllImages(ids []string) ([]*Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	baseQuery := `SELECT id, filename, name, no_optimize, size, created_at FROM images`
	if len(ids) > 0 {
		baseQuery += ` WHERE id = ANY(@ids::uuid[])`
	}

	rows, err := conn.Query(
		ctx,
		baseQuery,
		pgx.NamedArgs{"ids": ids},
	)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var images []*Image
	for rows.Next() {
		var image Image
		err = rows.Scan(
			&image.ID,
			&image.Filename,
			&image.Name,
			&image.NoOptimize,
			&image.Size,
			&image.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		images = append(images, &image)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return images, nil
}

func UpdateImage(image *Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`UPDATE images SET filename = $1, no_optimize = $2, size = $3, created_at = $4 WHERE id = $5`,
		image.Filename,
		image.NoOptimize,
		image.Size,
		image.CreatedAt,
		image.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

func DeleteImages(ids []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var deletedFilenames []string
	rows, err := conn.Query(
		ctx,
		`DELETE FROM images WHERE id = ANY(@ids::uuid[]) RETURNING filename`,
		pgx.NamedArgs{"ids": ids},
	)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var filename string
		err = rows.Scan(&filename)
		if err != nil {
			return nil, err
		}
		deletedFilenames = append(deletedFilenames, filename)
	}

	return deletedFilenames, nil
}

func DeleteImage(id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return "", err
	}
	defer conn.Release()

	var filename string
	row := conn.QueryRow(
		ctx,
		`DELETE FROM images WHERE id = $1 RETURNING filename`,
		id,
	)
	err = row.Scan(&filename)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func FilterImages(filters ImageFilterParams) (*ImageFilterResult, error) {
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
	if filters.SortBy == "" {
		filters.SortBy = "created_at"
	}
	if filters.SortOrder == "" {
		filters.SortOrder = "DESC"
	}
	if filters.Pinned == nil {
		filters.Pinned = []string{}
	}

	// Build query conditions and named arguments
	conditions, namedArgs := buildImageQueryConditions(filters)

	// Base query
	baseQuery := `FROM images`
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
	orderBy := buildImageOrderByClause(filters)
	selectQuery := fmt.Sprintf(`
		SELECT 
			id, filename, name, no_optimize, size, created_at,
			%s %s
		%s %s 
		LIMIT @limit OFFSET @offset`,
		buildPinnedSelectClause(filters),
		buildImageSearchRankSelect(filters),
		baseQuery, orderBy)

	// Execute query
	rows, err := conn.Query(ctx, selectQuery, namedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan results
	var images []*Image
	hasSearchRank := filters.Name != ""

	for rows.Next() {
		var image Image
		var searchRank float32

		if hasSearchRank {
			err = rows.Scan(
				&image.ID,
				&image.Filename,
				&image.Name,
				&image.NoOptimize,
				&image.Size,
				&image.CreatedAt,
				&image.Pinned,
				&searchRank,
			)
		} else {
			err = rows.Scan(
				&image.ID,
				&image.Filename,
				&image.Name,
				&image.NoOptimize,
				&image.Size,
				&image.CreatedAt,
				&image.Pinned,
				&searchRank, // Still need to scan the rank column (will be 0)
			)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}

		images = append(images, &image)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Build result
	result := &ImageFilterResult{
		Images:      images,
		Total:       total,
		Page:        filters.Page,
		Limit:       filters.Limit,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	}

	return result, nil
}

// buildImageQueryConditions creates WHERE conditions and named arguments
func buildImageQueryConditions(filters ImageFilterParams) ([]string, pgx.NamedArgs) {
	var conditions []string
	namedArgs := make(pgx.NamedArgs)

	// Full-text search by name
	if filters.Name != "" {
		conditions = append(conditions, "search_vector @@ plainto_tsquery('spanish', @search_query)")
		namedArgs["search_query"] = filters.Name
	}

	// Exact date filter
	if !filters.ExactDate.IsZero() {
		conditions = append(conditions, "DATE(created_at) = DATE(@exact_date)")
		namedArgs["exact_date"] = filters.ExactDate
	}

	// Date range filters (only if exact date is not set)
	if filters.ExactDate.IsZero() {
		if !filters.DateAfter.IsZero() {
			conditions = append(conditions, "created_at >= @date_after")
			namedArgs["date_after"] = filters.DateAfter
		}

		if !filters.DateBefore.IsZero() {
			conditions = append(conditions, "created_at <= @date_before")
			namedArgs["date_before"] = filters.DateBefore
		}
	}

	if len(filters.Pinned) > 0 {
		namedArgs["pinned"] = filters.Pinned
	}

	return conditions, namedArgs
}

func buildPinnedSelectClause(filters ImageFilterParams) string {
	if len(filters.Pinned) == 0 {
		return "False as pinned,"
	}

	return `
		CASE
			WHEN id = ANY(@pinned::uuid[]) 
			THEN True
			ELSE False
		END as pinned,
	`
}

// buildImageSearchRankSelect adds search ranking column when using full-text search
func buildImageSearchRankSelect(filters ImageFilterParams) string {
	if filters.Name != "" {
		return "ts_rank(search_vector, plainto_tsquery('spanish', @search_query)) as search_rank"
	}
	return "0 as search_rank"
}

// buildImageOrderByClause constructs the ORDER BY clause
func buildImageOrderByClause(filters ImageFilterParams) string {
	orderClause := "ORDER BY"
	if len(filters.Pinned) > 0 {
		orderClause += `
			CASE
				WHEN id = ANY(@pinned::uuid[]) 
				THEN array_position(@pinned::uuid[], id)
				ELSE @limit + 2
			END,
		`
	}
	// If using full-text search, prioritize search ranking
	if filters.Name != "" {
		orderClause += fmt.Sprintf(" search_rank DESC, %s %s",
			sanitizeSortBy(filters.SortBy), sanitizeSortOrder(filters.SortOrder))
		return orderClause
	}

	orderClause += fmt.Sprintf(" %s %s",
		sanitizeSortBy(filters.SortBy), sanitizeSortOrder(filters.SortOrder))
	return orderClause
}

// sanitizeSortBy ensures only valid column names are used for sorting
func sanitizeSortBy(sortBy string) string {
	switch strings.ToLower(sortBy) {
	case "name":
		return "name"
	case "filename":
		return "filename"
	case "size":
		return "size"
	case "created_at", "createdAt", "date":
		return "created_at"
	default:
		return "created_at"
	}
}

// sanitizeSortOrder ensures only valid sort orders are used
func sanitizeSortOrder(sortOrder string) string {
	switch strings.ToUpper(sortOrder) {
	case "ASC", "asc":
		return "ASC"
	case "DESC", "desc":
		return "DESC"
	default:
		return "DESC"
	}
}
