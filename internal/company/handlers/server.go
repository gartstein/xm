// Package handlers provides gRPC and HTTP server implementations for
// serving the CompanyService, bridging the transport layer and business logic,
// translating between protobuf messages and domain models.
package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	pb "github.com/gartstein/xm/api/gen/definition/v1"
	"github.com/gartstein/xm/internal/company/auth"
	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// CompanyController defines the business logic interface
// that the gRPC/HTTP handlers will invoke.
type CompanyController interface {
	CreateCompany(ctx context.Context, company *models.Company) (*models.Company, error)
	GetCompany(ctx context.Context, id uuid.UUID) (*models.Company, error)
	UpdateCompany(ctx context.Context, update *models.CompanyUpdate) (*models.Company, error)
	DeleteCompany(ctx context.Context, id uuid.UUID) error
}

// Server holds references to both a gRPC server and an HTTP server.
type Server struct {
	grpcServer   *grpc.Server
	httpServer   *http.Server
	logger       *zap.Logger
	grpcEndpoint string
	httpEndpoint string
}

// NewServer constructs a Server with separate endpoints for gRPC and HTTP.
func NewServer(
	grpcPort int,
	httpPort int,
	logger *zap.Logger,
	grpcOpts ...grpc.ServerOption,
) *Server {
	return &Server{
		grpcServer:   grpc.NewServer(grpcOpts...),
		httpServer:   &http.Server{},
		logger:       logger,
		grpcEndpoint: fmt.Sprintf(":%d", grpcPort),
		httpEndpoint: fmt.Sprintf(":%d", httpPort),
	}
}

// RegisterGRPCHandler registers the gRPC handler for the CompanyService.
func (s *Server) RegisterGRPCHandler(h *CompanyHandler) {
	pb.RegisterCompanyServiceServer(s.grpcServer, h)
}

// RegisterHTTPGateway sets up the HTTP reverse-proxy (gRPC-Gateway) with the specified dial options.
func (s *Server) RegisterHTTPGateway(ctx context.Context, dialOpts []grpc.DialOption, jwtSecret string) error {
	mux := runtime.NewServeMux()
	err := pb.RegisterCompanyServiceHandlerFromEndpoint(
		ctx,
		mux,
		s.grpcEndpoint,
		dialOpts,
	)
	if err != nil {
		return err
	}

	// Wrap the mux with auth middleware
	authMiddleware := auth.HTTPMiddleware(mux, jwtSecret)

	s.httpServer.Handler = authMiddleware
	s.httpServer.Addr = s.httpEndpoint
	return nil
}

// Start runs the gRPC and HTTP servers concurrently, returning on the first error.
func (s *Server) Start() error {
	var wg sync.WaitGroup
	wg.Add(2)
	errChan := make(chan error, 2)

	// Start gRPC Server
	go func() {
		defer wg.Done()
		s.logger.Info("Starting gRPC server", zap.String("endpoint", s.grpcEndpoint))
		lis, err := net.Listen("tcp", s.grpcEndpoint)
		if err != nil {
			errChan <- fmt.Errorf("gRPC listen error: %w", err)
			return
		}
		if err := s.grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("gRPC serve error: %w", err)
		}
	}()

	// Start HTTP Server
	go func() {
		defer wg.Done()
		s.logger.Info("Starting HTTP server", zap.String("endpoint", s.httpEndpoint))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP serve error: %w", err)
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down both gRPC and HTTP servers.
func (s *Server) Stop() {
	s.logger.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.grpcServer.GracefulStop()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	s.logger.Info("Servers stopped")
}
