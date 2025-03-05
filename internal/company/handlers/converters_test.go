package handlers

import (
	"errors"
	"testing"

	pb "github.com/gartstein/xm/api/gen/definition/v1"
	e "github.com/gartstein/xm/internal/company/errors"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProtoToModel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := &CompanyHandler{
		logger: logger,
	}

	// Test nil input returns error.
	_, err := h.protoToModel(nil)
	if err == nil {
		t.Error("expected error for nil input, got nil")
	}

	// Test valid input.
	pbCompany := &pb.Company{
		Name:        "Test Company",
		Description: "A company for testing",
		Employees:   100,
		Registered:  true,
		Type:        pb.CompanyType_COMPANY_TYPE_CORPORATIONS,
	}

	company, err := h.protoToModel(pbCompany)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if company.Name != pbCompany.Name {
		t.Errorf("expected name %q, got %q", pbCompany.Name, company.Name)
	}
	if company.Description != pbCompany.Description {
		t.Errorf("expected description %q, got %q", pbCompany.Description, company.Description)
	}
	if company.Employees != int(pbCompany.Employees) {
		t.Errorf("expected employees %d, got %d", pbCompany.Employees, company.Employees)
	}
	if company.Registered != pbCompany.Registered {
		t.Errorf("expected registered %v, got %v", pbCompany.Registered, company.Registered)
	}
	// For Type, compare string representations.
	expectedType := models.CompanyType(pbCompany.Type.String())
	if company.Type != expectedType {
		t.Errorf("expected type %q, got %q", expectedType, company.Type)
	}
}

func TestProtoToUpdate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := &CompanyHandler{
		logger: logger,
	}

	id := uuid.New()

	// Test nil input returns error.
	_, err := h.protoToUpdate(nil, id)
	if err == nil {
		t.Error("expected error for nil input, got nil")
	}

	// Test valid conversion.
	pbCompany := &pb.Company{
		Name:        "Updated Name",
		Description: "Updated Description",
		Employees:   150,
		Registered:  false,
		Type:        pb.CompanyType_COMPANY_TYPE_NON_PROFIT,
	}

	update, err := h.protoToUpdate(pbCompany, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if update.ID != id {
		t.Errorf("expected ID %v, got %v", id, update.ID)
	}
	if update.Name == nil || *update.Name != pbCompany.Name {
		t.Errorf("expected Name %q, got %v", pbCompany.Name, update.Name)
	}
	if update.Description == nil || *update.Description != pbCompany.Description {
		t.Errorf("expected Description %q, got %v", pbCompany.Description, update.Description)
	}
	if update.Employees == nil || *update.Employees != int(pbCompany.Employees) {
		t.Errorf("expected Employees %d, got %v", pbCompany.Employees, update.Employees)
	}
	if update.Registered == nil || *update.Registered != pbCompany.Registered {
		t.Errorf("expected Registered %v, got %v", pbCompany.Registered, update.Registered)
	}
	expectedType := models.CompanyType(pbCompany.Type.String())
	if update.Type == nil || *update.Type != expectedType {
		t.Errorf("expected Type %q, got %v", expectedType, update.Type)
	}
}

func TestModelToProto(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := &CompanyHandler{
		logger: logger,
	}

	// Create a sample company.
	id := uuid.New()
	company := &models.Company{
		ID:          id,
		Name:        "Test Company",
		Description: "A description",
		Employees:   50,
		Registered:  true,
		Type:        models.NonProfit,
	}

	pbCompany := h.modelToProto(company)
	if pbCompany.Id != id.String() {
		t.Errorf("expected ID %q, got %q", id.String(), pbCompany.Id)
	}
	if pbCompany.Name != company.Name {
		t.Errorf("expected Name %q, got %q", company.Name, pbCompany.Name)
	}
	if pbCompany.Description != company.Description {
		t.Errorf("expected Description %q, got %q", company.Description, pbCompany.Description)
	}
	if pbCompany.Employees != int32(company.Employees) {
		t.Errorf("expected Employees %d, got %d", company.Employees, pbCompany.Employees)
	}
	if pbCompany.Registered != company.Registered {
		t.Errorf("expected Registered %v, got %v", company.Registered, pbCompany.Registered)
	}
	// Check type conversion (using string representation).
	expectedType := pb.CompanyType(pb.CompanyType_value[string(company.Type)])
	if pbCompany.Type != expectedType {
		t.Errorf("expected Type %v, got %v", expectedType, pbCompany.Type)
	}
}

func TestMapServiceError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := &CompanyHandler{
		logger: logger,
	}

	// Test mapping for not found error.
	errNotFound := e.ErrNotFound
	mappedErr := h.mapServiceError(errNotFound)
	if status.Code(mappedErr) != codes.NotFound {
		t.Errorf("expected code %v, got %v", codes.NotFound, status.Code(mappedErr))
	}

	// Test mapping for duplicate name error.
	errDup := e.ErrDuplicateName
	mappedErr = h.mapServiceError(errDup)
	if status.Code(mappedErr) != codes.AlreadyExists {
		t.Errorf("expected code %v, got %v", codes.AlreadyExists, status.Code(mappedErr))
	}

	// Test mapping for invalid input error.
	errInvalid := e.ErrInvalidInput
	mappedErr = h.mapServiceError(errInvalid)
	if status.Code(mappedErr) != codes.InvalidArgument {
		t.Errorf("expected code %v, got %v", codes.InvalidArgument, status.Code(mappedErr))
	}

	// Test mapping for an unknown error.
	genericErr := errors.New("some error")
	mappedErr = h.mapServiceError(genericErr)
	if status.Code(mappedErr) != codes.Internal {
		t.Errorf("expected code %v, got %v", codes.Internal, status.Code(mappedErr))
	}
}
