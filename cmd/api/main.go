package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	_ "github.com/joho/godotenv/autoload"
	"github.com/prometheus/client_golang/prometheus"

	"docapi/docs"
	"docapi/internal/config"
	"docapi/internal/database"
	"docapi/internal/database/migration"
	handlers "docapi/internal/http/handler"
	"docapi/internal/http/middleware"
	"docapi/internal/otel"
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

	// Initialize OpenTelemetry tracing
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otelShutdown, err := otel.Init(ctx, cfg.Location)
	if err != nil {
		log.Fatalf("failed to initialize tracing: %v", err)
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown tracing: %v", err)
		}
	}()

	// Initialize PostgreSQL connection (with pooling via database/sql)
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure database schema is up to date
	if err := migration.EnsureMigrated(ctx, db, cfg.Location, cfg.Database.Host); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Initialize reusable S3-compatible object storage client (MinIO-supported)
	objStore, err := storage.NewMinIO(cfg.MinIO)
	if err != nil {
		log.Fatalf("failed to initialize object storage: %v", err)
	}

	// Initialize repositories and services
	docRepo := postgres.NewDocumentPostgres(db)
	docSvc := service.NewDocumentService(objStore, docRepo)

	app := fiber.New(fiber.Config{
		ErrorHandler:          handlers.ErrorHandler(),
		DisableStartupMessage: true,
	})

	// Initialize Prometheus middleware
	promMiddleware, err := middleware.NewPrometheusMiddleware(prometheus.DefaultRegisterer)
	if err != nil {
		log.Fatalf("failed to initialize prometheus middleware: %v", err)
	}

	// Register global middleware
	// Tracing middleware should be first to capture the whole request
	app.Use(otelfiber.Middleware())
	// RequestID middleware adds/propagates X-Request-ID and stores it in context
	app.Use(middleware.RequestID())
	// JSON Logger middleware for structured request logs
	app.Use(middleware.Logger(cfg.Location))
	// Prometheus middleware to track request count
	app.Use(promMiddleware.Handler())

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

	// Startup log in JSON format
	startupLog := map[string]string{
		"ts":    time.Now().In(cfg.Location).Format(time.RFC3339Nano),
		"level": "info",
		"msg":   "server_starting",
		"addr":  addr,
		"port":  cfg.Port,
	}
	if b, err := json.Marshal(startupLog); err == nil {
		log.SetFlags(0)
		log.Println(string(b))
	}

	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
