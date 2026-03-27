package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"event-engine-starter/internal/model"
)

const payloadKeyPrefix = "event:payload:"

// redisPayloadRepository is a Redis-backed implementation of PayloadRepository.
type redisPayloadRepository struct {
	client *redis.Client
}

// NewRedisPayloadRepository creates a new Redis-backed PayloadRepository.
func NewRedisPayloadRepository(client *redis.Client) PayloadRepository {
	return &redisPayloadRepository{client: client}
}

// Save stores an event payload in Redis with no expiry.
func (r *redisPayloadRepository) Save(ctx context.Context, eventID uuid.UUID, payload string) error {
	key := payloadKeyPrefix + eventID.String()
	if err := r.client.Set(ctx, key, payload, 0).Err(); err != nil {
		return fmt.Errorf("saving payload for event %s: %w", eventID, err)
	}
	return nil
}

// Load retrieves an event payload from Redis. Returns model.ErrPayloadNotFound
// if the key does not exist.
func (r *redisPayloadRepository) Load(ctx context.Context, eventID uuid.UUID) (string, error) {
	key := payloadKeyPrefix + eventID.String()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", model.ErrPayloadNotFound
		}
		return "", fmt.Errorf("loading payload for event %s: %w", eventID, err)
	}
	return val, nil
}
