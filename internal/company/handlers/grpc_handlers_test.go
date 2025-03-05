package handlers

import (
	"context"
	"errors"
	"testing"

	pb "github.com/gartstein/xm/api/gen/definition/v1"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockCompanyController is a simple mock implementation of CompanyController.
type mockCompanyController struct {
	createCompanyFunc func(ctx context.Context, company *models.Company) (*models.Company, error)
	updateCompanyFunc func(ctx context.Context, update *models.CompanyUpdate) (*models.Company, error)
	deleteCompanyFunc func(ctx context.Context, id uuid.UUID) error
	getCompanyFunc    func(ctx context.Context, id uuid.UUID) (*models.Company, error)
}

func (m *mockCompanyController) CreateCompany(ctx context.Context, company *models.Company) (*models.Company, error) {
	return m.createCompanyFunc(ctx, company)
}

func (m *mockCompanyController) UpdateCompany(ctx context.Context, update *models.CompanyUpdate) (*models.Company, error) {
	return m.updateCompanyFunc(ctx, update)
}

func (m *mockCompanyController) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return m.deleteCompanyFunc(ctx, id)
}

func (m *mockCompanyController) GetCompany(ctx context.Context, id uuid.UUID) (*models.Company, error) {
	return m.getCompanyFunc(ctx, id)
}

