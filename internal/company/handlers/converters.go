package handlers

import (
	"errors"
	"fmt"

	pb "github.com/gartstein/xm/api/gen/definition/v1"
	e "github.com/gartstein/xm/internal/company/errors"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/gartstein/xm/internal/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// protoToModel converts a protobuf Company object into an internal Company model.
func (h *CompanyHandler) protoToModel(pbCompany *pb.Company) (*models.Company, error) {
	if pbCompany == nil {
		return nil, errors.New("nil company data")
	}

	return &models.Company{
		Name:        pbCompany.GetName(),
		Description: pbCompany.GetDescription(),
		Employees:   int(pbCompany.GetEmployees()),
		Registered:  pbCompany.GetRegistered(),
		Type:        normalizeCompanyType(pbCompany.Type),
	}, nil
}

// protoToUpdate converts a protobuf Company object into an internal CompanyUpdate model,
// including a UUID for the company ID.
func (h *CompanyHandler) protoToUpdate(pbCompany *pb.Company, id uuid.UUID) (*models.CompanyUpdate, error) {
	if pbCompany == nil {
		return nil, errors.New("nil update data")
	}

	return &models.CompanyUpdate{
		ID:          id,
		Name:        &pbCompany.Name,
		Description: &pbCompany.Description,
		Employees:   utils.Ptr(int(pbCompany.Employees)),
		Registered:  &pbCompany.Registered,
		Type:        utils.Ptr(models.CompanyType(pbCompany.Type.String())),
	}, nil
}

// modelToProto converts an internal Company model into a protobuf Company object.
func (h *CompanyHandler) modelToProto(company *models.Company) *pb.Company {
	return &pb.Company{
		Id:          company.ID.String(),
		Name:        company.Name,
		Description: company.Description,
		Employees:   int32(company.Employees),
		Registered:  company.Registered,
		Type:        pb.CompanyType(pb.CompanyType_value[string(company.Type)]),
	}
}

// normalizeCompanyType converts string input to CompanyType enum
func normalizeCompanyType(companyType pb.CompanyType) models.CompanyType {
	switch companyType {
	case pb.CompanyType_CORPORATIONS:
		return models.Corporations
	case pb.CompanyType_NON_PROFIT:
		return models.NonProfit
	case pb.CompanyType_COOPERATIVE:
		return models.Cooperative
	case pb.CompanyType_SOLE_PROPRIETORSHIP:
		return models.SoleProprietorship
	default:
		return models.Corporations
	}
}

// mapServiceError maps domain or repository errors to appropriate gRPC status codes.
func (h *CompanyHandler) mapServiceError(err error) error {
	switch {
	case errors.Is(err, e.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, e.ErrDuplicateName):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, e.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		h.logger.Error("Internal server error", zap.Error(err))
		return status.Error(codes.Internal, fmt.Sprintf("internal server error: %v", err))
	}
}
