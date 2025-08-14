package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type EventKindDetails struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type QuoteRequestType string

const (
	QuoteRequestTypeReservation QuoteRequestType = "reservación"
	QuoteRequestTypeBudget      QuoteRequestType = "cotización"
	QuoteRequestTypeContact     QuoteRequestType = "contacto"
)

type QuoteStatus string

const (
	QuoteStatusPending   QuoteStatus = "pendiente"
	QuoteStatusProcessed QuoteStatus = "procesada"
	QuoteStatusProgress  QuoteStatus = "en_progreso"
	QuoteStatusCancelled QuoteStatus = "cancelada"
)

type Quote struct {
	ID            string           `json:"id"`
	CustomerName  string           `json:"customer_name"`
	CustomerPhone string           `json:"customer_phone"`
	CustomerEmail string           `json:"customer_email"`
	TimeStart     *time.Time       `json:"time_start"`
	TimeEnd       *time.Time       `json:"time_end"`
	RequestType   QuoteRequestType `json:"request_type"`
	Status        QuoteStatus      `json:"status"`
	Comments      sql.NullString   `json:"comments"`
	CartID        sql.NullString   `json:"cart_id"`
	Cart          *Cart            `json:"cart"`
	EventKindID   sql.NullString   `json:"event_kind_id"`
	EventKindName sql.NullString   `json:"event_kind_name"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type QuoteFilterParams struct {
	CustomerName   string `json:"customer_name"`
	Phone          string `json:"phone"`
	CreatedFrom    string `json:"created_from"`
	CreatedTo      string `json:"created_to"`
	EventStartFrom string `json:"event_start_from"`
	EventStartTo   string `json:"event_start_to"`
	Status         string `json:"status"`
	RequestType    string `json:"request_type"`
	Comments       string `json:"comments"`
	Sort           string `json:"sort"`
	Page           int    `json:"page"`
	Limit          int    `json:"limit"`
}

type QuoteFilterResult struct {
	Quotes      []*Quote `json:"quotes"`
	Total       int      `json:"total"`
	Page        int      `json:"page"`
	Limit       int      `json:"limit"`
	TotalPages  int      `json:"total_pages"`
	HasNext     bool     `json:"has_next"`
	HasPrevious bool     `json:"has_previous"`
	HasError    bool     `json:"has_error"`
	Error       string   `json:"error"`
}

func CreateQuote(quote *Quote) error {
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

	var timeStart, timeEnd sql.NullTime
	if quote.TimeStart != nil {
		timeStart = sql.NullTime{Time: *quote.TimeStart, Valid: true}
	}
	if quote.TimeEnd != nil {
		timeEnd = sql.NullTime{Time: *quote.TimeEnd, Valid: true}
	}

	_, err = conn.Exec(
		ctx,
		`INSERT INTO quotes (
			id, customer_name, customer_phone, time_start, time_end, status, comments, cart_id, request_type, event_kind_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id.String(),
		quote.CustomerName,
		quote.CustomerPhone,
		timeStart,
		timeEnd,
		quote.Status,
		quote.Comments,
		quote.CartID,
		quote.RequestType,
		quote.EventKindID,
	)
	if err != nil {
		return err
	}

	return nil
}

func FindQuoteByID(id string) (*Quote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var quote Quote
	var timeStart, timeEnd sql.NullTime
	var jsonItems []byte

	err = conn.QueryRow(
		ctx,
		`SELECT
			id, customer_name, customer_phone, time_start, time_end, cart_id, 
			request_type, status, comments, event_kind_id, event_kind_name, 
			created_at, updated_at, cart_items
		FROM quote_details
		WHERE id = $1
		`,
		id,
	).Scan(
		&quote.ID,
		&quote.CustomerName,
		&quote.CustomerPhone,
		&timeStart,
		&timeEnd,
		&quote.CartID,
		&quote.RequestType,
		&quote.Status,
		&quote.Comments,
		&quote.EventKindID,
		&quote.EventKindName,
		&quote.CreatedAt,
		&quote.UpdatedAt,
		&jsonItems,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if timeStart.Valid {
		quote.TimeStart = &timeStart.Time
	}
	if timeEnd.Valid {
		quote.TimeEnd = &timeEnd.Time
	}

	if len(jsonItems) > 0 {
		quote.Cart = &Cart{}
		err = json.Unmarshal(jsonItems, &quote.Cart.Items)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal cart items: %w", err)
		}
	}

	return &quote, nil
}

