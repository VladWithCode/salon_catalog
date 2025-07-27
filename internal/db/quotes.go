package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type EventKindDetails struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Quote struct {
	ID               string            `json:"id"`
	CustomerName     string            `json:"customer_name"`
	CustomerPhone    string            `json:"customer_phone"`
	TimeStart        sql.NullTime      `json:"time_start"`
	TimeEnd          sql.NullTime      `json:"time_end"`
	RequestType      string            `json:"request_type"` // The type refers to the type of request
	Status           string            `json:"status"`
	Comments         sql.NullString    `json:"comments"`
	CartID           sql.NullString    `json:"cart_id"`
	EventKindID      string            `json:"service_kind_id"` // The kind refers to the type of event
	EventKind        string            `json:"service_kind"`
	EventKindDetails *EventKindDetails `json:"event_kind_details"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
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
	QuoteStatusResponded QuoteStatus = "respondido"
	QuoteStatusApproved  QuoteStatus = "aprobado"
	QuoteStatusRejected  QuoteStatus = "rechazado"
	QuoteStatusCancelled QuoteStatus = "cancelado"
)

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

	args := pgx.NamedArgs{
		"id":            id.String(),
		"name":          quote.CustomerName,
		"phone":         quote.CustomerPhone,
		"time_start":    quote.TimeStart,
		"time_end":      quote.TimeEnd,
		"cart_id":       quote.CartID,
		"request_type":  quote.RequestType,
		"event_kind_id": quote.EventKindID,
		"status":        quote.Status,
		"comments":      quote.Comments,
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO quotes (
			id, customer_name, customer_phone, time_start, time_end, status, comments, cart_id, request_type, event_kind_id
		) VALUES (@id, @name, @phone, @time_start, @time_end, @status, @comments, @cart_id, @request_type, @event_kind_id)`,
		id.String(),
		args,
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
	var eventKindDetails EventKindDetails
	err = conn.QueryRow(
		ctx,
		`SELECT 
			q.id, q.customer_name, q.customer_phone, q.time_start, q.time_end, q.status, 
			q.comments, q.cart_id, q.request_type, q.event_kind_id, 
			ek.name AS event_kind, ekd.name AS event_kind_details
		FROM quotes q
			LEFT JOIN event_kinds ek ON q.event_kind_id = ek.id
			LEFT JOIN event_kinds_details ekd ON q.event_kind_id = ekd.id
		WHERE q.id = $1`,
		id,
	).Scan(
		&quote.ID,
		&quote.CustomerName,
		&quote.CustomerPhone,
		&quote.TimeStart,
		&quote.TimeEnd,
		&quote.Status,
		&quote.Comments,
		&quote.CartID,
		&quote.RequestType,
		&quote.EventKindID,
		&eventKindDetails.Name,
		&eventKindDetails.Description,
	)
	if err != nil {
		return nil, err
	}

	quote.EventKind = eventKindDetails.Name
	quote.EventKindDetails = &eventKindDetails

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
			ek.name AS event_kind
		FROM quotes q
			LEFT JOIN event_kinds ek ON q.event_kind_id = ek.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []*Quote
	for rows.Next() {
		var quote Quote
		var eventKindDetails EventKindDetails
		err = rows.Scan(
			&quote.ID,
			&quote.CustomerName,
			&quote.CustomerPhone,
			&quote.TimeStart,
			&quote.TimeEnd,
			&quote.Status,
			&quote.Comments,
			&quote.CartID,
			&quote.RequestType,
			&quote.EventKindID,
			&eventKindDetails.Name,
			&eventKindDetails.Description,
		)
		if err != nil {
			return nil, err
		}

		quote.EventKind = eventKindDetails.Name
		quote.EventKindDetails = &eventKindDetails

		quotes = append(quotes, &quote)
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

	args := pgx.NamedArgs{
		"id":            quote.ID,
		"name":          quote.CustomerName,
		"phone":         quote.CustomerPhone,
		"time_start":    quote.TimeStart,
		"time_end":      quote.TimeEnd,
		"cart_id":       quote.CartID,
		"request_type":  quote.RequestType,
		"event_kind_id": quote.EventKindID,
		"status":        quote.Status,
		"comments":      quote.Comments,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE quotes SET
			customer_name = @name, customer_phone = @phone, time_start = @time_start, time_end = @time_end, cart_id = @cart_id, request_type = @request_type, event_kind_id = @event_kind_id, status = @status, comments = @comments
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
			q.id, q.customer_name, q.customer_phone, q.time_start, q.time_end, q.cart_id, q.request_type, q.event_kind_id, 
			ek.name AS event_kind
		FROM quotes q
			LEFT JOIN event_kinds ek ON q.event_kind_id = ek.id
		WHERE q.customer_name = $1`,
		customerName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []*Quote
	for rows.Next() {
		var quote Quote
		var eventKindDetails EventKindDetails
		err = rows.Scan(
			&quote.ID,
			&quote.CustomerName,
			&quote.CustomerPhone,
			&quote.TimeStart,
			&quote.TimeEnd,
			&quote.CartID,
			&quote.RequestType,
			&quote.EventKindID,
			&eventKindDetails.Name,
			&eventKindDetails.Description,
		)
		if err != nil {
			return nil, err
		}

		quote.EventKind = eventKindDetails.Name
		quote.EventKindDetails = &eventKindDetails

		quotes = append(quotes, &quote)
	}
	return quotes, nil
}
