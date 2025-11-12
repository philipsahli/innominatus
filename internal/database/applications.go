package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"innominatus/internal/types"
	"time"

	"github.com/lib/pq"
)

// Application represents a Score specification stored in the database
type Application struct {
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	ScoreSpec *types.ScoreSpec `json:"score_spec"`
	Team      string           `json:"team"`
	CreatedBy string           `json:"created_by"`
	Labels    []string         `json:"labels"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// Environment represents an environment configuration
type Environment struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	TTL       string            `json:"ttl"`
	Status    string            `json:"status"`
	Resources map[string]string `json:"resources"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// AddApplication stores a new application with its Score spec
func (d *Database) AddApplication(name string, spec *types.ScoreSpec, team string, createdBy string) error {
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal score spec: %w", err)
	}

	query := `
		INSERT INTO applications (name, score_spec, team, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET
			score_spec = EXCLUDED.score_spec,
			team = EXCLUDED.team,
			created_by = EXCLUDED.created_by,
			updated_at = NOW()
	`

	_, err = d.db.Exec(query, name, specJSON, team, createdBy)
	if err != nil {
		return fmt.Errorf("failed to insert application: %w", err)
	}

	return nil
}

// GetApplication retrieves an application by name
func (d *Database) GetApplication(name string) (*Application, error) {
	query := `
		SELECT id, name, score_spec, team, created_by, COALESCE(labels, '{}'), created_at, updated_at
		FROM applications
		WHERE name = $1
	`

	var app Application
	var specJSON []byte

	err := d.db.QueryRow(query, name).Scan(
		&app.ID,
		&app.Name,
		&specJSON,
		&app.Team,
		&app.CreatedBy,
		pq.Array(&app.Labels),
		&app.CreatedAt,
		&app.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("application not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query application: %w", err)
	}

	// Unmarshal Score spec
	var spec types.ScoreSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal score spec: %w", err)
	}
	app.ScoreSpec = &spec

	return &app, nil
}

// ListApplications returns all applications
func (d *Database) ListApplications() ([]*Application, error) {
	query := `
		SELECT id, name, score_spec, team, created_by, created_at, updated_at
		FROM applications
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var apps []*Application
	for rows.Next() {
		var app Application
		var specJSON []byte

		err := rows.Scan(
			&app.ID,
			&app.Name,
			&specJSON,
			&app.Team,
			&app.CreatedBy,
			&app.CreatedAt,
			&app.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan application: %w", err)
		}

		// Unmarshal Score spec
		var spec types.ScoreSpec
		if err := json.Unmarshal(specJSON, &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal score spec: %w", err)
		}
		app.ScoreSpec = &spec

		apps = append(apps, &app)
	}

	return apps, nil
}

// ListApplicationsByTeam returns applications for a specific team
func (d *Database) ListApplicationsByTeam(team string) ([]*Application, error) {
	query := `
		SELECT id, name, score_spec, team, created_by, created_at, updated_at
		FROM applications
		WHERE team = $1
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query, team)
	if err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var apps []*Application
	for rows.Next() {
		var app Application
		var specJSON []byte

		err := rows.Scan(
			&app.ID,
			&app.Name,
			&specJSON,
			&app.Team,
			&app.CreatedBy,
			&app.CreatedAt,
			&app.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan application: %w", err)
		}

		// Unmarshal Score spec
		var spec types.ScoreSpec
		if err := json.Unmarshal(specJSON, &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal score spec: %w", err)
		}
		app.ScoreSpec = &spec

		apps = append(apps, &app)
	}

	return apps, nil
}

// DeleteApplication removes an application from the database
func (d *Database) DeleteApplication(name string) error {
	query := `DELETE FROM applications WHERE name = $1`

	result, err := d.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("application not found")
	}

	return nil
}

// AddEnvironment stores a new environment
func (d *Database) AddEnvironment(env *Environment) error {
	resourcesJSON, err := json.Marshal(env.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources: %w", err)
	}

	query := `
		INSERT INTO environments (name, type, ttl, status, resources, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET
			type = EXCLUDED.type,
			ttl = EXCLUDED.ttl,
			status = EXCLUDED.status,
			resources = EXCLUDED.resources,
			updated_at = NOW()
	`

	_, err = d.db.Exec(query, env.Name, env.Type, env.TTL, env.Status, resourcesJSON)
	if err != nil {
		return fmt.Errorf("failed to insert environment: %w", err)
	}

	return nil
}

// GetEnvironment retrieves an environment by name
func (d *Database) GetEnvironment(name string) (*Environment, error) {
	query := `
		SELECT id, name, type, ttl, status, resources, created_at, updated_at
		FROM environments
		WHERE name = $1
	`

	var env Environment
	var resourcesJSON []byte

	err := d.db.QueryRow(query, name).Scan(
		&env.ID,
		&env.Name,
		&env.Type,
		&env.TTL,
		&env.Status,
		&resourcesJSON,
		&env.CreatedAt,
		&env.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query environment: %w", err)
	}

	// Unmarshal resources
	if err := json.Unmarshal(resourcesJSON, &env.Resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	return &env, nil
}

// ListEnvironments returns all environments
func (d *Database) ListEnvironments() ([]*Environment, error) {
	query := `
		SELECT id, name, type, ttl, status, resources, created_at, updated_at
		FROM environments
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query environments: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var envs []*Environment
	for rows.Next() {
		var env Environment
		var resourcesJSON []byte

		err := rows.Scan(
			&env.ID,
			&env.Name,
			&env.Type,
			&env.TTL,
			&env.Status,
			&resourcesJSON,
			&env.CreatedAt,
			&env.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}

		// Unmarshal resources
		if err := json.Unmarshal(resourcesJSON, &env.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		envs = append(envs, &env)
	}

	return envs, nil
}
