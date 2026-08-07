package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Wizard struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	EventKindID string        `json:"event_kind_id"`
	EventKind   string        `json:"event_kind"`
	IsGeneral   bool          `json:"is_general"`
	Steps       []*WizardStep `json:"steps"`
	Enabled     bool          `json:"enabled"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type WizardStep struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Required    bool      `json:"required"`
	MultiSelect bool      `json:"multi_select"`
	MinSelected int       `json:"min_selected"`
	MaxSelected int       `json:"max_selected"`
	CategoryIDs []string  `json:"category_ids"`
	Categories  []string  `json:"categories"`
	StepOrder   int       `json:"step_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WizardStepFilterParams struct {
	Search     string     `json:"search"`
	SearchMode SearchMode `json:"search_mode"`
	Categories []string   `json:"categories"`
	Sort       string     `json:"sort"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
}

type WizardStepFilterResult struct {
	WizardSteps []*WizardStep `json:"wizard_steps"`
	Total       int           `json:"total"`
	Page        int           `json:"page"`
	Limit       int           `json:"limit"`
	TotalPages  int           `json:"total_pages"`
	HasNext     bool          `json:"has_next"`
	HasPrevious bool          `json:"has_previous"`
	HasError    bool          `json:"has_error"`
	Error       string        `json:"error"`
}

func CreateWizard(ctx context.Context, wizard *Wizard) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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

	if wizard.ID == "" {
		id := uuid.Must(uuid.NewV7()).String()
		wizard.ID = id
	}
	args := pgx.NamedArgs{
		"id":          wizard.ID,
		"name":        wizard.Name,
		"description": wizard.Description,
		"is_general":  wizard.IsGeneral,
		"enabled":     wizard.Enabled,
	}

	if wizard.EventKindID != "" {
		args["event_kind_id"] = wizard.EventKindID
	} else {
		args["event_kind_id"] = nil
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO wizards (id, name, description, event_kind_id, is_general, enabled)
		VALUES (@id, @name, @description, @event_kind_id, @is_general, @enabled)`,
		args,
	)

	if err != nil {
		return err
	}

	batch := pgx.Batch{}
	for _, step := range wizard.Steps {
		args := pgx.NamedArgs{
			"wizard_id":      wizard.ID,
			"wizard_step_id": step.ID,
			"required":       step.Required,
			"step_order":     step.StepOrder,
			"multi_select":   step.MultiSelect,
			"min_selected":   step.MinSelected,
			"max_selected":   step.MaxSelected,
		}
		batch.Queue(
			`INSERT INTO wizard_steps_wizards 
				(wizard_id, wizard_step_id, required, step_order, multi_select, min_selected, max_selected)
			VALUES (@wizard_id, @wizard_step_id, @required, @step_order, @multi_select, @min_selected, @max_selected)`,
			args,
		)
	}
	batchResults := tx.SendBatch(ctx, &batch)
	defer batchResults.Close()

	for i, l := 0, batch.Len(); i < l; i++ {
		_, err := batchResults.Exec()
		if err != nil {
			return err
		}
	}

	batchResults.Close()
	return tx.Commit(ctx)
}

