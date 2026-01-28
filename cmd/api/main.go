package main

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
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
// @BasePath /
func main() {
	// Load configuration from environment variables (.env auto-loaded if present)
	cfg := config.Load()

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

	// Swagger UI with dynamic host and scheme
	app.Get("/swagger/*", func(c *fiber.Ctx) error {
		scheme := c.Protocol()
		if proto := c.Get("X-Forwarded-Proto"); proto != "" {
			scheme = strings.Split(proto, ",")[0]
		}

		docs.SwaggerInfo.Host = c.Get("Host")
		docs.SwaggerInfo.Schemes = []string{scheme}

		return swagger.HandlerDefault(c)
	})

	addr := ":" + cfg.Port

	if err := app.Listen(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
