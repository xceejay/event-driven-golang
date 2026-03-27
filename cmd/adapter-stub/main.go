package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// eventMessage mirrors the message published by the event dispatcher.
type eventMessage struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	FlowType  string `json:"flow_type"`
	FlowID    string `json:"flow_id"`
}

// rideStep defines how to process a ride event type.
type rideStep struct {
	minDelay    time.Duration
	maxDelay    time.Duration
	nextEvent   string // spinoff event type on success (empty = terminal)
	failChance  int    // 1 in N chance of failure; 0 = never fail
	description string
}

// Ride-hailing event chain with realistic timing.
var rideSteps = map[string]rideStep{
	"ride_requested": {
		minDelay:    3 * time.Second,
		maxDelay:    6 * time.Second,
		nextEvent:   "DRIVER_MATCHED",
		failChance:  8, // ~12% — sometimes no driver available
		description: "searching for nearby drivers",
	},
	"driver_matched": {
		minDelay:    4 * time.Second,
		maxDelay:    8 * time.Second,
		nextEvent:   "DRIVER_ARRIVED",
		failChance:  15, // ~7% — driver cancels
		description: "driver heading to pickup",
	},
	"driver_arrived": {
		minDelay:    2 * time.Second,
		maxDelay:    4 * time.Second,
		nextEvent:   "TRIP_STARTED",
		failChance:  20, // ~5% — rider no-show
		description: "waiting for rider",
	},
	"trip_started": {
		minDelay:    5 * time.Second,
		maxDelay:    10 * time.Second,
		nextEvent:   "TRIP_COMPLETED",
		failChance:  0, // trips don't fail mid-ride
		description: "driving to destination",
	},
	"trip_completed": {
		minDelay:    1 * time.Second,
		maxDelay:    2 * time.Second,
		nextEvent:   "PAYMENT_PROCESSED",
		failChance:  0,
		description: "calculating fare",
	},
	"payment_processed": {
		minDelay:    1 * time.Second,
		maxDelay:    3 * time.Second,
		nextEvent:   "", // terminal — ride flow done
		failChance:  10, // ~10% — payment fails, triggers retry
		description: "processing payment",
	},
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[adapter-stub] ")

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	engineURL := os.Getenv("ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("connected to NATS at %s", natsURL)

	sub, err := nc.QueueSubscribe("event-engine.*", "adapter-stub", func(msg *nats.Msg) {
		handleMessage(engineURL, msg)
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	log.Printf("subscribed to event-engine.* (queue: adapter-stub)")
	log.Printf("engine API at %s", engineURL)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down")
}

func handleMessage(engineURL string, msg *nats.Msg) {
	// Check TTL.
	if ttlHeader := msg.Header.Get("Published-At"); ttlHeader != "" {
		publishedAt, err := time.Parse(time.RFC3339, ttlHeader)
		if err == nil {
			ttlStr := msg.Header.Get("Nats-TTL")
			if ttlStr != "" {
				var ttlSec int
				fmt.Sscanf(ttlStr, "%d", &ttlSec)
				if time.Since(publishedAt) > time.Duration(ttlSec)*time.Second {
					log.Printf("discarding expired message (published %s, TTL %ds)", ttlHeader, ttlSec)
					return
				}
			}
		}
	}

	var event eventMessage
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("unmarshal error: %v", err)
		return
	}

	subject := msg.Subject
	eventType := strings.TrimPrefix(subject, "event-engine.")
	log.Printf("received event %s (type=%s, flow=%s)", event.EventID, eventType, event.FlowType)

	// Acquire processing permit.
	acquired, err := acquirePermit(engineURL, event.EventID)
	if err != nil {
		log.Printf("acquire permit error for %s: %v", event.EventID, err)
		return
	}
	if !acquired {
		log.Printf("permit not granted for %s (another instance got it)", event.EventID)
		return
	}
	log.Printf("permit acquired for %s", event.EventID)

	// Look up ride step config, or use generic default.
	step, isRide := rideSteps[strings.ToLower(eventType)]
	if !isRide {
		// Generic non-ride event: fast processing, 20% fail.
		log.Printf("processing event %s (generic, 500ms)...", event.EventID)
		time.Sleep(500 * time.Millisecond)
		if rand.Intn(5) == 0 {
			reportFailure(engineURL, event.EventID, "simulated random failure")
			return
		}
		reportSuccess(engineURL, event.EventID, nil)
		return
	}

	// Ride event: simulate with realistic timing.
	delay := step.minDelay + time.Duration(rand.Int63n(int64(step.maxDelay-step.minDelay)))
	log.Printf("[ride] %s — %s (%s)...", event.EventID, step.description, delay)
	time.Sleep(delay)

	// Check for failure.
	if step.failChance > 0 && rand.Intn(step.failChance) == 0 {
		failMsg := fmt.Sprintf("%s failed: simulated failure during '%s'", eventType, step.description)
		log.Printf("[ride] %s FAILED: %s", event.EventID, failMsg)
		reportFailure(engineURL, event.EventID, failMsg)
		return
	}

	// Success — include spinoff if there's a next step.
	var spinoffs []map[string]string
	if step.nextEvent != "" {
		spinoffs = []map[string]string{
			{"event_type": step.nextEvent, "payload": msg.Header.Get("Event-Payload")},
		}
		log.Printf("[ride] %s SUCCESS — chaining to %s", event.EventID, step.nextEvent)
	} else {
		log.Printf("[ride] %s SUCCESS — ride flow complete!", event.EventID)
	}

	reportSuccess(engineURL, event.EventID, spinoffs)
}

func acquirePermit(engineURL, eventID string) (bool, error) {
	resp, err := http.Post(engineURL+"/api/events/"+eventID+"/acquire", "application/json", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Acquired bool `json:"acquired"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Acquired, nil
}

func reportSuccess(engineURL, eventID string, spinoffs []map[string]string) error {
	body, _ := json.Marshal(map[string]any{"spinoffs": spinoffs})
	resp, err := http.Post(
		engineURL+"/api/events/"+eventID+"/success",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func reportFailure(engineURL, eventID, errorMessage string) error {
	body, _ := json.Marshal(map[string]string{"error_message": errorMessage})
	resp, err := http.Post(
		engineURL+"/api/events/"+eventID+"/failure",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
