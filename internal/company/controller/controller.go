// Package controller implements the core business logic (service layer)
// for managing Company entities, orchestrating repository operations
// and sending relevant events.
package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/gartstein/xm/internal/company/db"
	e "github.com/gartstein/xm/internal/company/errors"
	"github.com/gartstein/xm/internal/company/events"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EventProducer interface {
	Produce(eventType events.EventType, company *models.Company)
}

// Repository defines the storage interface for Company objects.
type Repository interface {
	CreateCompany(ctx context.Context, company *models.Company) error
	GetCompany(ctx context.Context, id uuid.UUID) (*models.Company, error)
	UpdateCompany(ctx context.Context, company *models.CompanyUpdate) error
	DeleteCompany(ctx context.Context, id uuid.UUID) error
	CompanyExistsByName(ctx context.Context, name string) (bool, error)
	WithTransaction(ctx context.Context, fn func(repo *db.Repository) error) error
	Close() error
}

// CompanyService provides methods to manage companies via repository
// operations and event production.
type CompanyService struct {
	repo     Repository
	producer EventProducer
	logger   *zap.Logger
}

// NewCompanyService constructs a CompanyService with a repository,
// an event producer, and a logger.
func NewCompanyService(repo Repository, producer EventProducer, logger *zap.Logger) *CompanyService {
	return &CompanyService{
		repo:     repo,
		producer: producer,
		logger:   logger.Named("company_service"),
	}
}

// CreateCompany adds a new Company after validating input data,
// ensures uniqueness by checking the name, and triggers an event.
func (s *CompanyService) CreateCompany(ctx context.Context, company *models.Company) (*models.Company, error) {
	if company.Name == "" || len(company.Name) > 15 {
		return nil, fmt.Errorf("%w: invalid name", e.ErrInvalidInput)
	}
	if company.Description != "" && len(company.Description) > 3000 {
		return nil, fmt.Errorf("%w: description too long", e.ErrInvalidInput)
	}

	exists, err := s.repo.CompanyExistsByName(ctx, company.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check name existence: %w", err)
	}
	if exists {
		return nil, e.ErrDuplicateName
	}

	company.ID = uuid.New()
	if err := s.repo.CreateCompany(ctx, company); err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}
	go func() {
		s.producer.Produce(events.CompanyCreated, company)
	}()
	return company, nil
}

// GetCompany retrieves a Company by ID, returning an error if not found.
func (s *CompanyService) GetCompany(ctx context.Context, id uuid.UUID) (*models.Company, error) {
	company, err := s.repo.GetCompany(ctx, id)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	return company, nil
}

// UpdateCompany modifies the specified Company fields,
// then fetches the updated version for returning and event production.
func (s *CompanyService) UpdateCompany(ctx context.Context, update *models.CompanyUpdate) (*models.Company, error) {
	if update.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid company ID", e.ErrInvalidInput)
	}

	err := s.repo.UpdateCompany(ctx, update)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to update company: %w", err)
	}

	updated, err := s.repo.GetCompany(context.Background(), update.ID)
	if err != nil {
		s.logger.Error("Failed to get company for event",
			zap.Error(err),
			zap.String("company_id", update.ID.String()),
		)
		return nil, err
	}
	go func() {
		s.producer.Produce(events.CompanyUpdated, updated)
	}()
	return updated, nil
}

// DeleteCompany removes a Company by ID and fires a deletion event.
func (s *CompanyService) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	company, err := s.repo.GetCompany(ctx, id)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to get company for deletion: %w", err)
	}

	if err := s.repo.DeleteCompany(ctx, id); err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	go func() {
		s.producer.Produce(events.CompanyDeleted, company)
	}()

	return nil
}
