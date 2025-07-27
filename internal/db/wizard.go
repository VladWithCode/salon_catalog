package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Wizard struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	EventKindID string        `json:"event_kind_id"`
	EventKind   string        `json:"event_kind"`
	Steps       []*WizardStep `json:"steps"`
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

func CreateWizard(wizard *Wizard, steps []*WizardStep) error {
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
	args := pgx.NamedArgs{
		"id":            id.String(),
		"name":          wizard.Name,
		"event_kind_id": wizard.EventKindID,
	}
	_, err = tx.Exec(
		ctx,
		`INSERT INTO wizards (
			id, name, event_kind_id
		) VALUES (@id, @name, @event_kind_id)`,
		args,
	)
	if err != nil {
		return err
	}

	for _, step := range steps {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO wizard_steps (
				id, wizard_id, name, description, required, multi_select, min_selected, max_selected, category_ids
			) VALUES (@id, @wizard_id, @name, @description, @required, @multi_select, @min_selected, @max_selected, @category_ids)`,
			pgx.NamedArgs{
				"id":           step.ID,
				"wizard_id":    step.WizardID,
				"name":         step.Name,
				"description":  step.Description,
				"required":     step.Required,
				"multi_select": step.MultiSelect,
				"min_selected": step.MinSelected,
				"max_selected": step.MaxSelected,
				"category_ids": step.CategoryIDs,
			},
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
			ek.name AS event_kind,
		FROM wizards w
			LEFT JOIN event_kinds ek ON w.event_kind_id = ek.id
		WHERE w.id = $1`,
		id,
	).Scan(
		&wizard.ID,
		&wizard.Name,
		&wizard.EventKindID,
		&eventKindDetails.Name,
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

func DeleteWizard(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM wizards WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}
