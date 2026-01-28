package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"

	"docapi/docs"
	"docapi/internal/config"
	"docapi/internal/database"
	handlers "docapi/internal/http/handler"
	"docapi/internal/http/middleware"
	"docapi/internal/repository/postgres"
	"docapi/internal/service"
	"docapi/internal/storage"
)

// @title Document API
// @version 1.0
// @host localhost:8080
// @BasePath /
func main() {
	// Load configuration from environment variables (.env auto-loaded if present)
	cfg := config.Load()

	// Update Swagger info with dynamic host from config
	docs.SwaggerInfo.Host = cfg.AppHost

	// Initialize PostgreSQL connection (with pooling via database/sql)
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize reusable S3-compatible object storage client (MinIO-supported)
	objStore, err := storage.NewMinIO(cfg.MinIO)
	if err != nil {
		log.Fatalf("failed to initialize object storage: %v", err)
	}

	// Initialize repositories and services
	docRepo := postgres.NewDocumentPostgres(db)
	docSvc := service.NewDocumentService(objStore, docRepo)

	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler(),
	})

	// Register global middleware
	// RequestID middleware adds/propagates X-Request-ID and stores it in context
	app.Use(middleware.RequestID())
	// JSON Logger middleware for structured request logs
	app.Use(middleware.Logger())

	// Register HTTP routes with injected service
	handlers.RegisterRoutes(app, db, docSvc)

	addr := ":" + cfg.Port

	if err := app.Listen(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
