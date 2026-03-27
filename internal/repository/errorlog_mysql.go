package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"event-engine-starter/internal/model"
)

// mysqlErrorLogRepository is a MySQL-backed implementation of ErrorLogRepository.
type mysqlErrorLogRepository struct {
	db *sqlx.DB
}

// NewMySQLErrorLogRepository creates a new MySQL-backed ErrorLogRepository.
func NewMySQLErrorLogRepository(db *sqlx.DB) ErrorLogRepository {
	return &mysqlErrorLogRepository{db: db}
}

// Save inserts a new error log entry.
func (r *mysqlErrorLogRepository) Save(ctx context.Context, log model.EventErrorLog) error {
	query := `
		INSERT INTO event_error_log (id, event_id, error_message, stack_trace, occurred_at)
		VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		log.ID.String(), log.EventID.String(), log.ErrorMessage, log.StackTrace, log.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("saving error log for event %s: %w", log.EventID, err)
	}

	return nil
}

// FindByEventID retrieves all error logs for a given event, ordered by most recent first.
func (r *mysqlErrorLogRepository) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]model.EventErrorLog, error) {
	query := `
		SELECT id, event_id, error_message, stack_trace, occurred_at
		FROM event_error_log
		WHERE event_id = ?
		ORDER BY occurred_at DESC`

	var logs []model.EventErrorLog
	if err := r.db.SelectContext(ctx, &logs, query, eventID.String()); err != nil {
		return nil, fmt.Errorf("finding error logs for event %s: %w", eventID, err)
	}

	return logs, nil
}
