package service

import (
	"context"
	"fmt"
	"time"

	"event-engine-starter/internal/model"
	"event-engine-starter/internal/repository"
)

const configCacheTTL = 60 * time.Second

// EventConfigurationService provides cached access to event lifecycle
// configuration and manages suspension state changes.
type EventConfigurationService struct {
	repo  repository.ConfigRepository
	cache *TTLCache[string, *model.EventLifecycleConfig]
}

// NewEventConfigurationService creates a new configuration service backed by
// the given repository with a 60-second TTL cache.
func NewEventConfigurationService(repo repository.ConfigRepository) *EventConfigurationService {
	return &EventConfigurationService{
		repo:  repo,
		cache: NewTTLCache[string, *model.EventLifecycleConfig](configCacheTTL),
	}
}

// LoadConfig returns the lifecycle configuration for the given event type.
// Results are cached for 60 seconds to reduce database load.
func (s *EventConfigurationService) LoadConfig(ctx context.Context, eventType string) (*model.EventLifecycleConfig, error) {
	if cfg, ok := s.cache.Get(eventType); ok {
		return cfg, nil
	}

	cfg, err := s.repo.FindByEventType(ctx, eventType)
	if err != nil {
		return nil, fmt.Errorf("load config for event type %q: %w", eventType, err)
	}

	s.cache.Set(eventType, cfg)
	return cfg, nil
}

// UpdateConfig persists the given lifecycle configuration and evicts the
// cached entry so the next read picks up the change.
func (s *EventConfigurationService) UpdateConfig(ctx context.Context, cfg model.EventLifecycleConfig) error {
	if err := s.repo.Save(ctx, cfg); err != nil {
		return fmt.Errorf("update config for event type %q: %w", cfg.EventType, err)
	}
	s.cache.Delete(cfg.EventType)
	return nil
}

// SuspendProcessing marks the given event type as suspended so that the
// dispatcher skips events of this type.
func (s *EventConfigurationService) SuspendProcessing(ctx context.Context, eventType string) error {
	if err := s.repo.UpdateSuspensionState(ctx, eventType, true); err != nil {
		return fmt.Errorf("suspend processing for event type %q: %w", eventType, err)
	}
	s.cache.Delete(eventType)
	return nil
}

// UnsuspendProcessing clears the suspension flag for the given event type,
// allowing the dispatcher to resume processing.
func (s *EventConfigurationService) UnsuspendProcessing(ctx context.Context, eventType string) error {
	if err := s.repo.UpdateSuspensionState(ctx, eventType, false); err != nil {
		return fmt.Errorf("unsuspend processing for event type %q: %w", eventType, err)
	}
	s.cache.Delete(eventType)
	return nil
}
