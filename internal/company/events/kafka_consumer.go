package events

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer struct {
	reader  *kafka.Reader
	logger  *zap.Logger
	handler func(context.Context, Event) error
}

// NewConsumer consumes kafka events.
// TODO: implement consuming logic if there is time
func NewConsumer(brokers []string, groupID string, logger *zap.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			GroupID: groupID,
			Topic:   "company.*",
			Dialer:  kafka.DefaultDialer,
		}),
		logger: logger.Named("kafka_consumer"),
	}
}

func (c *Consumer) Start(ctx context.Context) {
	go func() {
		for {
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("Failed to fetch message", zap.Error(err))
				continue
			}

			var event Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				c.logger.Error("Failed to parse event",
					zap.Error(err),
					zap.ByteString("value", msg.Value),
				)
				continue
			}

			if err := c.handler(ctx, event); err != nil {
				c.logger.Error("Failed to handle event",
					zap.Error(err),
					zap.String("event_type", string(event.Type)),
				)
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("Failed to commit message",
					zap.Error(err),
					zap.String("event_type", string(event.Type)),
				)
			}
		}
	}()
}

func (c *Consumer) RegisterHandler(fn func(context.Context, Event) error) {
	c.handler = fn
}

func (c *Consumer) Close() {
	if err := c.reader.Close(); err != nil {
		c.logger.Error("Failed to close Kafka reader", zap.Error(err))
	}
}
