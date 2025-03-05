package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gartstein/xm/internal/company/controller"
	"github.com/gartstein/xm/internal/company/db"
	e "github.com/gartstein/xm/internal/company/errors"
	"github.com/gartstein/xm/internal/company/events"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type kafkaEvent struct {
	Key     string
	Type    events.EventType
	Company *models.Company
}

type IntegrationTestSuite struct {
	suite.Suite
	dbRepo       *db.Repository
	kafkaReader  *kafka.Reader
	producer     *events.Producer
	logger       *zap.Logger
	testTimeout  time.Duration
	cleanupFuncs []func()
}

func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.logger = zap.NewNop()
	s.testTimeout = 20 * time.Second

	// Initialize database with retries
	var dbErr error
	s.dbRepo, dbErr = initializeDBWithRetry()
	if dbErr != nil {
		s.T().Fatal("Database initialization failed:", dbErr)
	}
}

func initializeDBWithRetry() (*db.Repository, error) {
	cfg := &db.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "test",
		Password: "test",
		DBName:   "test",
		SSLMode:  "disable",
	}

	var repo *db.Repository
	var err error

	// Retry for 30 seconds
	err = backoff.Retry(func() error {
		repo, err = db.NewRepository(cfg)
		return err
	}, backoff.NewExponentialBackOff())

	return repo, err
}

