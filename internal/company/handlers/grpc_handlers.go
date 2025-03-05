package handlers

import (
	"context"
	"fmt"

	pb "github.com/gartstein/xm/api/gen/definition/v1"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CompanyHandler provides gRPC methods for Company operations,
// mapping requests to a CompanyController interface.
type CompanyHandler struct {
	pb.UnimplementedCompanyServiceServer
	service CompanyController
	logger  *zap.Logger
}

// NewCompanyHandler constructs a new CompanyHandler with the given service and logger.
func NewCompanyHandler(service CompanyController, logger *zap.Logger) *CompanyHandler {
	return &CompanyHandler{
		service: service,
		logger:  logger.Named("grpc_handler"),
	}
}

// CreateCompany processes a CreateCompanyRequest, creating a new Company in the system.
func (h *CompanyHandler) CreateCompany(ctx context.Context, req *pb.CreateCompanyRequest) (*pb.CreateCompanyResponse, error) {
	reqCompany := req.GetCompany()
	if reqCompany == nil {
		return nil, status.Error(codes.InvalidArgument, "company data required")
	}

	company, err := h.protoToModel(reqCompany)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	created, err := h.service.CreateCompany(ctx, company)
	if err != nil {
		h.logger.Error("Create company failed", zap.Error(err))
		return nil, h.mapServiceError(err)
	}
	fmt.Println("CREATEd COMPANY", h.modelToProto(created))
	return &pb.CreateCompanyResponse{
		Company: h.modelToProto(created),
	}, nil
}

// UpdateCompany processes updates to an existing Company based on the provided ID and update data.
func (h *CompanyHandler) UpdateCompany(ctx context.Context, req *pb.UpdateCompanyRequest) (*pb.UpdateCompanyResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company ID")
	}

	update, err := h.protoToUpdate(req.GetCompany(), id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	updated, err := h.service.UpdateCompany(ctx, update)
	if err != nil {
		return nil, h.mapServiceError(err)
	}

	return &pb.UpdateCompanyResponse{
		Company: h.modelToProto(updated),
	}, nil
}

// DeleteCompany removes a Company given its ID.
func (h *CompanyHandler) DeleteCompany(ctx context.Context, req *pb.DeleteCompanyRequest) (*pb.DeleteCompanyResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company ID")
	}

	if err := h.service.DeleteCompany(ctx, id); err != nil {
		return nil, h.mapServiceError(err)
	}

	return &pb.DeleteCompanyResponse{}, nil
}

// GetCompany fetches a Company by ID, returning an error if not found.
func (h *CompanyHandler) GetCompany(ctx context.Context, req *pb.GetCompanyRequest) (*pb.GetCompanyResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company ID")
	}

	company, err := h.service.GetCompany(ctx, id)
	if err != nil {
		return nil, h.mapServiceError(err)
	}

	return &pb.GetCompanyResponse{
		Company: h.modelToProto(company),
	}, nil
}
