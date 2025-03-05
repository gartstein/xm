package controller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gartstein/xm/internal/company/db"
	e "github.com/gartstein/xm/internal/company/errors"
	"github.com/gartstein/xm/internal/company/events"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/gartstein/xm/internal/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"
)

// MockRepository implements the Repository interface for testing
type MockRepository struct {
	createCompany       func(context.Context, *models.Company) error
	getCompany          func(context.Context, uuid.UUID) (*models.Company, error)
	updateCompany       func(context.Context, *models.CompanyUpdate) error
	deleteCompany       func(context.Context, uuid.UUID) error
	companyExistsByName func(context.Context, string) (bool, error)
	withTransaction     func(context.Context, func(*db.Repository) error) error
}

func (m *MockRepository) CreateCompany(ctx context.Context, c *models.Company) error {
	return m.createCompany(ctx, c)
}

func (m *MockRepository) GetCompany(ctx context.Context, id uuid.UUID) (*models.Company, error) {
	return m.getCompany(ctx, id)
}

func (m *MockRepository) UpdateCompany(ctx context.Context, u *models.CompanyUpdate) error {
	return m.updateCompany(ctx, u)
}

func (m *MockRepository) Close() error {
	return nil
}

func (m *MockRepository) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return m.deleteCompany(ctx, id)
}

func (m *MockRepository) CompanyExistsByName(ctx context.Context, name string) (bool, error) {
	return m.companyExistsByName(ctx, name)
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(*db.Repository) error) error {
	return m.withTransaction(ctx, fn)
}

// MockProducer is a test double for the Kafka producer.
type MockProducer struct {
	producedEvents []interface{}
	wg             *sync.WaitGroup
}

// Produce records the event and signals the wait group.
func (m *MockProducer) Produce(eventType events.EventType, company *models.Company) {
	m.producedEvents = append(m.producedEvents, struct {
		EventType events.EventType
		Company   *models.Company
	}{eventType, company})
	if m.wg != nil {
		m.wg.Done()
	}
}

func TestCompanyService_CreateCompany(t *testing.T) {
	testID := uuid.New()
	now := time.Now()

	tests := []struct {
		name          string
		input         *models.Company
		mockSetup     func(*MockRepository, *MockProducer)
		expectError   bool
		expectedError error
	}{
		{
			name: "successful creation",
			input: &models.Company{
				Name:        "Valid Name",
				Description: "Valid Desc",
				Employees:   50,
				Registered:  true,
				Type:        models.NonProfit,
			},
			mockSetup: func(mr *MockRepository, _ *MockProducer) {
				mr.companyExistsByName = func(_ context.Context, _ string) (bool, error) {
					return false, nil
				}
				mr.createCompany = func(_ context.Context, c *models.Company) error {
					c.ID = testID
					c.CreatedAt = now
					c.UpdatedAt = now
					return nil
				}
			},
			expectError: false,
		},
		{
			name: "duplicate name",
			input: &models.Company{
				Name: "Duplicate",
			},
			mockSetup: func(mr *MockRepository, _ *MockProducer) {
				mr.companyExistsByName = func(_ context.Context, _ string) (bool, error) {
					return true, nil
				}
			},
			expectError:   true,
			expectedError: e.ErrDuplicateName,
		},
		{
			name: "invalid name length",
			input: &models.Company{
				Name: "This name is way too long for the validation",
			},
			mockSetup:   func(_ *MockRepository, _ *MockProducer) {},
			expectError: true,
		},
		{
			name: "repository error",
			input: &models.Company{
				Name: "Valid",
			},
			mockSetup: func(mr *MockRepository, _ *MockProducer) {
				mr.companyExistsByName = func(_ context.Context, _ string) (bool, error) {
					return false, nil
				}
				mr.createCompany = func(_ context.Context, _ *models.Company) error {
					return errors.New("database error")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			mockRepo := &MockRepository{}
			mockProducer := &MockProducer{wg: new(sync.WaitGroup)}
			tt.mockSetup(mockRepo, mockProducer)
			service := NewCompanyService(mockRepo, mockProducer, logger)

			// For successful creation, add one waitgroup counter for the async event.
			if !tt.expectError {
				mockProducer.wg.Add(1)
			}

			result, err := service.CreateCompany(context.Background(), tt.input)

			// Wait for the event production to complete.
			if !tt.expectError {
				mockProducer.wg.Wait()
			}

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.expectedError != nil && !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.ID == uuid.Nil {
					t.Error("expected company ID to be set")
				}
				if len(mockProducer.producedEvents) != 1 {
					t.Error("expected creation event to be produced")
				}
			}
		})
	}
}

