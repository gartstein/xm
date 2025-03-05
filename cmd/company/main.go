package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gartstein/xm/internal/company/auth"
	"github.com/gartstein/xm/internal/company/controller"
	gorm "github.com/gartstein/xm/internal/company/db"
	"github.com/gartstein/xm/internal/company/events"
	"github.com/gartstein/xm/internal/company/handlers"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

// Config struct for YAML configuration
type Config struct {
	GRPCPort     int      `yaml:"GRPC_PORT"`
	HTTPPort     int      `yaml:"HTTP_PORT"`
	DBHost       string   `yaml:"DB_HOST"`
	DBPort       int      `yaml:"DB_PORT"`
	DBUser       string   `yaml:"DB_USER"`
	DBPassword   string   `yaml:"DB_PASSWORD"`
	DBName       string   `yaml:"DB_NAME"`
	DBSSLMode    string   `yaml:"DB_SSLMODE"`
	KafkaBrokers []string `yaml:"KAFKA_BROKERS"`
	JWTSecret    string   `yaml:"JWT_SECRET"`
	Topic        string   `yaml:"TOPIC"`
}

func main() {
	logger := initLogger()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			logger.Error("failed to sync logger", zap.Error(err))
		}
	}(logger)

	cfg, err := loadConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	dbConf := initDatabase(cfg)
	repo, err := gorm.NewRepository(dbConf)
	if err != nil {
		log.Fatal("failed to initialize database", err)
	}

	producer, err := events.NewProducer(cfg.KafkaBrokers, logger, cfg.Topic)
	if err != nil {
		log.Fatal("failed to initialize Kafka producer", err)
	}
	defer producer.Close()

	companySvc := controller.NewCompanyService(repo, producer, logger)

	// Create handlers
	companyHandler := handlers.NewCompanyHandler(companySvc, logger)

	// Initialize auth interceptor
	authInterceptor := auth.NewAuthInterceptor(cfg.JWTSecret)
	// Create server
	server := handlers.NewServer(cfg.GRPCPort, cfg.HTTPPort, logger, grpc.UnaryInterceptor(authInterceptor.Unary()))
	server.RegisterGRPCHandler(companyHandler)

	// Register HTTP gateway
	if err := server.RegisterHTTPGateway(
		context.Background(),
		[]grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		cfg.JWTSecret); err != nil {
		logger.Fatal("Failed to register HTTP gateway", zap.Error(err))
	}
	// Start servers
	if err := server.Start(); err != nil {
		logger.Fatal("Failed to start servers", zap.Error(err))
	}

	waitForShutdown(server, logger)
}

// initLogger initializes a Zap production logger.
func initLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}

// loadConfig loads configuration. Use real config tooling (e.g. Viper) in production.
// TODO: some settings to env
func loadConfig() (*Config, error) {
	configPath := filepath.Join("internal", "company", "config", "config.yaml")
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// initDatabase initializes the database connection.
func initDatabase(cfg *Config) *gorm.Config {
	return &gorm.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}
}

// waitForShutdown blocks until an interrupt or SIGTERM is received, then shuts down servers.
func waitForShutdown(server *handlers.Server, logger *zap.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	server.Stop()
	logger.Info("Servers stopped properly")
}
