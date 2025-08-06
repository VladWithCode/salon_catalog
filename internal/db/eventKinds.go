package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type EventKind struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EventKindFilterParams struct {
	Search string
	Page   int
	Limit  int
}

type EventKindFilterResult struct {
	EventKinds []*EventKind `json:"event_kinds"`
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	HasError   bool         `json:"has_error"`
	Error      string       `json:"error"`
}

func CreateEventKind(eventKind *EventKind) error {
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
		"id":          id.String(),
		"name":        eventKind.Name,
		"description": eventKind.Description,
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO event_kinds (id, name, description) VALUES (@id, @name, @description)`,
		args,
	)
	if err != nil {
		return err
	}

	eventKind.ID = id.String()
	return nil
}

func FindAllEventKinds() ([]*EventKind, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(
		ctx,
		`SELECT id, name, description, created_at, updated_at FROM event_kinds ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventKinds []*EventKind
	for rows.Next() {
		var eventKind EventKind
		err = rows.Scan(
			&eventKind.ID,
			&eventKind.Name,
			&eventKind.Description,
			&eventKind.CreatedAt,
			&eventKind.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		eventKinds = append(eventKinds, &eventKind)
	}

	return eventKinds, nil
}

func FindEventKindByID(id string) (*EventKind, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var eventKind EventKind
	err = conn.QueryRow(
		ctx,
		`SELECT id, name, description, created_at, updated_at FROM event_kinds WHERE id = $1`,
		id,
	).Scan(
		&eventKind.ID,
		&eventKind.Name,
		&eventKind.Description,
		&eventKind.CreatedAt,
		&eventKind.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &eventKind, nil
}

func UpdateEventKind(eventKind *EventKind) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	args := pgx.NamedArgs{
		"id":          eventKind.ID,
		"name":        eventKind.Name,
		"description": eventKind.Description,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE event_kinds SET name = @name, description = @description, updated_at = NOW() WHERE id = @id`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteEventKind(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM event_kinds WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}

func FilterEventKinds(params EventKindFilterParams) (*EventKindFilterResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	result := &EventKindFilterResult{
		Page:  params.Page,
		Limit: params.Limit,
	}

	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := (params.Page - 1) * params.Limit

	// Build query with search filter
	baseQuery := `FROM event_kinds`
	whereClause := ""
	args := pgx.NamedArgs{
		"limit":  params.Limit,
		"offset": offset,
	}

	if params.Search != "" {
		whereClause = ` WHERE name ILIKE @search OR description ILIKE @search`
		args["search"] = "%" + params.Search + "%"
	}

	// Get total count
	countQuery := `SELECT COUNT(*) ` + baseQuery + whereClause
	err = conn.QueryRow(ctx, countQuery, args).Scan(&result.Total)
	if err != nil {
		return nil, err
	}

	// Get filtered results
	selectQuery := `SELECT id, name, description, created_at, updated_at ` + baseQuery + whereClause + ` ORDER BY name LIMIT @limit OFFSET @offset`
	rows, err := conn.Query(ctx, selectQuery, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventKinds []*EventKind
	for rows.Next() {
		var eventKind EventKind
		err = rows.Scan(
			&eventKind.ID,
			&eventKind.Name,
			&eventKind.Description,
			&eventKind.CreatedAt,
			&eventKind.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		eventKinds = append(eventKinds, &eventKind)
	}

	result.EventKinds = eventKinds
	return result, nil
}
