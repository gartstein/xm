// Package models contains the domain models for the application,
// configured to work using GORM as the ORM.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompanyType represents the type or category of a company.
type CompanyType string

// Company represents a company entity in the database.
// It uses a UUID as the primary key and includes standard timestamp fields.
// Note: We removed the embedded gorm.Model to avoid duplicate fields since we define
// our own ID, CreatedAt, UpdatedAt, and DeletedAt.
type Company struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"size:15;uniqueIndex"`
	Description string    `gorm:"size:3000"`
	Employees   int       `gorm:"check:employees >= 0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
