package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"event-engine-starter/internal/model"
)

// eventRow is a database-scannable representation of an Event.
type eventRow struct {
	ID                     string         `db:"id"`
	EventType              string         `db:"event_type"`
	FlowType               string         `db:"flow_type"`
	FlowID                 string         `db:"flow_id"`
	Status                 string         `db:"status"`
	Version                int64          `db:"version"`
	AttemptsLeft           int            `db:"attempts_left"`
	AttemptsFailed         int            `db:"attempts_failed"`
	AttemptScheduledAt     sql.NullTime   `db:"attempt_scheduled_at"`
	AttemptDueDate         sql.NullTime   `db:"attempt_due_date"`
	EventProcessingDueDate sql.NullTime   `db:"event_processing_due_date"`
	OnFailEventType        sql.NullString `db:"on_fail_event_type"`
	ScheduleState          string         `db:"schedule_state"`
	CreatedAt              time.Time      `db:"created_at"`
	UpdatedAt              time.Time      `db:"updated_at"`
}

func (r *eventRow) toModel() (*model.Event, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing event id %q: %w", r.ID, err)
	}

	e := &model.Event{
		ID:             id,
		EventType:      r.EventType,
		FlowType:       model.FlowType(r.FlowType),
		FlowID:         r.FlowID,
		Status:         model.EventStatus(r.Status),
		Version:        r.Version,
		AttemptsLeft:   r.AttemptsLeft,
		AttemptsFailed: r.AttemptsFailed,
		ScheduleState:  model.ScheduleState(r.ScheduleState),
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}

	if r.AttemptScheduledAt.Valid {
		e.AttemptScheduledAt = &r.AttemptScheduledAt.Time
	}
	if r.AttemptDueDate.Valid {
		e.AttemptDueDate = &r.AttemptDueDate.Time
	}
	if r.EventProcessingDueDate.Valid {
		e.EventProcessingDueDate = &r.EventProcessingDueDate.Time
	}
	if r.OnFailEventType.Valid {
		e.OnFailEventType = &r.OnFailEventType.String
	}

	return e, nil
}

// mysqlEventRepository is a MySQL-backed implementation of EventRepository.
type mysqlEventRepository struct {
	db *sqlx.DB
}

// NewMySQLEventRepository creates a new MySQL-backed EventRepository.
func NewMySQLEventRepository(db *sqlx.DB) EventRepository {
	return &mysqlEventRepository{db: db}
}

// Create inserts a new event using INSERT IGNORE for idempotency.
// Returns true if the row was inserted, false if it already existed.
func (r *mysqlEventRepository) Create(ctx context.Context, cmd model.EventInitiationCommand) (bool, error) {
	query := `
		INSERT IGNORE INTO event (
			id, event_type, flow_type, flow_id, status, version,
			attempts_left, attempts_failed,
			attempt_scheduled_at, attempt_due_date, event_processing_due_date,
			on_fail_event_type, schedule_state, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, 1,
			?, 0,
			?, ?, ?,
			?, ?, NOW(), NOW()
		)`

	result, err := r.db.ExecContext(ctx, query,
		cmd.EventID.String(), cmd.EventType, string(cmd.FlowType), cmd.FlowID, string(cmd.Status),
		cmd.AttemptsLeft,
		nullTimePtr(cmd.AttemptScheduledAt), nullTimePtr(cmd.AttemptDueDate), nullTimePtr(cmd.EventProcessingDueDate),
		nullStringPtr(cmd.OnFailEventType), string(cmd.ScheduleState),
	)
	if err != nil {
		return false, fmt.Errorf("creating event %s: %w", cmd.EventID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking rows affected for event %s: %w", cmd.EventID, err)
	}

	return rows > 0, nil
}

// Load retrieves a single event by ID.
func (r *mysqlEventRepository) Load(ctx context.Context, eventID uuid.UUID) (*model.Event, error) {
	query := `SELECT * FROM event WHERE id = ?`

	var row eventRow
	if err := r.db.GetContext(ctx, &row, query, eventID.String()); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrEventNotFound
		}
		return nil, fmt.Errorf("loading event %s: %w", eventID, err)
	}

	return row.toModel()
}

// FindByFlowAndCreatedBefore selects events matching the given flow, status, and creation time filter.
func (r *mysqlEventRepository) FindByFlowAndCreatedBefore(ctx context.Context, filter model.CreatedBeforeFilter) ([]model.Event, error) {
	query := `
		SELECT * FROM event
		WHERE flow_type = ? AND status = ? AND created_at <= ?
		LIMIT ?`

	var rows []eventRow
	if err := r.db.SelectContext(ctx, &rows, query,
		string(filter.FlowType), string(filter.Status), filter.CreatedBefore, filter.Limit,
	); err != nil {
		return nil, fmt.Errorf("finding events by flow %s and created before: %w", filter.FlowType, err)
	}

	return toModelSlice(rows)
}

// FindByFlowAndScheduledBefore selects events matching the given flow, schedule state,
// and scheduled time filter (inclusive).
func (r *mysqlEventRepository) FindByFlowAndScheduledBefore(ctx context.Context, filter model.ScheduledBeforeInclusiveFilter) ([]model.Event, error) {
	query := `
		SELECT * FROM event
		WHERE flow_type = ? AND schedule_state = ? AND attempt_scheduled_at <= ?
		LIMIT ?`

	var rows []eventRow
	if err := r.db.SelectContext(ctx, &rows, query,
		string(filter.FlowType), string(filter.ScheduleState), filter.ScheduledBefore, filter.Limit,
	); err != nil {
		return nil, fmt.Errorf("finding events by flow %s and scheduled before: %w", filter.FlowType, err)
	}

	return toModelSlice(rows)
}