func FindAllQuotes() ([]*Quote, error) {
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
			q.id, q.customer_name, q.customer_phone, q.time_start, q.time_end,
			q.status, q.comments, q.cart_id, q.request_type, q.event_kind_id, 
			ek.name AS event_kind_name, q.created_at, q.updated_at
		FROM quotes q
			LEFT JOIN event_kinds ek ON q.event_kind_id = ek.id
		ORDER BY q.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes, err := scanQuotes(rows)
	if err != nil {
		return nil, err
	}

	return quotes, nil
}

func UpdateQuote(quote *Quote) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	var timeStart, timeEnd sql.NullTime
	if quote.TimeStart != nil {
		timeStart = sql.NullTime{Time: *quote.TimeStart, Valid: true}
	}
	if quote.TimeEnd != nil {
		timeEnd = sql.NullTime{Time: *quote.TimeEnd, Valid: true}
	}

	args := pgx.NamedArgs{
		"time_start":    timeStart,
		"time_end":      timeEnd,
		"cart_id":       quote.CartID,
		"request_type":  quote.RequestType,
		"event_kind_id": quote.EventKindID,
		"status":        quote.Status,
		"comments":      quote.Comments,
		"id":            quote.ID,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE quotes SET
			time_start = @time_start, time_end = @time_end, cart_id = @cart_id, 
			request_type = @request_type, event_kind_id = @event_kind_id, status = @status, 
			comments = @comments
		WHERE id = @id`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteQuote(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM quotes WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteQuotes(ids []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM quotes WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return err
	}

	return nil
}

func UpdateQuoteStatus(ids []string, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`UPDATE quotes SET status = $1 WHERE id = ANY($2)`,
		status,
		ids,
	)
	if err != nil {
		return err
	}

	return nil
}

func FilterQuotes(filters QuoteFilterParams) (*QuoteFilterResult, error) {
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

	// Build query conditions and named arguments
	conditions, namedArgs := buildQuoteQueryConditions(filters)

	// Base query with explicit column selection
	baseQuery := `
		FROM quotes q
		LEFT JOIN event_kinds ek ON q.event_kind_id = ek.id
		`
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := "SELECT COUNT(q.id) " + baseQuery
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
	orderBy := buildQuoteOrderByClause(filters)
	selectQuery := fmt.Sprintf(`
		SELECT 
			q.id, q.customer_name, q.customer_phone, q.time_start, q.time_end,
			q.request_type, q.status, q.comments, q.cart_id, q.event_kind_id,
			ek.name as event_kind_name, q.created_at, q.updated_at
		%s %s
		LIMIT @limit OFFSET @offset`,
		baseQuery, orderBy)

	// Execute query
	rows, err := conn.Query(ctx, selectQuery, namedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan results
	quotes, err := scanQuotes(rows)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &QuoteFilterResult{
		Quotes:      quotes,
		Total:       total,
		Page:        filters.Page,
		Limit:       filters.Limit,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	}

	return result, nil
}

// buildQuoteQueryConditions creates WHERE conditions and named arguments
func buildQuoteQueryConditions(filters QuoteFilterParams) ([]string, pgx.NamedArgs) {
	var conditions []string
	namedArgs := make(pgx.NamedArgs)

	// Customer name filter
	if filters.CustomerName != "" {
		conditions = append(conditions, "q.customer_name ILIKE @customer_name")
		namedArgs["customer_name"] = "%" + filters.CustomerName + "%"
	}

	// Phone filter
	if filters.Phone != "" {
		conditions = append(conditions, "q.customer_phone ILIKE @phone")
		namedArgs["phone"] = "%" + filters.Phone + "%"
	}

	// Created date range filter
	if filters.CreatedFrom != "" {
		conditions = append(conditions, "q.created_at >= @created_from")
		namedArgs["created_from"] = filters.CreatedFrom + " 00:00:00"
	}
	if filters.CreatedTo != "" {
		conditions = append(conditions, "q.created_at <= @created_to")
		namedArgs["created_to"] = filters.CreatedTo + " 23:59:59"
	}

	// Event start date range filter
	if filters.EventStartFrom != "" {
		conditions = append(conditions, "q.time_start >= @event_start_from")
		namedArgs["event_start_from"] = filters.EventStartFrom + " 00:00:00"
	}
	if filters.EventStartTo != "" {
		conditions = append(conditions, "q.time_start <= @event_start_to")
		namedArgs["event_start_to"] = filters.EventStartTo + " 23:59:59"
	}

	// Status filter
	if filters.Status != "" {
		conditions = append(conditions, "q.status = @status")
		namedArgs["status"] = filters.Status
	}

	// Request type filter
	if filters.RequestType != "" {
		conditions = append(conditions, "q.request_type ILIKE @request_type")
		namedArgs["request_type"] = "%" + filters.RequestType + "%"
	}

	// Comments filter
	if filters.Comments != "" {
		conditions = append(conditions, "q.comments ILIKE @comments")
		namedArgs["comments"] = "%" + filters.Comments + "%"
	}

	return conditions, namedArgs
}

// buildQuoteOrderByClause constructs the ORDER BY clause
func buildQuoteOrderByClause(filters QuoteFilterParams) string {
	switch strings.ToLower(filters.Sort) {
	case "created_asc":
		return "ORDER BY q.created_at ASC"
	case "created_desc", "":
		return "ORDER BY q.created_at DESC"
	case "event_start_asc":
		return "ORDER BY q.time_start ASC NULLS LAST"
	case "event_start_desc":
		return "ORDER BY q.time_start DESC NULLS LAST"
	case "event_kind_asc":
		return "ORDER BY ek.name ASC NULLS LAST"
	case "event_kind_desc":
		return "ORDER BY ek.name DESC NULLS LAST"
	case "request_type_asc":
		return "ORDER BY q.request_type ASC"
	case "request_type_desc":
		return "ORDER BY q.request_type DESC"
	case "status_asc":
		return "ORDER BY q.status ASC"
	case "status_desc":
		return "ORDER BY q.status DESC"
	default:
		return "ORDER BY q.created_at DESC"
	}
}

// scanQuotes scans the query results into Quote structs
func scanQuotes(rows pgx.Rows) ([]*Quote, error) {
	var quotes []*Quote

	for rows.Next() {
		var quote Quote
		var timeStart sql.NullTime
		var timeEnd sql.NullTime

		err := rows.Scan(
			&quote.ID,
			&quote.CustomerName,
			&quote.CustomerPhone,
			&timeStart,
			&timeEnd,
			&quote.RequestType,
			&quote.Status,
			&quote.Comments,
			&quote.CartID,
			&quote.EventKindID,
			&quote.EventKindName,
			&quote.CreatedAt,
			&quote.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quote: %w", err)
		}

		// Handle nullable fields
		if timeStart.Valid {
			quote.TimeStart = &timeStart.Time
		}
		if timeEnd.Valid {
			quote.TimeEnd = &timeEnd.Time
		}

		quotes = append(quotes, &quote)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return quotes, nil
}

func FindQuotesByCustomerName(customerName string) ([]*Quote, error) {
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
			q.id, q.customer_name, q.customer_phone, q.time_start, q.time_end, 
			q.cart_id, q.request_type, q.event_kind_id, q.status, q.comments,
			ek.name AS event_kind_name, q.created_at, q.updated_at
		FROM quotes q
			LEFT JOIN event_kinds ek ON q.event_kind_id = ek.id
		WHERE q.customer_name = $1`,
		customerName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes, err := scanQuotes(rows)
	if err != nil {
		return nil, err
	}

	return quotes, nil
}