func TestCompanyService_GetCompany(t *testing.T) {
	testID := uuid.New()
	validCompany := &models.Company{
		ID:   testID,
		Name: "Existing Company",
	}

	tests := []struct {
		name          string
		input         uuid.UUID
		mockSetup     func(*MockRepository)
		expectError   bool
		expectedError error
	}{
		{
			name:  "successful get",
			input: testID,
			mockSetup: func(mr *MockRepository) {
				mr.getCompany = func(_ context.Context, _ uuid.UUID) (*models.Company, error) {
					return validCompany, nil
				}
			},
			expectError: false,
		},
		{
			name:  "not found",
			input: uuid.New(),
			mockSetup: func(mr *MockRepository) {
				mr.getCompany = func(_ context.Context, _ uuid.UUID) (*models.Company, error) {
					return nil, e.ErrNotFound
				}
			},
			expectError:   true,
			expectedError: e.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			mockRepo := &MockRepository{}
			tt.mockSetup(mockRepo)

			service := NewCompanyService(mockRepo, &MockProducer{}, logger)
			result, err := service.GetCompany(context.Background(), tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.ID != tt.input {
					t.Errorf("expected company ID %v, got %v", tt.input, result.ID)
				}
			}
		})
	}
}

func TestCompanyService_UpdateCompany(t *testing.T) {
	testID := uuid.New()
	validUpdate := &models.CompanyUpdate{
		ID:          testID,
		Name:        utils.Ptr("Updated Name"),
		Description: utils.Ptr("Updated Description"),
		Employees:   utils.Ptr(200),
	}

	tests := []struct {
		name          string
		input         *models.CompanyUpdate
		mockSetup     func(*MockRepository, *MockProducer)
		expectError   bool
		expectedError error
	}{
		{
			name:  "successful update",
			input: validUpdate,
			mockSetup: func(mr *MockRepository, _ *MockProducer) {
				mr.updateCompany = func(_ context.Context, _ *models.CompanyUpdate) error {
					return nil
				}
				mr.getCompany = func(_ context.Context, _ uuid.UUID) (*models.Company, error) {
					return &models.Company{ID: testID}, nil
				}
			},
			expectError: false,
		},
		{
			name: "invalid ID",
			input: &models.CompanyUpdate{
				ID: uuid.Nil,
			},
			mockSetup:     func(_ *MockRepository, _ *MockProducer) {},
			expectError:   true,
			expectedError: e.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			mockRepo := &MockRepository{}
			// Initialize a new WaitGroup for this test.
			mockProducer := &MockProducer{wg: new(sync.WaitGroup)}
			tt.mockSetup(mockRepo, mockProducer)

			service := NewCompanyService(mockRepo, mockProducer, logger)

			// For successful update, add one count to the wait group.
			if !tt.expectError {
				mockProducer.wg.Add(1)
			}

			_, err := service.UpdateCompany(context.Background(), tt.input)

			// Wait for the asynchronous event to be produced.
			if !tt.expectError {
				mockProducer.wg.Wait()
			}

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(mockProducer.producedEvents) != 1 {
					t.Error("expected update event to be produced")
				}
			}
		})
	}
}

func TestCompanyService_DeleteCompany(t *testing.T) {
	testID := uuid.New()

	tests := []struct {
		name          string
		input         uuid.UUID
		mockSetup     func(*MockRepository, *MockProducer)
		expectError   bool
		expectedError error
	}{
		{
			name:  "successful deletion",
			input: testID,
			mockSetup: func(mr *MockRepository, _ *MockProducer) {
				mr.getCompany = func(_ context.Context, _ uuid.UUID) (*models.Company, error) {
					return &models.Company{ID: testID}, nil
				}
				mr.deleteCompany = func(_ context.Context, _ uuid.UUID) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name:  "not found",
			input: testID,
			mockSetup: func(mr *MockRepository, _ *MockProducer) {
				mr.getCompany = func(_ context.Context, _ uuid.UUID) (*models.Company, error) {
					return nil, e.ErrNotFound
				}
			},
			expectError:   true,
			expectedError: e.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			mockRepo := &MockRepository{}
			// Initialize the mock producer with a WaitGroup to wait for the async event.
			mockProducer := &MockProducer{wg: new(sync.WaitGroup)}
			tt.mockSetup(mockRepo, mockProducer)

			service := NewCompanyService(mockRepo, mockProducer, logger)

			// For a successful deletion, add one counter for the async deletion event.
			if !tt.expectError {
				mockProducer.wg.Add(1)
			}

			err := service.DeleteCompany(context.Background(), tt.input)

			// Wait for the asynchronous deletion event to be produced.
			if !tt.expectError {
				mockProducer.wg.Wait()
			}

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(mockProducer.producedEvents) != 1 {
					t.Error("expected deletion event to be produced")
				}
			}
		})
	}
}
