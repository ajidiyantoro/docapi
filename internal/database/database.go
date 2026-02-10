package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"docapi/internal/config"
)

var sqlOpen = sql.Open

// BuildPostgresDSN constructs a DSN for PostgreSQL using standard components.
// Example: postgres://user:pass@host:port/dbname?sslmode=disable
func BuildPostgresDSN(c config.DatabaseConfig) (string, error) {
	if c.Host == "" || c.Port == "" || c.User == "" || c.Name == "" {
		return "", fmt.Errorf("invalid database config: host, port, user, and name are required")
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
		Path:   c.Name,
	}
	if c.Password != "" {
		u.User = url.UserPassword(c.User, c.Password)
	} else {
		u.User = url.User(c.User)
	}

	q := u.Query()
	if c.SSLMode != "" {
		q.Set("sslmode", c.SSLMode)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// NewPostgres opens a database/sql connection using the pgx stdlib driver and applies pooling settings.
func NewPostgres(c config.DatabaseConfig) (*sql.DB, error) {
	dsn, err := BuildPostgresDSN(c)
	if err != nil {
		return nil, err
	}

	// Register the otelsql driver wrapper
	driverName, err := otelsql.Register("pgx",
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSQLCommenter(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register otelsql: %w", err)
	}

	db, err := sqlOpen(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}

	// Apply connection pool settings if provided
	if c.MaxOpenConns > 0 {
		db.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		db.SetMaxIdleConns(c.MaxIdleConns)
	}
	if c.ConnMaxLifetimeSec > 0 {
		db.SetConnMaxLifetime(time.Duration(c.ConnMaxLifetimeSec) * time.Second)
	}

	// Verify connectivity with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return db, nil
}
