package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type migrationStep struct {
	Name string
	SQL  string
}

var steps = []migrationStep{
	{
		Name: "create_extension_uuid_ossp",
		SQL:  `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,
	},
	{
		Name: "create_table_documents",
		SQL: `CREATE TABLE IF NOT EXISTS documents (
  id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  filename     TEXT        NOT NULL,
  storage_path TEXT        NOT NULL UNIQUE,
  size         BIGINT      NOT NULL CHECK (size >= 0),
  content_type TEXT        NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);`,
	},
	{
		Name: "create_index_documents_filename",
		SQL:  `CREATE INDEX IF NOT EXISTS idx_documents_filename ON documents (filename);`,
	},
	{
		Name: "create_index_documents_content_type",
		SQL:  `CREATE INDEX IF NOT EXISTS idx_documents_content_type ON documents (content_type);`,
	},
	{
		Name: "create_index_documents_created_at",
		SQL:  `CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents (created_at);`,
	},
}

// EnsureMigrated checks if the 'documents' table exists and runs migrations if it doesn't.
func EnsureMigrated(ctx context.Context, db *sql.DB, loc *time.Location, dbHost string) error {
	start := time.Now()

	logJSON(loc, map[string]any{
		"component": "database",
		"event":     "db_migration_check",
		"status":    "starting",
		"db_host":   dbHost,
	})

	var exists bool
	query := "SELECT to_regclass('public.documents') IS NOT NULL"
	err := db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {
		logJSON(loc, map[string]any{
			"component":     "database",
			"event":         "db_migration_failed",
			"status":        "error",
			"error_message": fmt.Sprintf("failed to check sentinel table: %v", err),
			"db_host":       dbHost,
			"duration_ms":   time.Since(start).Milliseconds(),
		})
		return fmt.Errorf("failed to check sentinel table: %w", err)
	}

	if exists {
		logJSON(loc, map[string]any{
			"component":   "database",
			"event":       "db_migration_skip",
			"status":      "success",
			"msg":         "schema already exists, skipping migration",
			"db_host":     dbHost,
			"duration_ms": time.Since(start).Milliseconds(),
		})
		return nil
	}

	logJSON(loc, map[string]any{
		"component": "database",
		"event":     "db_migration_start",
		"status":    "in_progress",
		"db_host":   dbHost,
	})

	for _, step := range steps {
		stepStart := time.Now()
		_, err := db.ExecContext(ctx, step.SQL)
		if err != nil {
			logJSON(loc, map[string]any{
				"component":        "database",
				"event":            "db_migration_failed",
				"status":           "error",
				"migration_step":   step.Name,
				"error_message":    err.Error(),
				"db_host":          dbHost,
				"duration_ms":      time.Since(start).Milliseconds(),
				"step_duration_ms": time.Since(stepStart).Milliseconds(),
			})
			return fmt.Errorf("migration step %s failed: %w", step.Name, err)
		}

		logJSON(loc, map[string]any{
			"component":        "database",
			"event":            "db_migration_step",
			"status":           "success",
			"migration_step":   step.Name,
			"db_host":          dbHost,
			"step_duration_ms": time.Since(stepStart).Milliseconds(),
		})
	}

	logJSON(loc, map[string]any{
		"component":   "database",
		"event":       "db_migration_success",
		"status":      "success",
		"db_host":     dbHost,
		"duration_ms": time.Since(start).Milliseconds(),
	})

	return nil
}

func logJSON(loc *time.Location, data map[string]any) {
	data["ts"] = time.Now().In(loc).Format(time.RFC3339Nano)
	if _, ok := data["level"]; !ok {
		if data["status"] == "error" {
			data["level"] = "error"
		} else {
			data["level"] = "info"
		}
	}

	b, err := json.Marshal(data)
	if err != nil {
		log.Printf("failed to marshal migration log: %v", err)
		return
	}
	log.SetFlags(0)
	log.Println(string(b))
}
