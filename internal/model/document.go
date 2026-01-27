package model

import "time"

// Document represents a stored file in the system.
// This is a pure domain model with no database-specific dependencies or tags.
// It can be used across layers (HTTP, service, storage) without coupling to persistence.
type Document struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	StoragePath string    `json:"storage_path"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}
