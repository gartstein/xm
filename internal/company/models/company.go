// Package models defines the core domain models for the Company entity.
// It includes definitions for Company, CompanyUpdate, and the CompanyType enumeration.
package models

import (
	"time"

	"github.com/google/uuid"
)

// CompanyType represents the type of a company.
type CompanyType string

const (
	// Corporations represents a corporation.
	Corporations       CompanyType = "CORPORATIONS"
	NonProfit          CompanyType = "NON_PROFIT"
	Cooperative        CompanyType = "COOPERATIVE"
	SoleProprietorship CompanyType = "SOLE_PROPRIETORSHIP"
)

// Company defines the domain model for a company entity.
type Company struct {
	// ID is the unique identifier for the company.
	ID uuid.UUID
	// Name is the company’s name.
	Name string
	// Description provides details about the company.
	Description string
	// Employees is the number of employees in the company.
	Employees int
	// Registered indicates whether the company is officially registered.
	Registered bool
	// Type specifies the category/type of the company.
	Type CompanyType
	// CreatedAt records the timestamp when the company was created.
	CreatedAt time.Time
	// UpdatedAt records the timestamp when the company was last updated.
	UpdatedAt time.Time
}

// CompanyUpdate represents the fields that can be updated for a Company.
// Pointer types are used to allow partial updates.
type CompanyUpdate struct {
	// ID is the unique identifier for the company to update.
	ID uuid.UUID
	// Name is the new name for the company.
	Name *string
	// Description is the new description.
	Description *string
	// Employees is the new employee count.
	Employees *int
	// Registered is the updated registration status.
	Registered *bool
	// Type is the updated company type.
	Type *CompanyType
}