func FindWizard(ctx context.Context, id string) (*Wizard, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	wizard := Wizard{}
	var eventKindID sql.NullString
	err = conn.QueryRow(
		ctx,
		`SELECT 
			id, name, description, event_kind_id, is_general, enabled, created_at, updated_at
		FROM wizards WHERE id = $1`,
		id,
	).Scan(
		&wizard.ID,
		&wizard.Name,
		&wizard.Description,
		&eventKindID,
		&wizard.IsGeneral,
		&wizard.Enabled,
		&wizard.CreatedAt,
		&wizard.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if eventKindID.Valid {
		wizard.EventKindID = eventKindID.String
	}

	return &wizard, nil
}

func UpdateWizard(ctx context.Context, wizard *Wizard) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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

	// Update wizard basic info
	_, err = tx.Exec(
		ctx,
		`UPDATE wizards 
		 SET name = @name, description = @description, event_kind_id = NULLIF(@event_kind_id, ''), 
		     is_general = @is_general, enabled = @enabled
		 WHERE id = @id`,
		pgx.NamedArgs{
			"id":            wizard.ID,
			"name":          wizard.Name,
			"description":   wizard.Description,
			"event_kind_id": wizard.EventKindID,
			"is_general":    wizard.IsGeneral,
			"enabled":       wizard.Enabled,
		},
	)
	if err != nil {
		return err
	}

	// Update wizard steps if provided
	if len(wizard.Steps) > 0 {
		// Delete existing wizard step associations
		_, err = tx.Exec(ctx, "DELETE FROM wizard_steps_wizards WHERE wizard_id = $1", wizard.ID)
		if err != nil {
			return err
		}

		// Insert new wizard step associations
		batch := pgx.Batch{}
		for _, step := range wizard.Steps {
			args := pgx.NamedArgs{
				"wizard_id":      wizard.ID,
				"wizard_step_id": step.ID,
				"required":       step.Required,
				"step_order":     step.StepOrder,
				"multi_select":   step.MultiSelect,
				"min_selected":   step.MinSelected,
				"max_selected":   step.MaxSelected,
			}
			batch.Queue(
				`INSERT INTO wizard_steps_wizards 
					(wizard_id, wizard_step_id, required, step_order, multi_select, min_selected, max_selected)
				VALUES (@wizard_id, @wizard_step_id, @required, @step_order, @multi_select, @min_selected, @max_selected)`,
				args,
			)
		}
		batchResults := tx.SendBatch(ctx, &batch)
		defer batchResults.Close()

		for i, l := 0, batch.Len(); i < l; i++ {
			_, err := batchResults.Exec()
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func DeleteWizard(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "DELETE FROM wizards WHERE id = $1", id)
	return err
}

func GetWizardWithSteps(ctx context.Context, id string) (*Wizard, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Get wizard basic info
	wizard := Wizard{}
	var eventKind sql.NullString
	var eventKindID sql.NullString
	err = conn.QueryRow(ctx, `
		SELECT w.id, w.name, w.description, w.event_kind_id, w.is_general, w.enabled, w.created_at, w.updated_at, ek.name as event_kind
		FROM wizards w
		LEFT JOIN event_kinds ek ON w.event_kind_id = ek.id
		WHERE w.id = $1`, id).Scan(
		&wizard.ID,
		&wizard.Name,
		&wizard.Description,
		&eventKindID,
		&wizard.IsGeneral,
		&wizard.Enabled,
		&wizard.CreatedAt,
		&wizard.UpdatedAt,
		&eventKind,
	)
	if err != nil {
		return nil, err
	}

	if eventKind.Valid {
		wizard.EventKind = eventKind.String
	}
	if eventKindID.Valid {
		wizard.EventKindID = eventKindID.String
	}

	// Get wizard steps with custom parameters
	query := `
		SELECT ws.id, ws.name, ws.description, 
		       COALESCE(wsw.step_order, ws.step_order) as step_order,
		       COALESCE(wsw.required, ws.required) as required, 
		       COALESCE(wsw.multi_select, ws.multi_select) as multi_select,
		       COALESCE(wsw.min_selected, ws.min_selected) as min_selected, 
		       COALESCE(wsw.max_selected, ws.max_selected) as max_selected,
		       ws.created_at, ws.updated_at,
		       array_remove(array_agg(DISTINCT c.id), NULL) as category_ids,
		       array_remove(array_agg(DISTINCT c.name), NULL) as categories
		FROM wizard_steps_wizards wsw
		JOIN wizard_steps ws ON wsw.wizard_step_id = ws.id
		LEFT JOIN wizard_step_categories wsc ON ws.id = wsc.wizard_step_id
		LEFT JOIN categories c ON wsc.category_id = c.id
		WHERE wsw.wizard_id = $1
		GROUP BY ws.id, ws.name, ws.description, ws.step_order, ws.required, ws.multi_select,
		         ws.min_selected, ws.max_selected, wsw.step_order, wsw.required, 
		         wsw.multi_select, wsw.min_selected, wsw.max_selected, ws.created_at, ws.updated_at
		ORDER BY COALESCE(wsw.step_order, ws.step_order) ASC
	`

	rows, err := conn.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*WizardStep
	for rows.Next() {
		var step WizardStep
		var categoryIDs []string
		var categories []string

		err := rows.Scan(
			&step.ID,
			&step.Name,
			&step.Description,
			&step.StepOrder,
			&step.Required,
			&step.MultiSelect,
			&step.MinSelected,
			&step.MaxSelected,
			&step.CreatedAt,
			&step.UpdatedAt,
			&categoryIDs,
			&categories,
		)
		if err != nil {
			return nil, err
		}

		step.CategoryIDs = categoryIDs
		step.Categories = categories
		steps = append(steps, &step)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	wizard.Steps = steps
	return &wizard, nil
}

func CreateWizardStep(ctx context.Context, wizardStep *WizardStep) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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

	if wizardStep.ID == "" {
		id := uuid.Must(uuid.NewV7()).String()
		wizardStep.ID = id
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO wizard_steps (
			id, name, description, multi_select, min_selected, max_selected, step_order
		) VALUES (@id, @name, @description, @multi_select, @min_selected, @max_selected, @step_order)`,
		pgx.NamedArgs{
			"id":           wizardStep.ID,
			"name":         wizardStep.Name,
			"description":  wizardStep.Description,
			"multi_select": wizardStep.MultiSelect,
			"min_selected": wizardStep.MinSelected,
			"max_selected": wizardStep.MaxSelected,
			"step_order":   wizardStep.StepOrder,
		},
	)
	if err != nil {
		return err
	}

	batch := pgx.Batch{}
	for _, stepCtg := range wizardStep.CategoryIDs {
		batch.Queue(
			`INSERT INTO wizard_step_categories (wizard_step_id, category_id) VALUES ($1, $2)`,
			wizardStep.ID,
			stepCtg,
		)
	}
	batchResults := tx.SendBatch(ctx, &batch)
	defer batchResults.Close()

	for i, l := 0, batch.Len(); i < l; i++ {
		_, err := batchResults.Exec()
		if err != nil {
			return err
		}
	}
	batchResults.Close()

	return tx.Commit(ctx)
}

type WizardFilterParams struct {
	Search     string     `json:"search"`
	SearchMode SearchMode `json:"search_mode"`
	EventKind  string     `json:"event_kind"`
	Sort       string     `json:"sort"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	Enabled    int        `json:"enabled"` // 0 = all, 1 = enabled, -1 = disabled
}

type WizardFilterResult struct {
	Wizards     []*Wizard `json:"wizards"`
	Total       int       `json:"total"`
	Page        int       `json:"page"`
	Limit       int       `json:"limit"`
	TotalPages  int       `json:"total_pages"`
	HasNext     bool      `json:"has_next"`
	HasPrevious bool      `json:"has_previous"`
	HasError    bool      `json:"has_error"`
	Error       string    `json:"error"`
}

func FilterWizards(filters WizardFilterParams) (*WizardFilterResult, error) {
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
	conditions, namedArgs := buildWizardQueryConditions(filters)

	if filters.Enabled != 0 {
		conditions = append(conditions, "w.enabled = @enabled")
		switch filters.Enabled {
		case 1:
			namedArgs["enabled"] = true
		case -1:
			namedArgs["enabled"] = false
		}
	}

	// Base query with explicit column selection
	baseQuery := `
		FROM wizards w
		LEFT JOIN event_kinds ek ON w.event_kind_id = ek.id
		`

	// When enabled filter is set to true, add join to filter out wizards with no steps
	if filters.Enabled == 1 {
		baseQuery = `
			FROM wizards w
			LEFT JOIN event_kinds ek ON w.event_kind_id = ek.id
			INNER JOIN wizard_steps_wizards wsw ON w.id = wsw.wizard_id
			`
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count - use DISTINCT when joining with wizard_steps_wizards to avoid duplicates
	var countQuery string
	if filters.Enabled == 1 {
		countQuery = "SELECT COUNT(DISTINCT w.id) " + baseQuery
	} else {
		countQuery = "SELECT COUNT(*) " + baseQuery
	}
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
	orderBy := buildWizardOrderByClause(filters)
	var selectQuery string
	if filters.Enabled == 1 {
		// Use DISTINCT when joining with wizard_steps_wizards to avoid duplicates
		selectQuery = fmt.Sprintf(`
			SELECT DISTINCT
				w.id, w.name, w.description, w.event_kind_id, w.is_general, w.enabled,
				ek.name as event_kind,
				%s
			%s %s
			LIMIT @limit OFFSET @offset`,
			buildWizardSearchRankSelect(filters), baseQuery, orderBy)
	} else {
		selectQuery = fmt.Sprintf(`
			SELECT 
				w.id, w.name, w.description, w.event_kind_id, w.is_general, w.enabled,
				ek.name as event_kind,
				%s
			%s %s
			LIMIT @limit OFFSET @offset`,
			buildWizardSearchRankSelect(filters), baseQuery, orderBy)
	}

	// Execute query
	rows, err := conn.Query(ctx, selectQuery, namedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan results
	wizards, err := scanWizards(rows, filters.SearchMode == SearchModeFullText)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &WizardFilterResult{
		Wizards:     wizards,
		Total:       total,
		Page:        filters.Page,
		Limit:       filters.Limit,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	}

	return result, nil
}

// buildWizardQueryConditions creates WHERE conditions and named arguments
func buildWizardQueryConditions(filters WizardFilterParams) ([]string, pgx.NamedArgs) {
	var conditions []string
	namedArgs := make(pgx.NamedArgs)

	// Add search condition
	if filters.Search != "" {
		switch filters.SearchMode {
		case SearchModeFullText:
			// Full-text search with ranking (assuming search_vector exists)
			conditions = append(conditions, "w.search_vector @@ plainto_tsquery('spanish', @search_query)")
			namedArgs["search_query"] = filters.Search

		case SearchModeExact:
			// Exact match search
			conditions = append(conditions, "(w.name ILIKE @exact_search)")
			namedArgs["exact_search"] = filters.Search

		case SearchModeFuzzy:
			// Fuzzy search (LIKE with wildcards)
			conditions = append(conditions, "(w.name ILIKE @fuzzy_search OR ek.name ILIKE @fuzzy_search)")
			namedArgs["fuzzy_search"] = "%" + filters.Search + "%"
		}
	}

	// Add event kind filter
	if filters.EventKind != "" {
		conditions = append(conditions, "w.event_kind_id = @event_kind_id")
		namedArgs["event_kind_id"] = filters.EventKind
	}

	return conditions, namedArgs
}

// buildWizardSearchRankSelect adds search ranking column when using full-text search
func buildWizardSearchRankSelect(filters WizardFilterParams) string {
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		return "ts_rank(w.search_vector, plainto_tsquery('spanish', @search_query)) as search_rank"
	}
	return "0 as search_rank"
}

// buildWizardOrderByClause constructs the ORDER BY clause
func buildWizardOrderByClause(filters WizardFilterParams) string {
	// If using full-text search with a query, prioritize search ranking
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		switch strings.ToLower(filters.Sort) {
		case "relevance", "":
			return "ORDER BY search_rank DESC, w.name ASC"
		case "name_asc", "name":
			return "ORDER BY w.name ASC"
		case "name_desc":
			return "ORDER BY w.name DESC"
		case "eventkind_asc":
			return "ORDER BY ek.name ASC, search_rank DESC"
		case "eventkind_desc":
			return "ORDER BY ek.name DESC, search_rank DESC"
		default:
			return "ORDER BY search_rank DESC, w.name ASC"
		}
	}

	// Regular sorting without search ranking
	switch strings.ToLower(filters.Sort) {
	case "name_asc", "name", "":
		return "ORDER BY w.name ASC"
	case "name_desc":
		return "ORDER BY w.name DESC"
	case "eventkind_asc":
		return "ORDER BY ek.name ASC"
	case "eventkind_desc":
		return "ORDER BY ek.name DESC"
	case "newest":
		return "ORDER BY w.id DESC"
	case "oldest":
		return "ORDER BY w.id ASC"
	default:
		return "ORDER BY w.name ASC"
	}
}

// scanWizards scans the query results into Wizard structs
func scanWizards(rows pgx.Rows, includeRank bool) ([]*Wizard, error) {
	var wizards []*Wizard

	for rows.Next() {
		var wizard Wizard
		var searchRank float32
		var eventKind sql.NullString
		var eventKindID sql.NullString

		if includeRank {
			err := rows.Scan(
				&wizard.ID,
				&wizard.Name,
				&wizard.Description,
				&eventKindID,
				&wizard.IsGeneral,
				&wizard.Enabled,
				&eventKind,
				&searchRank,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan wizard with rank: %w", err)
			}
		} else {
			err := rows.Scan(
				&wizard.ID,
				&wizard.Name,
				&wizard.Description,
				&eventKindID,
				&wizard.IsGeneral,
				&wizard.Enabled,
				&eventKind,
				&searchRank, // Still need to scan the rank column (will be 0)
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan wizard: %w", err)
			}
		}

		if eventKind.Valid {
			wizard.EventKind = eventKind.String
		}
		if eventKindID.Valid {
			wizard.EventKindID = eventKindID.String
		}

		wizards = append(wizards, &wizard)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return wizards, nil
}

func FilterWizardSteps(filters WizardStepFilterParams) (*WizardStepFilterResult, error) {
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
	conditions, namedArgs := buildWizardStepQueryConditions(filters)

	// Base query with joins to get categories
	baseQuery := `
		FROM wizard_steps ws
		LEFT JOIN wizard_step_categories wsc ON ws.id = wsc.wizard_step_id
		LEFT JOIN categories c ON wsc.category_id = c.id
		`
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count (distinct wizard steps) - need to handle HAVING clause
	var total int
	if len(filters.Categories) > 0 {
		// When filtering by categories, use subquery to count after HAVING clause
		namedArgs["category_ids"] = filters.Categories
		countQuery := fmt.Sprintf(`
			SELECT COUNT(*) FROM (
				SELECT DISTINCT ws.id
				%s
				GROUP BY ws.id, ws.name, ws.description, ws.required, ws.multi_select,
					ws.min_selected, ws.max_selected, ws.step_order, ws.created_at, ws.updated_at
				HAVING array_remove(array_agg(DISTINCT c.id), NULL) && @category_ids
			) AS filtered_steps
		`, baseQuery)
		err = conn.QueryRow(ctx, countQuery, namedArgs).Scan(&total)
	} else {
		// When not filtering by categories, use simple count
		countQuery := "SELECT COUNT(DISTINCT ws.id) " + baseQuery
		err = conn.QueryRow(ctx, countQuery, namedArgs).Scan(&total)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination
	offset := (filters.Page - 1) * filters.Limit
	totalPages := int(math.Ceil(float64(total) / float64(filters.Limit)))

	// Add pagination to named args
	namedArgs["limit"] = filters.Limit
	namedArgs["offset"] = offset

	// Build HAVING clause for category filtering
	havingClause := ""
	if len(filters.Categories) > 0 {
		havingClause = " HAVING array_remove(array_agg(DISTINCT c.id), NULL) && @category_ids"
		// Add category_ids to namedArgs for the HAVING clause
		namedArgs["category_ids"] = filters.Categories
	}

	// Build final query with sorting and pagination
	orderBy := buildWizardStepOrderByClause(filters)
	selectQuery := fmt.Sprintf(`
		SELECT DISTINCT
			ws.id, ws.name, ws.description, ws.required, ws.multi_select,
			ws.min_selected, ws.max_selected, ws.step_order, ws.created_at, ws.updated_at,
			array_remove(array_agg(DISTINCT c.id), NULL) as category_ids,
			array_remove(array_agg(DISTINCT c.name), NULL) as categories
		%s 
		GROUP BY ws.id, ws.name, ws.description, ws.required, ws.multi_select,
			ws.min_selected, ws.max_selected, ws.step_order, ws.created_at, ws.updated_at
		%s
		%s
		LIMIT @limit OFFSET @offset`, baseQuery, havingClause, orderBy)

	// Execute query
	rows, err := conn.Query(ctx, selectQuery, namedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan results
	wizardSteps, err := scanWizardSteps(rows)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &WizardStepFilterResult{
		WizardSteps: wizardSteps,
		Total:       total,
		Page:        filters.Page,
		Limit:       filters.Limit,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	}

	return result, nil
}

func buildWizardStepQueryConditions(filters WizardStepFilterParams) ([]string, pgx.NamedArgs) {
	var conditions []string
	namedArgs := make(pgx.NamedArgs)

	// Add search condition (WHERE clause only - category filtering moved to HAVING clause)
	if filters.Search != "" {
		switch filters.SearchMode {
		case SearchModeFullText:
			conditions = append(conditions, "(ws.name ILIKE @search_query OR ws.description ILIKE @search_query)")
			namedArgs["search_query"] = "%" + filters.Search + "%"

		case SearchModeExact:
			conditions = append(conditions, "(ws.name ILIKE @exact_search OR ws.description ILIKE @exact_search)")
			namedArgs["exact_search"] = filters.Search

		case SearchModeFuzzy:
			conditions = append(conditions, "(ws.name ILIKE @fuzzy_search OR ws.description ILIKE @fuzzy_search)")
			namedArgs["fuzzy_search"] = "%" + filters.Search + "%"
		}
	}

	return conditions, namedArgs
}

func buildWizardStepOrderByClause(filters WizardStepFilterParams) string {
	switch strings.ToLower(filters.Sort) {
	case "name_asc", "name", "":
		return "ORDER BY ws.name ASC"
	case "name_desc":
		return "ORDER BY ws.name DESC"
	case "newest":
		return "ORDER BY ws.created_at DESC"
	case "oldest":
		return "ORDER BY ws.created_at ASC"
	case "step_order":
		return "ORDER BY ws.step_order ASC, ws.name ASC"
	default:
		return "ORDER BY ws.name ASC"
	}
}

func scanWizardSteps(rows pgx.Rows) ([]*WizardStep, error) {
	var wizardSteps []*WizardStep

	for rows.Next() {
		var step WizardStep
		var categoryIDs []string
		var categories []string

		err := rows.Scan(
			&step.ID,
			&step.Name,
			&step.Description,
			&step.Required,
			&step.MultiSelect,
			&step.MinSelected,
			&step.MaxSelected,
			&step.StepOrder,
			&step.CreatedAt,
			&step.UpdatedAt,
			&categoryIDs,
			&categories,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wizard step: %w", err)
		}

		step.CategoryIDs = categoryIDs
		step.Categories = categories

		wizardSteps = append(wizardSteps, &step)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return wizardSteps, nil
}

func UpdateWizardStep(ctx context.Context, step *WizardStep) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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

	// Update wizard step
	_, err = tx.Exec(
		ctx,
		`UPDATE wizard_steps 
		 SET name = @name, description = @description, required = @required,
		     multi_select = @multi_select, min_selected = @min_selected, 
		     max_selected = @max_selected, step_order = @step_order,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE id = @id`,
		pgx.NamedArgs{
			"id":           step.ID,
			"name":         step.Name,
			"description":  step.Description,
			"required":     step.Required,
			"multi_select": step.MultiSelect,
			"min_selected": step.MinSelected,
			"max_selected": step.MaxSelected,
			"step_order":   step.StepOrder,
		},
	)
	if err != nil {
		return err
	}

	// Delete existing category associations
	_, err = tx.Exec(ctx, "DELETE FROM wizard_step_categories WHERE wizard_step_id = $1", step.ID)
	if err != nil {
		return err
	}

	// Insert new category associations
	if len(step.CategoryIDs) > 0 {
		batch := pgx.Batch{}
		batch.Queue(
			`DELETE FROM wizard_step_categories WHERE wizard_step_id = $1`,
			step.ID,
		)

		for _, categoryID := range step.CategoryIDs {
			batch.Queue(
				`INSERT INTO wizard_step_categories (wizard_step_id, category_id) VALUES ($1, $2)`,
				step.ID,
				categoryID,
			)
		}

		batchResults := tx.SendBatch(ctx, &batch)
		defer batchResults.Close()

		for i, l := 0, batch.Len(); i < l; i++ {
			_, err := batchResults.Exec()
			if err != nil {
				return err
			}
		}

		batchResults.Close()
	}

	return tx.Commit(ctx)
}

func DeleteWizardStep(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "DELETE FROM wizard_steps WHERE id = $1", id)
	return err
}

func FindWizardStep(ctx context.Context, id string) (*WizardStep, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		SELECT ws.id, ws.name, ws.description, ws.required, ws.multi_select,
		       ws.min_selected, ws.max_selected, ws.step_order, ws.created_at, ws.updated_at,
		       array_remove(array_agg(DISTINCT c.id), NULL) as category_ids,
		       array_remove(array_agg(DISTINCT c.name), NULL) as categories
		FROM wizard_steps ws
		LEFT JOIN wizard_step_categories wsc ON ws.id = wsc.wizard_step_id
		LEFT JOIN categories c ON wsc.category_id = c.id
		WHERE ws.id = $1
		GROUP BY ws.id, ws.name, ws.description, ws.required, ws.multi_select,
		         ws.min_selected, ws.max_selected, ws.step_order, ws.created_at, ws.updated_at
	`

	var step WizardStep
	var categoryIDs []string
	var categories []string

	err = conn.QueryRow(ctx, query, id).Scan(
		&step.ID,
		&step.Name,
		&step.Description,
		&step.Required,
		&step.MultiSelect,
		&step.MinSelected,
		&step.MaxSelected,
		&step.StepOrder,
		&step.CreatedAt,
		&step.UpdatedAt,
		&categoryIDs,
		&categories,
	)
	if err != nil {
		return nil, err
	}

	step.CategoryIDs = categoryIDs
	step.Categories = categories

	return &step, nil
}

func GetAllWizardSteps(ctx context.Context) ([]*WizardStep, error) {
	filters := WizardStepFilterParams{
		Page:  1,
		Limit: 1000, // Get all steps
	}
	result, err := FilterWizardSteps(filters)
	if err != nil {
		return nil, err
	}
	return result.WizardSteps, nil
}

func AttachStepToWizard(ctx context.Context, wizardID, stepID string, stepParams *WizardStep) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	args := pgx.NamedArgs{
		"wizard_id":      wizardID,
		"wizard_step_id": stepID,
		"required":       stepParams.Required,
		"step_order":     stepParams.StepOrder,
		"multi_select":   stepParams.MultiSelect,
		"min_selected":   stepParams.MinSelected,
		"max_selected":   stepParams.MaxSelected,
	}

	_, err = conn.Exec(
		ctx,
		`INSERT INTO wizard_steps_wizards 
			(wizard_id, wizard_step_id, required, step_order, multi_select, min_selected, max_selected)
		VALUES (@wizard_id, @wizard_step_id, @required, @step_order, @multi_select, @min_selected, @max_selected)
		ON CONFLICT (wizard_id, wizard_step_id) 
		DO UPDATE SET
			required = EXCLUDED.required,
			step_order = EXCLUDED.step_order,
			multi_select = EXCLUDED.multi_select,
			min_selected = EXCLUDED.min_selected,
			max_selected = EXCLUDED.max_selected`,
		args,
	)

	return err
}

func DetachStepFromWizard(ctx context.Context, wizardID, stepID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		"DELETE FROM wizard_steps_wizards WHERE wizard_id = $1 AND wizard_step_id = $2",
		wizardID,
		stepID,
	)

	return err
}

func UpdateWizardStepParams(ctx context.Context, wizardID, stepID string, stepParams *WizardStep) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	args := pgx.NamedArgs{
		"wizard_id":      wizardID,
		"wizard_step_id": stepID,
		"required":       stepParams.Required,
		"step_order":     stepParams.StepOrder,
		"multi_select":   stepParams.MultiSelect,
		"min_selected":   stepParams.MinSelected,
		"max_selected":   stepParams.MaxSelected,
	}

	_, err = conn.Exec(
		ctx,
		`UPDATE wizard_steps_wizards 
		 SET required = @required, step_order = @step_order, multi_select = @multi_select,
		     min_selected = @min_selected, max_selected = @max_selected
		 WHERE wizard_id = @wizard_id AND wizard_step_id = @wizard_step_id`,
		args,
	)

	return err
}

func ValidateStepOrderUnique(ctx context.Context, wizardID, stepID string, stepOrder int) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `
		SELECT COUNT(*)
		FROM wizard_steps_wizards wsw
		JOIN wizard_steps ws ON wsw.wizard_step_id = ws.id
		WHERE wsw.wizard_id = $1 
		AND wsw.wizard_step_id != $2
		AND COALESCE(wsw.step_order, ws.step_order) = $3
	`

	var count int
	err = conn.QueryRow(ctx, query, wizardID, stepID, stepOrder).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("step order %d is already taken by another step", stepOrder)
	}

	return nil
}

func GetNextAvailableStepPosition(ctx context.Context, wizardID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return 1, err
	}
	defer conn.Release()

	query := `
		SELECT COALESCE(wsw.step_order, ws.step_order) as step_order
		FROM wizard_steps_wizards wsw
		JOIN wizard_steps ws ON wsw.wizard_step_id = ws.id
		WHERE wsw.wizard_id = $1
		ORDER BY COALESCE(wsw.step_order, ws.step_order) ASC
	`

	rows, err := conn.Query(ctx, query, wizardID)
	if err != nil {
		return 1, err
	}
	defer rows.Close()

	var usedPositions []int
	for rows.Next() {
		var position int
		err := rows.Scan(&position)
		if err != nil {
			return 1, err
		}
		if position > 0 {
			usedPositions = append(usedPositions, position)
		}
	}

	if err := rows.Err(); err != nil {
		return 1, err
	}

	// If no steps exist, return 1
	if len(usedPositions) == 0 {
		return 1, nil
	}

	// Find the first gap in the sequence
	for i := 1; i <= len(usedPositions)+1; i++ {
		found := slices.Contains(usedPositions, i)
		if !found {
			return i, nil
		}
	}

	// This should never happen, but return next position after last
	return len(usedPositions) + 1, nil
}

func GetWizardStepWithDefaults(ctx context.Context, wizardID, stepID string) (*WizardStep, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		SELECT ws.id, ws.name, ws.description, 
		       COALESCE(wsw.step_order, ws.step_order) as step_order,
		       COALESCE(wsw.required, ws.required) as required, 
		       COALESCE(wsw.multi_select, ws.multi_select) as multi_select,
		       COALESCE(wsw.min_selected, ws.min_selected) as min_selected, 
		       COALESCE(wsw.max_selected, ws.max_selected) as max_selected,
		       ws.created_at, ws.updated_at,
		       array_remove(array_agg(DISTINCT c.id), NULL) as category_ids,
		       array_remove(array_agg(DISTINCT c.name), NULL) as categories
		FROM wizard_steps ws
		LEFT JOIN wizard_steps_wizards wsw ON (wsw.wizard_step_id = ws.id AND wsw.wizard_id = $1)
		LEFT JOIN wizard_step_categories wsc ON ws.id = wsc.wizard_step_id
		LEFT JOIN categories c ON wsc.category_id = c.id
		WHERE ws.id = $2
		GROUP BY ws.id, ws.name, ws.description, ws.step_order, ws.required, ws.multi_select,
		         ws.min_selected, ws.max_selected, wsw.step_order, wsw.required, 
		         wsw.multi_select, wsw.min_selected, wsw.max_selected, ws.created_at, ws.updated_at
	`

	var step WizardStep
	var categoryIDs []string
	var categories []string

	err = conn.QueryRow(ctx, query, wizardID, stepID).Scan(
		&step.ID,
		&step.Name,
		&step.Description,
		&step.StepOrder,
		&step.Required,
		&step.MultiSelect,
		&step.MinSelected,
		&step.MaxSelected,
		&step.CreatedAt,
		&step.UpdatedAt,
		&categoryIDs,
		&categories,
	)
	if err != nil {
		return nil, err
	}

	step.CategoryIDs = categoryIDs
	step.Categories = categories

	return &step, nil
}

func GetWizardSteps(ctx context.Context, wizardID string) ([]*WizardStep, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		SELECT ws.id, ws.name, ws.description, 
		       COALESCE(wsw.step_order, ws.step_order) as step_order,
		       COALESCE(wsw.required, ws.required) as required, 
		       COALESCE(wsw.multi_select, ws.multi_select) as multi_select,
		       COALESCE(wsw.min_selected, ws.min_selected) as min_selected, 
		       COALESCE(wsw.max_selected, ws.max_selected) as max_selected,
		       ws.created_at, ws.updated_at,
		       array_remove(array_agg(DISTINCT c.id), NULL) as category_ids,
		       array_remove(array_agg(DISTINCT c.name), NULL) as categories
		FROM wizard_steps_wizards wsw
		JOIN wizard_steps ws ON wsw.wizard_step_id = ws.id
		LEFT JOIN wizard_step_categories wsc ON ws.id = wsc.wizard_step_id
		LEFT JOIN categories c ON wsc.category_id = c.id
		WHERE wsw.wizard_id = $1
		GROUP BY ws.id, ws.name, ws.description, ws.step_order, ws.required, ws.multi_select,
		         ws.min_selected, ws.max_selected, wsw.step_order, wsw.required, 
		         wsw.multi_select, wsw.min_selected, wsw.max_selected, ws.created_at, ws.updated_at
		ORDER BY COALESCE(wsw.step_order, ws.step_order) ASC
	`

	rows, err := conn.Query(ctx, query, wizardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*WizardStep
	for rows.Next() {
		var step WizardStep
		var categoryIDs []string
		var categories []string

		err := rows.Scan(
			&step.ID,
			&step.Name,
			&step.Description,
			&step.StepOrder,
			&step.Required,
			&step.MultiSelect,
			&step.MinSelected,
			&step.MaxSelected,
			&step.CreatedAt,
			&step.UpdatedAt,
			&categoryIDs,
			&categories,
		)
		if err != nil {
			return nil, err
		}

		step.CategoryIDs = categoryIDs
		step.Categories = categories
		steps = append(steps, &step)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return steps, nil
}
