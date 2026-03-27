package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"event-engine-starter/internal/model"
)

// ttlBuckets are the available message TTL buckets in seconds, ascending.
var ttlBuckets = []int{30, 75, 150, 300, 600, 1200, 3600, 7200, 86400}

// eventMessage is the JSON payload published to the broker.
type eventMessage struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	FlowType  string `json:"flow_type"`
	FlowID    string `json:"flow_id"`
}

// EventDispatcher publishes events to NATS with TTL-aware message expiry.
type EventDispatcher struct {
	nc *nats.Conn
}

// NewEventDispatcher creates a dispatcher backed by the given NATS connection.
func NewEventDispatcher(nc *nats.Conn) *EventDispatcher {
	return &EventDispatcher{nc: nc}
}

// Dispatch publishes an event to the broker subject event-engine.<eventType>.
// The message TTL is selected based on the remaining time until the event's
// processing due date. If the event is already expired, an error is returned.
func (d *EventDispatcher) Dispatch(_ context.Context, event *model.Event, payload string) error {
	if event.EventProcessingDueDate == nil {
		return fmt.Errorf("event %s has no processing due date", event.ID)
	}

	remaining := time.Until(*event.EventProcessingDueDate)
	if remaining <= 0 {
		return fmt.Errorf("event %s is expired (due date %s has passed)", event.ID, event.EventProcessingDueDate)
	}

	ttl := selectTTLBucket(remaining)
	subject := "event-engine." + strings.ToLower(event.EventType)

	msg := eventMessage{
		EventID:   event.ID.String(),
		EventType: event.EventType,
		FlowType:  string(event.FlowType),
		FlowID:    event.FlowID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal event message: %w", err)
	}

	natsMsg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}
	natsMsg.Header.Set("Nats-TTL", fmt.Sprintf("%d", ttl))
	natsMsg.Header.Set("Published-At", time.Now().UTC().Format(time.RFC3339))
	natsMsg.Header.Set("Event-Payload", payload)

	if err := d.nc.PublishMsg(natsMsg); err != nil {
		return fmt.Errorf("publish event %s to %s: %w", event.ID, subject, err)
	}

	return nil
}

// selectTTLBucket returns the smallest TTL bucket >= the remaining duration.
func selectTTLBucket(remaining time.Duration) int {
	remainingSec := int(remaining.Seconds())
	for _, bucket := range ttlBuckets {
		if bucket >= remainingSec {
			return bucket
		}
	}
	return ttlBuckets[len(ttlBuckets)-1]
}
