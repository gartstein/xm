package events

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

// MockKafkaWriter implements kafka.Writer for testing
type MockKafkaWriter struct {
	mock.Mock
}

func (m *MockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockKafkaWriter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewProducer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	producer, err := NewProducer([]string{"localhost:9092"}, logger, "topic")

	// Skip test if Kafka connection fails
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			t.Skip("Skipping test: Kafka broker not available")
		}
		t.Fatalf("Failed to create producer: %v", err)
	}

	assert.NotNil(t, producer.writer)
	assert.NotNil(t, producer.events)
	assert.NotNil(t, producer.closeChan)

	// Check logger name safely
	assert.Equal(t, "kafka_producer", producer.logger.Name())
}

func TestProducer_SendEvent(t *testing.T) {
	mockWriter := new(MockKafkaWriter)
	logger := zaptest.NewLogger(t)
	company := &models.Company{ID: uuid.New(), Name: "Test Company"}

	producer := &Producer{
		writer: mockWriter,
		logger: logger,
	}

	t.Run("successful send", func(t *testing.T) {
		mockWriter.On("WriteMessages", mock.Anything, mock.Anything).Return(nil)

		event := Event{Type: CompanyCreated, Company: company}
		producer.sendEvent(context.Background(), event)

		mockWriter.AssertCalled(t, "WriteMessages", mock.Anything, []kafka.Message{
			{
				Key:   []byte(company.ID.String()),
				Value: mustMarshal(&event),
			},
		})
	})

	t.Run("serialization error", func(t *testing.T) {
		core, recorded := observer.New(zap.ErrorLevel)
		producer.logger = zap.New(core)

		// Create valid company
		company := &models.Company{ID: uuid.New(), Name: "Valid Company"}

		// Mock JSON marshaling to force error
		oldMarshal := jsonMarshal
		jsonMarshal = func(_ interface{}) ([]byte, error) {
			return nil, errors.New("mock marshal error")
		}
		defer func() { jsonMarshal = oldMarshal }()

		event := Event{Type: CompanyCreated, Company: company}
		producer.sendEvent(context.Background(), event)

		// Verify error logging
		assert.Equal(t, 1, recorded.FilterMessage("Failed to serialize event").Len())
		assert.Equal(t, 1, recorded.FilterField(zap.String("company_id", company.ID.String())).Len())
	})

	t.Run("write error", func(t *testing.T) {
		core, recorded := observer.New(zap.ErrorLevel)
		producer.logger = zap.New(core)
		mockWriter.ExpectedCalls = nil
		mockWriter.On("WriteMessages", mock.Anything, mock.Anything).Return(errors.New("kafka error"))

		event := Event{Type: CompanyCreated, Company: company}
		producer.sendEvent(context.Background(), event)

		assert.Equal(t, 1, recorded.FilterMessage("Failed to produce event").Len())
	})
}

func TestProducer_Close(t *testing.T) {
	mockWriter := new(MockKafkaWriter)
	mockWriter.On("Close").Return(nil)

	producer := &Producer{
		writer:    mockWriter,
		closeChan: make(chan struct{}),
		logger:    zaptest.NewLogger(t),
	}

	producer.Close()

	// Verify close channel is closed
	select {
	case <-producer.closeChan:
	default:
		t.Error("closeChan not closed")
	}

	mockWriter.AssertCalled(t, "Close")
}

func TestProducer_EventLoop(t *testing.T) {
	mockWriter := new(MockKafkaWriter)
	mockWriter.On("WriteMessages", mock.Anything, mock.Anything).Return(nil)

	producer := &Producer{
		writer: mockWriter,
		events: make(chan Event, 1),
		logger: zaptest.NewLogger(t),
	}

	company := &models.Company{ID: uuid.New()}
	event := Event{Type: CompanyCreated, Company: company}

	// Start event loop
	go producer.eventLoop()

	// Send event
	producer.events <- event

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	mockWriter.AssertCalled(t, "WriteMessages", mock.Anything, mock.Anything)
}

func mustMarshal(c *Event) []byte {
	data, _ := json.Marshal(c)
	return data
}