// Test for CreateCompany.
func TestCompanyHandler_CreateCompany(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("NilCompany", func(t *testing.T) {
		mockCtrl := &mockCompanyController{}
		handler := NewCompanyHandler(mockCtrl, logger)
		req := &pb.CreateCompanyRequest{Company: nil}
		_, err := handler.CreateCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected error for nil company, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected code %v, got %v", codes.InvalidArgument, st.Code())
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		expectedErr := errors.New("service error")
		mockCtrl := &mockCompanyController{
			createCompanyFunc: func(_ context.Context, _ *models.Company) (*models.Company, error) {
				return nil, expectedErr
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		pbCompany := &pb.Company{
			Name:        "Test Co",
			Description: "Desc",
			Employees:   10,
			Registered:  true,
			Type:        pb.CompanyType_COMPANY_TYPE_CORPORATIONS,
		}
		req := &pb.CreateCompanyRequest{Company: pbCompany}
		_, err := handler.CreateCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected service error, got nil")
		}
		st, _ := status.FromError(err)
		// mapServiceError maps unknown errors to Internal.
		if st.Code() != codes.Internal {
			t.Errorf("expected code %v, got %v", codes.Internal, st.Code())
		}
	})

	t.Run("Success", func(t *testing.T) {
		testID := uuid.New()
		mockCtrl := &mockCompanyController{
			createCompanyFunc: func(_ context.Context, company *models.Company) (*models.Company, error) {
				company.ID = testID
				return company, nil
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		pbCompany := &pb.Company{
			Name:        "Test Co",
			Description: "Desc",
			Employees:   10,
			Registered:  true,
			Type:        pb.CompanyType_COMPANY_TYPE_CORPORATIONS,
		}
		req := &pb.CreateCompanyRequest{Company: pbCompany}
		resp, err := handler.CreateCompany(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Company.Id != testID.String() {
			t.Errorf("expected company ID %q, got %q", testID.String(), resp.Company.Id)
		}
	})
}

// Test for PatchCompany.
func TestCompanyHandler_PatchCompany(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("InvalidID", func(t *testing.T) {
		mockCtrl := &mockCompanyController{}
		handler := NewCompanyHandler(mockCtrl, logger)
		req := &pb.UpdateCompanyRequest{
			Id:      "invalid-uuid",
			Company: &pb.Company{Name: "Update"},
		}
		_, err := handler.PatchCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected error for invalid uuid, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected code %v, got %v", codes.InvalidArgument, st.Code())
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		expectedErr := errors.New("update error")
		mockCtrl := &mockCompanyController{
			updateCompanyFunc: func(_ context.Context, _ *models.CompanyUpdate) (*models.Company, error) {
				return nil, expectedErr
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		testID := uuid.New().String()
		pbCompany := &pb.Company{
			Name:        "Updated Name",
			Description: "Updated Desc",
			Employees:   20,
			Registered:  false,
			Type:        pb.CompanyType_COMPANY_TYPE_NON_PROFIT,
		}
		req := &pb.UpdateCompanyRequest{
			Id:      testID,
			Company: pbCompany,
		}
		_, err := handler.PatchCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected service error, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.Internal {
			t.Errorf("expected error code %v, got %v", codes.Internal, st.Code())
		}
	})

	t.Run("Success", func(t *testing.T) {
		testID := uuid.New()
		mockCtrl := &mockCompanyController{
			updateCompanyFunc: func(_ context.Context, _ *models.CompanyUpdate) (*models.Company, error) {
				return &models.Company{
					ID:          testID,
					Name:        "Updated Name",
					Description: "Updated Desc",
					Employees:   20,
					Registered:  false,
					Type:        models.NonProfit,
				}, nil
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		pbCompany := &pb.Company{
			Name:        "Updated Name",
			Description: "Updated Desc",
			Employees:   20,
			Registered:  false,
			Type:        pb.CompanyType_COMPANY_TYPE_NON_PROFIT,
		}
		req := &pb.UpdateCompanyRequest{
			Id:      testID.String(),
			Company: pbCompany,
		}
		resp, err := handler.PatchCompany(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Company.Id != testID.String() {
			t.Errorf("expected company ID %q, got %q", testID.String(), resp.Company.Id)
		}
	})
}

// Test for DeleteCompany.
func TestCompanyHandler_DeleteCompany(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("InvalidID", func(t *testing.T) {
		mockCtrl := &mockCompanyController{}
		handler := NewCompanyHandler(mockCtrl, logger)
		req := &pb.DeleteCompanyRequest{Id: "invalid-uuid"}
		_, err := handler.DeleteCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected error for invalid uuid, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected code %v, got %v", codes.InvalidArgument, st.Code())
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		expectedErr := errors.New("delete error")
		mockCtrl := &mockCompanyController{
			deleteCompanyFunc: func(_ context.Context, _ uuid.UUID) error {
				return expectedErr
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		testID := uuid.New().String()
		req := &pb.DeleteCompanyRequest{Id: testID}
		_, err := handler.DeleteCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.Internal {
			t.Errorf("expected error code %v, got %v", codes.Internal, st.Code())
		}
	})

	t.Run("Success", func(t *testing.T) {
		mockCtrl := &mockCompanyController{
			deleteCompanyFunc: func(_ context.Context, _ uuid.UUID) error {
				return nil
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		testID := uuid.New().String()
		req := &pb.DeleteCompanyRequest{Id: testID}
		resp, err := handler.DeleteCompany(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Error("expected non-nil response")
		}
	})
}

// Test for GetCompany.
func TestCompanyHandler_GetCompany(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("InvalidID", func(t *testing.T) {
		mockCtrl := &mockCompanyController{}
		handler := NewCompanyHandler(mockCtrl, logger)
		req := &pb.GetCompanyRequest{Id: "invalid-uuid"}
		_, err := handler.GetCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected error for invalid uuid, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected error code %v, got %v", codes.InvalidArgument, st.Code())
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		expectedErr := errors.New("get error")
		mockCtrl := &mockCompanyController{
			getCompanyFunc: func(_ context.Context, _ uuid.UUID) (*models.Company, error) {
				return nil, expectedErr
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		testID := uuid.New().String()
		req := &pb.GetCompanyRequest{Id: testID}
		_, err := handler.GetCompany(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.Internal {
			t.Errorf("expected error code %v, got %v", codes.Internal, st.Code())
		}
	})

	t.Run("Success", func(t *testing.T) {
		testID := uuid.New()
		mockCtrl := &mockCompanyController{
			getCompanyFunc: func(_ context.Context, _ uuid.UUID) (*models.Company, error) {
				return &models.Company{
					ID:          testID,
					Name:        "Test Co",
					Description: "Desc",
					Employees:   30,
					Registered:  true,
					Type:        models.NonProfit,
				}, nil
			},
		}
		handler := NewCompanyHandler(mockCtrl, logger)
		req := &pb.GetCompanyRequest{Id: testID.String()}
		resp, err := handler.GetCompany(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Company.Id != testID.String() {
			t.Errorf("expected company ID %q, got %q", testID.String(), resp.Company.Id)
		}
		if resp.Company.Name != "Test Co" {
			t.Errorf("expected company name %q, got %q", "Test Co", resp.Company.Name)
		}
	})
}