func initializeKafkaWithRetry(topic string) (*events.Producer, *kafka.Reader, error) {
	kafkaBrokers := []string{"localhost:9092"}
	var producer *events.Producer
	var reader *kafka.Reader
	var err error
	// Retry producer initialization
	err = backoff.Retry(func() error {
		producer, err = events.NewProducer(kafkaBrokers, zap.NewNop(), topic)
		if err != nil || producer == nil {
			return fmt.Errorf("failed to create Kafka produce: %v", err)
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if err != nil {
		return nil, nil, fmt.Errorf("Kafka producer initialization failed: %w", err)
	}

	// Verify Kafka readiness using metadata instead of blocking on ReadMessage
	err = backoff.Retry(func() error {
		conn, err := kafka.Dial("tcp", kafkaBrokers[0])
		if err != nil {
			return err
		}
		defer conn.Close()

		// Fetch metadata and ensure the topic exists
		partitions, err := conn.ReadPartitions(topic)
		if err != nil || len(partitions) == 0 {
			return fmt.Errorf("topic company_created not found")
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)) // 🔹 Limit retries to avoid infinite loop

	if err != nil {
		return nil, nil, fmt.Errorf("Kafka topic check failed: %w", err)
	}

	// Initialize Kafka Reader (Without Blocking on ReadMessage)
	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     kafkaBrokers,
		Topic:       topic,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	return producer, reader, nil
}

func (s *IntegrationTestSuite) TearDownSuite() {
	for _, fn := range s.cleanupFuncs {
		fn()
	}
}

func (s *IntegrationTestSuite) SetupTest() {
	// Verify database connection
	if s.dbRepo == nil {
		s.T().Fatal("Database connection not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.testTimeout)
	defer cancel()

	// Clean database safely
	if err := s.dbRepo.Exec(ctx, "TRUNCATE TABLE companies CASCADE"); err != nil {
		s.T().Fatal("Failed to clean database:", err)
	}
}

func (s *IntegrationTestSuite) createCompany(ctx context.Context, service *controller.CompanyService, newCompany *models.Company) uuid.UUID {
	created, err := service.CreateCompany(ctx, newCompany)
	if err != nil {
		s.T().Fatal("CreateCompany failed:", err)
	}
	return created.ID
}

func (s *IntegrationTestSuite) TestCompanyCreate() {
	// Initialize Kafka components with retries
	var kafkaErr error
	s.producer, s.kafkaReader, kafkaErr = initializeKafkaWithRetry(string(events.CompanyCreated))
	if kafkaErr != nil {
		s.T().Fatal("Kafka initialization failed:", kafkaErr)
	}
	// Verify dependencies
	if s.dbRepo == nil || s.producer == nil {
		s.T().Fatal("Dependencies not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.testTimeout)
	defer cancel()

	ctrl := controller.NewCompanyService(s.dbRepo, s.producer, s.logger)
	newCompany := &models.Company{
		Name:        "New Company",
		Description: "Integration Test Company",
		Employees:   100,
		Registered:  true,
		Type:        models.Corporations,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	created, err := ctrl.CreateCompany(ctx, newCompany)
	if err != nil {
		s.T().Fatal("CreateCompany failed:", err)
	}

	assert.Equal(s.T(), newCompany.Name, created.Name)
	s.verifyKafkaEvent(ctx, events.CompanyCreated, created.ID)
}

func (s *IntegrationTestSuite) TestCompanyUpdate() {
	// Initialize Kafka components with retries
	var kafkaErr error
	s.producer, s.kafkaReader, kafkaErr = initializeKafkaWithRetry(string(events.CompanyCreated))
	if kafkaErr != nil {
		s.T().Fatal("Kafka initialization failed:", kafkaErr)
	}
	// Verify dependencies
	if s.dbRepo == nil || s.producer == nil {
		s.T().Fatal("Dependencies not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.testTimeout)
	defer cancel()

	ctrl := controller.NewCompanyService(s.dbRepo, s.producer, s.logger)
	newCompany := &models.Company{
		ID:          uuid.New(),
		Name:        "New Company",
		Description: "Integration Test Company",
		Employees:   100,
		Registered:  true,
		Type:        models.Corporations,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := s.dbRepo.CreateCompany(ctx, newCompany)
	if err != nil {
		s.T().Fatal("CreateCompany failed:", err)
	}
	newName := "Updated Company"
	update := &models.CompanyUpdate{
		ID:          newCompany.ID,
		Name:        &newName,
		Description: &newCompany.Description,
		Employees:   &newCompany.Employees,
	}

	updatedCompany, err := ctrl.UpdateCompany(ctx, update)
	if err != nil {
		s.T().Fatal("UpdateCompany failed:", err)
	}

	assert.Equal(s.T(), newName, updatedCompany.Name)
	time.Sleep(2 * time.Second)
	s.verifyKafkaEvent(ctx, events.CompanyUpdated, updatedCompany.ID)
}

func (s *IntegrationTestSuite) TestCompanyDelete() {
	// Initialize Kafka components with retries
	var kafkaErr error
	s.producer, s.kafkaReader, kafkaErr = initializeKafkaWithRetry(string(events.CompanyDeleted))

	if kafkaErr != nil {
		s.T().Fatal("Kafka initialization failed:", kafkaErr)
	}
	// Verify dependencies
	if s.dbRepo == nil || s.producer == nil {
		s.T().Fatal("Dependencies not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.testTimeout)
	defer cancel()

	ctrl := controller.NewCompanyService(s.dbRepo, s.producer, s.logger)

	company := &models.Company{
		Name:        "New Company",
		Description: "Integration Test Company",
		Employees:   100,
		Registered:  true,
		Type:        models.Corporations,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := s.dbRepo.CreateCompany(ctx, company)
	if err != nil {
		s.T().Fatal("CreateCompany failed:", err)
	}
	err = ctrl.DeleteCompany(ctx, company.ID)
	if err != nil {
		s.T().Fatal("DeleteCompany failed:", err)
	}

	_, err = s.dbRepo.GetCompany(ctx, company.ID)
	assert.ErrorIs(s.T(), err, e.ErrNotFound)
	s.T().Logf("Deleted companyID=%s", company.ID.String())
	time.Sleep(2 * time.Second)
	s.verifyKafkaEvent(ctx, events.CompanyDeleted, company.ID)
}

func (s *IntegrationTestSuite) verifyKafkaEvent(ctx context.Context, eventType events.EventType, companyID uuid.UUID) {
	event := s.consumeKafkaEvent(ctx, eventType, companyID)

	if event.Company == nil {
		s.T().Fatal("Received nil company in Kafka event")
	}

	assert.Equal(s.T(), companyID.String(), event.Company.ID.String(), "Kafka message company ID mismatch")
}

func (s *IntegrationTestSuite) consumeKafkaEvent(ctx context.Context, eventType events.EventType, companyID uuid.UUID) kafkaEvent {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	maxRetries := 200
	attempts := 0
	for {
		select {
		case <-ctx.Done():
			s.T().Fatalf("Timeout: No %s event received after %d attempts", eventType, attempts)
			return kafkaEvent{}
		default:
			if attempts >= maxRetries {
				s.T().Fatalf("Max retry attempts reached for %s", eventType)
				return kafkaEvent{}
			}
			msg, err := s.kafkaReader.ReadMessage(ctx)
			if err != nil {
				s.T().Logf("Kafka read attempt %d failed: %v", attempts, err)
				attempts++
				time.Sleep(1 * time.Second)
				continue
			}
			s.T().Logf("Received Kafka message: Topic=%s Key=%s", msg.Topic, string(msg.Key))
			fmt.Println("Received Kafka: ", string(msg.Value))
			if string(msg.Key) != companyID.String() {
				s.T().Logf("Skipping message with unmatched key: %s (Expected: %s)", string(msg.Key), companyID.String())
				attempts++
				continue
			}
			var event kafkaEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				s.T().Fatalf("Failed to unmarshal Kafka message: %v", err)
			}
			if event.Type != eventType {
				s.T().Logf("Skipping message with unmatched eventType: %s (Expected: %s)", string(event.Type), eventType)
				attempts++
				continue
			}
			s.T().Logf("Successfully consumed event: %s, ID=%s, attempts=%d", eventType, event.Company.ID.String(), attempts)
			return kafkaEvent{
				Key:     string(msg.Key),
				Company: event.Company,
				Type:    eventType,
			}
		}
	}
}
