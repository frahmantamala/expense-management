package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/frahmantamala/expense-management/internal/core/events"
	"github.com/frahmantamala/expense-management/pkg/logger"
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Event management commands",
	Long:  `Manage events: publish test events, monitor event bus, inspect handlers`,
}

var publishEventCmd = &cobra.Command{
	Use:   "publish [event-type]",
	Short: "Publish a test event",
	Long:  `Publish a test event to the event bus for testing and debugging`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		publishTestEvent(args[0])
	},
}

var eventData string

func publishTestEvent(eventType string) {
	logger := logger.LoggerWrapper()

	eventBus := events.NewEventBus(logger)

	eventBus.Subscribe(eventType, func(ctx context.Context, event events.Event) error {
		logger.Info("test handler received event",
			"event_id", event.EventID(),
			"event_type", event.EventType(),
			"payload", event.Payload())
		return nil
	})

	testEvent := events.BaseEvent{
		ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
		Type:      eventType,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": eventData,
			"source":  "cli-command",
		},
	}

	logger.Info("publishing test event", "event_type", eventType, "event_id", testEvent.ID)

	ctx := context.Background()
	if err := eventBus.Publish(ctx, testEvent); err != nil {
		logger.Error("failed to publish event", "error", err)
		return
	}

	time.Sleep(100 * time.Millisecond)
	logger.Info("test event published successfully")
}

func init() {

	publishEventCmd.Flags().StringVar(&eventData, "data", "test message", "Event data message")

	eventCmd.AddCommand(publishEventCmd)

	rootCmd.AddCommand(eventCmd)
}
