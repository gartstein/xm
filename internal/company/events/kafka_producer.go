package events

import (
	"context"
	"encoding/json"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var jsonMarshal = json.Marshal

type EventType string

const (
	CompanyCreated EventType = "company_created"
	CompanyUpdated EventType = "company_updated"
	CompanyDeleted EventType = "company_deleted"
)

type Event struct {
	Type    EventType
	Company *models.Company
}

type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type Producer struct {
	writer    KafkaWriter // Use interface instead of concrete type
	events    chan Event
	logger    *zap.Logger
	closeChan chan struct{}
}

func NewProducer(brokers []string, logger *zap.Logger, topic string) (*Producer, error) {
	// Create topic if it doesn't exist
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil {
		logger.Warn("failed to create topic (may already exist)", zap.Error(err))
	}
	p := &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
			Topic:    topic,
		},
		events:    make(chan Event, 1000), // Buffered channel
		logger:    logger.Named("kafka_producer"),
		closeChan: make(chan struct{}),
	}

	go p.eventLoop()
	return p, nil
}

func (p *Producer) Produce(eventType EventType, company *models.Company) {
	select {
	case p.events <- Event{Type: eventType, Company: company}:
	default:
		p.logger.Warn("Kafka producer queue full, dropping event",
			zap.String("event_type", string(eventType)),
			zap.String("company_id", company.ID.String()),
		)
	}
}

func (p *Producer) eventLoop() {
	for {
		select {
		case event := <-p.events:
			p.sendEvent(context.Background(), event)
		case <-p.closeChan:
			return
		}
	}
}

func (p *Producer) sendEvent(ctx context.Context, event Event) {
	value, err := jsonMarshal(event)
	if err != nil {
		p.logger.Error("Failed to serialize event",
			zap.Error(err),
			zap.String("company_id", event.Company.ID.String()),
		)
		return
	}
	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Company.ID.String()),
		Value: value,
	})
	if err != nil {
		p.logger.Error("Failed to produce event",
			zap.Error(err),
			zap.String("event_type", string(event.Type)),
			zap.String("company_id", event.Company.ID.String()),
		)
		return
	}
}

func (p *Producer) Close() {
	close(p.closeChan)
	if err := p.writer.Close(); err != nil {
		p.logger.Error("Failed to close Kafka writer", zap.Error(err))
	}
}
