package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Event interface {
	EventType() string
	EventID() string
	OccurredAt() time.Time
	Payload() interface{}
}

type BaseEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

func (e BaseEvent) EventType() string {
	return e.Type
}

func (e BaseEvent) EventID() string {
	return e.ID
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

func (e BaseEvent) Payload() interface{} {
	return e.Data
}

type Handler func(ctx context.Context, event Event) error

type EventBus struct {
	handlers map[string][]Handler
	logger   *slog.Logger
	mu       sync.RWMutex
}

func NewEventBus(logger *slog.Logger) *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
		logger:   logger,
	}
}

func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.logger.Info("event handler registered",
		"event_type", eventType,
		"total_handlers", len(eb.handlers[eventType]))
}

func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers, exists := eb.handlers[event.EventType()]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		eb.logger.Debug("no handlers for event type", "event_type", event.EventType())
		return nil
	}

	eb.logger.Info("publishing event",
		"event_type", event.EventType(),
		"event_id", event.EventID(),
		"handlers_count", len(handlers))

	for _, handler := range handlers {
		go func(h Handler) {
			if err := h(ctx, event); err != nil {
				eb.logger.Error("event handler failed",
					"event_type", event.EventType(),
					"event_id", event.EventID(),
					"error", err)
			}
		}(handler)
	}

	return nil
}

func (eb *EventBus) PublishSync(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers, exists := eb.handlers[event.EventType()]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		eb.logger.Debug("no handlers for event type", "event_type", event.EventType())
		return nil
	}

	eb.logger.Info("publishing event synchronously",
		"event_type", event.EventType(),
		"event_id", event.EventID(),
		"handlers_count", len(handlers))

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			eb.logger.Error("event handler failed",
				"event_type", event.EventType(),
				"event_id", event.EventID(),
				"error", err)
			return fmt.Errorf("handler failed for event %s: %w", event.EventType(), err)
		}
	}

	return nil
}
