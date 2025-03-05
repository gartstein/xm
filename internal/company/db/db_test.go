package db

import (
	"context"
	"testing"

	e "github.com/gartstein/xm/internal/company/errors"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/gartstein/xm/internal/pkg/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB initializes an in-memory SQLite database for testing.
func SetupTestDB(t *testing.T) *Repository {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to open test database")

	err = db.AutoMigrate(&models.Company{})
	require.NoError(t, err, "failed to migrate test database")

	return &Repository{db: db}
}

// TestCreateCompany tests the creation of a company record.
func TestCreateCompany(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	company := &models.Company{
		ID:   uuid.New(),
		Name: "Test Company",
	}

	err := repo.CreateCompany(ctx, company)
	assert.NoError(t, err, "CreateCompany should not return an error")

	// Verify the company was created
	retrieved, err := repo.GetCompany(ctx, company.ID)
	assert.NoError(t, err, "GetCompany should retrieve the created company")
	assert.Equal(t, company.Name, retrieved.Name, "Company name should match")
}

// TestGetCompany ensures retrieval works correctly.
func TestGetCompany(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	company := &models.Company{
		ID:   uuid.New(),
		Name: "Existing Company",
	}

	require.NoError(t, repo.CreateCompany(ctx, company), "CreateCompany should succeed")

	// Fetch the company
	result, err := repo.GetCompany(ctx, company.ID)
	assert.NoError(t, err, "GetCompany should succeed")
	assert.Equal(t, company.ID, result.ID, "Company ID should match")
}

// TestGetCompanyNotFound verifies error handling when the company does not exist.
func TestGetCompanyNotFound(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetCompany(ctx, uuid.New())
	assert.ErrorIs(t, err, e.ErrNotFound, "GetCompany should return ErrNotFound for non-existent company")
}

// TestUpdateCompany checks if updating a company's name works.
func TestUpdateCompany(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	company := &models.Company{
		ID:   uuid.New(),
		Name: "Old Name",
	}
	require.NoError(t, repo.CreateCompany(ctx, company), "CreateCompany should succeed")

	update := &models.CompanyUpdate{
		ID:   company.ID,
		Name: utils.Ptr("New Name"),
	}

	err := repo.UpdateCompany(ctx, update)
	assert.NoError(t, err, "UpdateCompany should not return an error")

	// Verify update
	updated, err := repo.GetCompany(ctx, company.ID)
	assert.NoError(t, err, "GetCompany should succeed")
	assert.Equal(t, "New Name", updated.Name, "Company name should be updated")
}

// TestUpdateCompanyNotFound tests updating a non-existing company.
func TestUpdateCompanyNotFound(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	update := &models.CompanyUpdate{
		ID:   uuid.New(),
		Name: utils.Ptr("Non-existent"),
	}

	err := repo.UpdateCompany(ctx, update)
	assert.ErrorIs(t, err, e.ErrNotFound, "UpdateCompany should return ErrNotFound for missing company")
}

// TestDeleteCompany ensures companies are deleted correctly.
func TestDeleteCompany(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	company := &models.Company{
		ID:   uuid.New(),
		Name: "To Be Deleted",
	}
	require.NoError(t, repo.CreateCompany(ctx, company), "CreateCompany should succeed")

	err := repo.DeleteCompany(ctx, company.ID)
	assert.NoError(t, err, "DeleteCompany should not return an error")

	// Ensure deletion
	_, err = repo.GetCompany(ctx, company.ID)
	assert.ErrorIs(t, err, e.ErrNotFound, "Deleted company should not be found")
}

// TestDeleteCompanyNotFound checks behavior when trying to delete a non-existent company.
func TestDeleteCompanyNotFound(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	err := repo.DeleteCompany(ctx, uuid.New())
	assert.ErrorIs(t, err, e.ErrNotFound, "DeleteCompany should return ErrNotFound for missing company")
}

// TestCompanyExistsByName verifies if the company existence check works.
func TestCompanyExistsByName(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	exists, err := repo.CompanyExistsByName(ctx, "Non-existent")
	assert.NoError(t, err, "CompanyExistsByName should not return an error")
	assert.False(t, exists, "Non-existent company should return false")

	company := &models.Company{
		ID:   uuid.New(),
		Name: "Existing Company",
	}
	require.NoError(t, repo.CreateCompany(ctx, company), "CreateCompany should succeed")

	exists, err = repo.CompanyExistsByName(ctx, company.Name)
	assert.NoError(t, err, "CompanyExistsByName should not return an error")
	assert.True(t, exists, "Existing company should return true")
}

// TestWithTransaction ensures transactions work correctly.
func TestWithTransaction(t *testing.T) {
	repo := SetupTestDB(t)
	ctx := context.Background()

	err := repo.WithTransaction(ctx, func(txRepo *Repository) error {
		company := &models.Company{
			ID:   uuid.New(),
			Name: "Transactional Company",
		}
		return txRepo.CreateCompany(ctx, company)
	})

	assert.NoError(t, err, "WithTransaction should execute successfully")

	// Verify the transaction was committed
	exists, _ := repo.CompanyExistsByName(ctx, "Transactional Company")
	assert.True(t, exists, "Company should exist after transaction")
}