// MarkAsDispatched transitions an event to DISPATCHED with optimistic locking.
func (r *mysqlEventRepository) MarkAsDispatched(ctx context.Context, cmd model.MarkAsDispatchedCommand) error {
	query := `
		UPDATE event
		SET status = ?, version = version + 1, schedule_state = ?, updated_at = NOW()
		WHERE id = ? AND version = ?`

	return r.execOptimistic(ctx, query, cmd.EventID, cmd.Version,
		string(model.StatusDispatched), string(cmd.ScheduleState),
		cmd.EventID.String(), cmd.Version,
	)
}

// AcquireProcessingPermit transitions an event to BEING_PROCESSED with optimistic locking.
func (r *mysqlEventRepository) AcquireProcessingPermit(ctx context.Context, cmd model.ProcessingPermitAcquisitionCommand) error {
	query := `
		UPDATE event
		SET status = ?, version = version + 1,
		    attempt_due_date = ?, event_processing_due_date = ?,
		    schedule_state = ?, updated_at = NOW()
		WHERE id = ? AND version = ?`

	return r.execOptimistic(ctx, query, cmd.EventID, cmd.Version,
		string(model.StatusBeingProcessed),
		nullTimePtr(cmd.AttemptDueDate), nullTimePtr(cmd.EventProcessingDueDate),
		string(cmd.ScheduleState),
		cmd.EventID.String(), cmd.Version,
	)
}

// SwitchToNextAttempt transitions an event back to AWAITING_PROCESSING with updated retry state.
func (r *mysqlEventRepository) SwitchToNextAttempt(ctx context.Context, cmd model.SwitchToNextAttemptCommand) error {
	query := `
		UPDATE event
		SET status = ?, version = version + 1,
		    attempts_left = ?, attempts_failed = ?,
		    attempt_scheduled_at = ?, attempt_due_date = ?,
		    schedule_state = ?, updated_at = NOW()
		WHERE id = ? AND version = ?`

	return r.execOptimistic(ctx, query, cmd.EventID, cmd.Version,
		string(model.StatusAwaitingProcessing),
		cmd.AttemptsLeft, cmd.AttemptsFailed,
		nullTimePtr(cmd.AttemptScheduledAt), nullTimePtr(cmd.AttemptDueDate),
		string(cmd.ScheduleState),
		cmd.EventID.String(), cmd.Version,
	)
}

// MarkAsSucceeded transitions an event to the PROCESSED terminal state.
func (r *mysqlEventRepository) MarkAsSucceeded(ctx context.Context, cmd model.MarkAsSucceededCommand) error {
	query := `
		UPDATE event
		SET status = ?, version = version + 1, schedule_state = ?, updated_at = NOW()
		WHERE id = ? AND version = ?`

	return r.execOptimistic(ctx, query, cmd.EventID, cmd.Version,
		string(model.StatusProcessed), string(model.ScheduleStateInactive),
		cmd.EventID.String(), cmd.Version,
	)
}

// MarkAsFailed transitions an event to the FAILED terminal state.
func (r *mysqlEventRepository) MarkAsFailed(ctx context.Context, cmd model.MarkAsFailedCommand) error {
	query := `
		UPDATE event
		SET status = ?, version = version + 1,
		    attempts_failed = ?, schedule_state = ?,
		    updated_at = NOW()
		WHERE id = ? AND version = ?`

	return r.execOptimistic(ctx, query, cmd.EventID, cmd.Version,
		string(model.StatusFailed),
		cmd.AttemptsFailed, string(model.ScheduleStateInactive),
		cmd.EventID.String(), cmd.Version,
	)
}

// ActivateSuspended reactivates a suspended event back to AWAITING_PROCESSING.
func (r *mysqlEventRepository) ActivateSuspended(ctx context.Context, cmd model.ActivateSuspendedCommand) error {
	query := `
		UPDATE event
		SET status = ?, version = version + 1, schedule_state = ?, updated_at = NOW()
		WHERE id = ? AND version = ?`

	return r.execOptimistic(ctx, query, cmd.EventID, cmd.Version,
		string(model.StatusAwaitingProcessing), string(cmd.ScheduleState),
		cmd.EventID.String(), cmd.Version,
	)
}

// GetQueueSize returns the count of events matching the given flow type and status.
func (r *mysqlEventRepository) GetQueueSize(ctx context.Context, flowType model.FlowType, status model.EventStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM event WHERE flow_type = ? AND status = ?`

	var count int64
	if err := r.db.GetContext(ctx, &count, query, string(flowType), string(status)); err != nil {
		return 0, fmt.Errorf("getting queue size for flow %s status %s: %w", flowType, status, err)
	}

	return count, nil
}

// execOptimistic executes an UPDATE with optimistic locking. If no rows are
// affected, it returns a RaceConditionError wrapping model.ErrRaceCondition.
func (r *mysqlEventRepository) execOptimistic(ctx context.Context, query string, eventID uuid.UUID, version int64, args ...interface{}) error {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("updating event %s: %w", eventID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for event %s: %w", eventID, err)
	}

	if rows == 0 {
		return &model.RaceConditionError{
			EventID:         eventID.String(),
			ExpectedVersion: version,
			Message:         "no rows updated, version mismatch or event not found",
		}
	}

	return nil
}

// toModelSlice converts a slice of eventRow to a slice of model.Event.
func toModelSlice(rows []eventRow) ([]model.Event, error) {
	events := make([]model.Event, 0, len(rows))
	for _, row := range rows {
		e, err := row.toModel()
		if err != nil {
			return nil, err
		}
		events = append(events, *e)
	}
	return events, nil
}

// nullTimePtr converts a *time.Time to a sql.NullTime-compatible value for query args.
func nullTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

// nullStringPtr converts a *string to a sql.NullString-compatible value for query args.
func nullStringPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}
