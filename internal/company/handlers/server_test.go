package handlers

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gartstein/xm/internal/company/models"
	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// dummyCompanyController is a simple dummy implementation of CompanyController.
type dummyCompanyController struct{}

func (d *dummyCompanyController) CreateCompany(_ context.Context, company *models.Company) (*models.Company, error) {
	// Simply return the company as created.
	return company, nil
}

func (d *dummyCompanyController) GetCompany(_ context.Context, id uuid.UUID) (*models.Company, error) {
	// Return a dummy company.
	return &models.Company{ID: id, Name: "Dummy"}, nil
}

func (d *dummyCompanyController) UpdateCompany(_ context.Context, update *models.CompanyUpdate) (*models.Company, error) {
	// Return a dummy updated company.
	return &models.Company{ID: update.ID, Name: "Updated"}, nil
}

func (d *dummyCompanyController) DeleteCompany(_ context.Context, _ uuid.UUID) error {
	// Assume deletion always succeeds.
	return nil
}

func TestServer_RegisterHTTPGateway(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Create a new Server with fixed ports.
	s := NewServer(50051, 8080, logger)
	// Call RegisterHTTPGateway with proper dial options.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.RegisterHTTPGateway(ctx, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, "secret")
	if err != nil {
		t.Fatalf("RegisterHTTPGateway failed: %v", err)
	}
	// Verify that the HTTP server is configured.
	if s.httpServer.Handler == nil {
		t.Error("expected httpServer.Handler to be set")
	}
	if s.httpServer.Addr != s.httpEndpoint {
		t.Errorf("expected httpServer.Addr %q, got %q", s.httpEndpoint, s.httpServer.Addr)
	}
}

func TestServer_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Use fixed ports so we know what address to dial.
	s := NewServer(50051, 8080, logger, grpc.Creds(insecure.NewCredentials()))

	// Create a dummy CompanyHandler using a dummy controller.
	dummyCtrl := &dummyCompanyController{}
	handler := NewCompanyHandler(dummyCtrl, logger)
	s.RegisterGRPCHandler(handler)

	// Also register the HTTP gateway.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.RegisterHTTPGateway(ctx, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, "secret"); err != nil {
		t.Fatalf("RegisterHTTPGateway failed: %v", err)
	}

	// Start the server in a separate goroutine.
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	// Give the server a moment to start.
	time.Sleep(200 * time.Millisecond)

	// Use the new recommended grpc.NewClient instead of grpc.Dial
	conn, err := grpc.NewClient(
		s.grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Errorf("failed to connect to gRPC server: %v", err)
	} else {
		conn.Close()
	}

	// Stop the server.
	s.Stop()

	// Wait for Start() to return.
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Server Start returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to stop")
	}

	// Verify that the gRPC server has stopped by attempting to listen on the same endpoint.
	lis, err := net.Listen("tcp", s.grpcEndpoint)
	if err != nil {
		t.Errorf("expected to be able to listen on %q after shutdown, but got error: %v", s.grpcEndpoint, err)
	} else {
		lis.Close()
	}
}
