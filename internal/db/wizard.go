package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Wizard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	EventKindID string `json:"event_kind_id"`
	EventKind   string `json:"event_kind"`
}

type WizardStep struct {
	ID          string `json:"id"`
	WizardID    string `json:"wizard_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	MultiSelect bool   `json:"multi_select"`
	MinSelected int    `json:"min_selected"`
	MaxSelected int    `json:"max_selected"`
	CategoryIDs string `json:"category_ids"`
}

func CreateWizard(wizard *Wizard) error {
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
		"name":          wizard.Name,
		"event_kind_id": wizard.EventKindID,
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO wizards (
			id, name, event_kind_id
		) VALUES (@id, @name, @event_kind_id)`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func FindWizardByID(id string) (*Wizard, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var wizard Wizard
	var eventKindDetails EventKindDetails
	err = conn.QueryRow(
		ctx,
		`SELECT 
			w.id, w.name, w.event_kind_id, 
			ek.name AS event_kind, ekd.name AS event_kind_details
		FROM wizards w
			LEFT JOIN event_kinds ek ON w.event_kind_id = ek.id
			LEFT JOIN event_kinds_details ekd ON w.event_kind_id = ekd.id
		WHERE w.id = $1`,
		id,
	).Scan(
		&wizard.ID,
		&wizard.Name,
		&wizard.EventKindID,
		&eventKindDetails.Name,
		&eventKindDetails.Description,
	)
	if err != nil {
		return nil, err
	}

	wizard.EventKind = eventKindDetails.Name

	return &wizard, nil
}

func FindAllWizards() ([]*Wizard, error) {
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
			w.id, w.name, w.event_kind_id, 
			ek.name AS event_kind
		FROM wizards w
			LEFT JOIN event_kinds ek ON w.event_kind_id = ek.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wizards []*Wizard
	for rows.Next() {
		var wizard Wizard
		var eventKindDetails EventKindDetails
		err = rows.Scan(
			&wizard.ID,
			&wizard.Name,
			&wizard.EventKindID,
			&eventKindDetails.Name,
			&eventKindDetails.Description,
		)
		if err != nil {
			return nil, err
		}

		wizard.EventKind = eventKindDetails.Name
		wizards = append(wizards, &wizard)
	}
	return wizards, nil
}

func UpdateWizard(wizard *Wizard) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	args := pgx.NamedArgs{
		"id":            wizard.ID,
		"name":          wizard.Name,
		"event_kind_id": wizard.EventKindID,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE wizards SET
			name = @name, event_kind_id = @event_kind_id
		WHERE id = @id`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}
