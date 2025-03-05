package db

import (
	"context"
	"errors"
	"fmt"

	e "github.com/gartstein/xm/internal/company/errors"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewRepository(cfg *Config) (*Repository, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(&models.Company{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) CreateCompany(ctx context.Context, company *models.Company) error {
	result := r.db.WithContext(ctx).Create(company)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return e.ErrDuplicateName
		}
		return result.Error
	}
	return nil
}

func (r *Repository) GetCompany(ctx context.Context, id uuid.UUID) (*models.Company, error) {
	var company models.Company
	result := r.db.WithContext(ctx).First(&company, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, e.ErrNotFound
		}
		return nil, result.Error
	}
	return &company, nil
}

func (r *Repository) UpdateCompany(ctx context.Context, update *models.CompanyUpdate) error {
	result := r.db.WithContext(ctx).Model(&models.Company{}).
		Where("id = ?", update.ID).
		Updates(update)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return e.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Company{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return e.ErrNotFound
	}
	return nil
}

func (r *Repository) CompanyExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Company{}).
		Select("name").
		Where("name = ?", name).
		Limit(1).
		Count(&count)
	return count > 0, result.Error
}

func (r *Repository) WithTransaction(ctx context.Context, fn func(repo *Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repository{db: tx})
	})
}

func (r *Repository) Exec(ctx context.Context, query string, params ...interface{}) error {
	result := r.db.WithContext(ctx).Exec(query, params...)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *Repository) Close() error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
