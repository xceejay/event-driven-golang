package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"event-engine-starter/internal/model"
)

// configRow is a database-scannable representation of EventLifecycleConfig.
type configRow struct {
	ID                      int64  `db:"id"`
	EventType               string `db:"event_type"`
	FlowType                string `db:"flow_type"`
	MaxAttempts             int    `db:"max_attempts"`
	EventLifespanSeconds    int64  `db:"event_lifespan_seconds"`
	IsSuspended             bool   `db:"is_suspended"`
	AttemptLifecycleConfigs string `db:"attempt_lifecycle_configs"`
}

func (r *configRow) toModel() (*model.EventLifecycleConfig, error) {
	cfg := &model.EventLifecycleConfig{
		ID:                   r.ID,
		EventType:            r.EventType,
		FlowType:             model.FlowType(r.FlowType),
		MaxAttempts:          r.MaxAttempts,
		EventLifespanSeconds: r.EventLifespanSeconds,
		IsSuspended:          r.IsSuspended,
	}

	if r.AttemptLifecycleConfigs != "" {
		if err := json.Unmarshal([]byte(r.AttemptLifecycleConfigs), &cfg.AttemptLifecycleConfigs); err != nil {
			return nil, fmt.Errorf("unmarshaling attempt_lifecycle_configs for %s: %w", r.EventType, err)
		}
	}

	return cfg, nil
}

// mysqlConfigRepository is a MySQL-backed implementation of ConfigRepository.
type mysqlConfigRepository struct {
	db *sqlx.DB
}

// NewMySQLConfigRepository creates a new MySQL-backed ConfigRepository.
func NewMySQLConfigRepository(db *sqlx.DB) ConfigRepository {
	return &mysqlConfigRepository{db: db}
}

// FindByEventType retrieves the lifecycle configuration for a specific event type.
// Returns model.ErrConfigNotFound if no configuration exists.
func (r *mysqlConfigRepository) FindByEventType(ctx context.Context, eventType string) (*model.EventLifecycleConfig, error) {
	query := `SELECT id, event_type, flow_type, max_attempts, event_lifespan_seconds, is_suspended, attempt_lifecycle_configs FROM event_lifecycle_config WHERE event_type = ?`

	var row configRow
	if err := r.db.GetContext(ctx, &row, query, eventType); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrConfigNotFound
		}
		return nil, fmt.Errorf("finding config for event type %s: %w", eventType, err)
	}

	return row.toModel()
}

// Save inserts or updates a lifecycle configuration using INSERT ... ON DUPLICATE KEY UPDATE.
func (r *mysqlConfigRepository) Save(ctx context.Context, cfg model.EventLifecycleConfig) error {
	attemptsJSON, err := json.Marshal(cfg.AttemptLifecycleConfigs)
	if err != nil {
		return fmt.Errorf("marshaling attempt_lifecycle_configs for %s: %w", cfg.EventType, err)
	}

	query := `
		INSERT INTO event_lifecycle_config (
			event_type, flow_type, max_attempts, event_lifespan_seconds,
			is_suspended, attempt_lifecycle_configs
		) VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			flow_type = VALUES(flow_type),
			max_attempts = VALUES(max_attempts),
			event_lifespan_seconds = VALUES(event_lifespan_seconds),
			is_suspended = VALUES(is_suspended),
			attempt_lifecycle_configs = VALUES(attempt_lifecycle_configs)`

	_, err = r.db.ExecContext(ctx, query,
		cfg.EventType, string(cfg.FlowType), cfg.MaxAttempts, cfg.EventLifespanSeconds,
		cfg.IsSuspended, string(attemptsJSON),
	)
	if err != nil {
		return fmt.Errorf("saving config for event type %s: %w", cfg.EventType, err)
	}

	return nil
}

// UpdateSuspensionState updates only the is_suspended flag for a given event type.
func (r *mysqlConfigRepository) UpdateSuspensionState(ctx context.Context, eventType string, suspended bool) error {
	query := `UPDATE event_lifecycle_config SET is_suspended = ? WHERE event_type = ?`

	_, err := r.db.ExecContext(ctx, query, suspended, eventType)
	if err != nil {
		return fmt.Errorf("updating suspension state for event type %s: %w", eventType, err)
	}

	return nil
}
