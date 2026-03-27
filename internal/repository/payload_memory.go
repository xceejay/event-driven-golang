package repository

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"event-engine-starter/internal/model"
)

// inMemoryPayloadRepository is an in-memory implementation of PayloadRepository
// suitable for testing and development.
type inMemoryPayloadRepository struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewInMemoryPayloadRepository creates a new in-memory PayloadRepository.
func NewInMemoryPayloadRepository() PayloadRepository {
	return &inMemoryPayloadRepository{
		data: make(map[string]string),
	}
}

// Save stores an event payload in memory.
func (r *inMemoryPayloadRepository) Save(_ context.Context, eventID uuid.UUID, payload string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[eventID.String()] = payload
	return nil
}

// Load retrieves an event payload from memory. Returns model.ErrPayloadNotFound
// if no payload exists for the given event ID.
func (r *inMemoryPayloadRepository) Load(_ context.Context, eventID uuid.UUID) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val, ok := r.data[eventID.String()]
	if !ok {
		return "", model.ErrPayloadNotFound
	}
	return val, nil
}
