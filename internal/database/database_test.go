package database

import (
	"database/sql"
	"errors"
	"testing"

	"docapi/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPostgresDSN(t *testing.T) {
	tests := []struct {
		name    string
		config  config.DatabaseConfig
		want    string
		wantErr bool
	}{
		{
			name: "valid config with password and sslmode",
			config: config.DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "user",
				Password: "pass",
				Name:     "dbname",
				SSLMode:  "disable",
			},
			want:    "postgres://user:pass@localhost:5432/dbname?sslmode=disable",
			wantErr: false,
		},
		{
			name: "valid config without password",
			config: config.DatabaseConfig{
				Host:    "localhost",
				Port:    "5432",
				User:    "user",
				Name:    "dbname",
				SSLMode: "require",
			},
			want:    "postgres://user@localhost:5432/dbname?sslmode=require",
			wantErr: false,
		},
		{
			name: "valid config without password and without sslmode",
			config: config.DatabaseConfig{
				Host: "localhost",
				Port: "5432",
				User: "user",
				Name: "dbname",
			},
			want:    "postgres://user@localhost:5432/dbname",
			wantErr: false,
		},
		{
			name: "invalid config missing host",
			config: config.DatabaseConfig{
				Port: "5432",
				User: "user",
				Name: "dbname",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid config missing port",
			config: config.DatabaseConfig{
				Host: "localhost",
				User: "user",
				Name: "dbname",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid config missing user",
			config: config.DatabaseConfig{
				Host: "localhost",
				Port: "5432",
				Name: "dbname",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid config missing name",
			config: config.DatabaseConfig{
				Host: "localhost",
				Port: "5432",
				User: "user",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildPostgresDSN(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewPostgres(t *testing.T) {
	conf := config.DatabaseConfig{
		Host:               "localhost",
		Port:               "5432",
		User:               "user",
		Password:           "pass",
		Name:               "dbname",
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetimeSec: 300,
	}

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		// Mock sqlOpen to return the mock db
		origSqlOpen := sqlOpen
		sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil
		}
		defer func() { sqlOpen = origSqlOpen }()

		mock.ExpectPing()

		gotDB, err := NewPostgres(conf)
		assert.NoError(t, err)
		assert.NotNil(t, gotDB)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sqlOpen error", func(t *testing.T) {
		// Mock sqlOpen to return error
		origSqlOpen := sqlOpen
		sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, errors.New("open error")
		}
		defer func() { sqlOpen = origSqlOpen }()

		gotDB, err := NewPostgres(conf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sql open: open error")
		assert.Nil(t, gotDB)
	})

	t.Run("ping error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		// No need to defer db.Close() because NewPostgres should close it on ping error

		origSqlOpen := sqlOpen
		sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil
		}
		defer func() { sqlOpen = origSqlOpen }()

		mock.ExpectPing().WillReturnError(errors.New("ping failed"))

		gotDB, err := NewPostgres(conf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db ping: ping failed")
		assert.Nil(t, gotDB)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid DSN", func(t *testing.T) {
		invalidConf := config.DatabaseConfig{} // missing host etc
		gotDB, err := NewPostgres(invalidConf)
		assert.Error(t, err)
		assert.Nil(t, gotDB)
	})
}
