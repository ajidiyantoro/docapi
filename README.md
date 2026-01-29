# DocAPI

DocAPI is a Go-based RESTful API service designed for managing documents. It uses the Fiber framework for high-performance HTTP handling, PostgreSQL for metadata storage, and MinIO for object storage.

## Features

- Document management (CRUD operations)
- High-performance HTTP server using [Fiber](https://gofiber.io/)
- PostgreSQL integration for document metadata
- MinIO integration for document file storage
- Environment-based configuration
- Docker support for easy deployment

## Tech Stack

- **Language:** Go 1.22
- **Web Framework:** [Fiber v2](https://github.com/gofiber/fiber)
- **Database:** PostgreSQL (via [pgx](https://github.com/jackc/pgx))
- **Object Storage:** [MinIO](https://min.io/)
- **Configuration:** [godotenv](https://github.com/joho/godotenv)
- **Containerization:** Docker

## Project Structure

```text
.
├── cmd/
│   └── api/
│       └── main.go           # Application entry point
├── docs/                    # Generated Swagger documentation
├── internal/
│   ├── config/               # Configuration loading logic
│   ├── database/             # Database connection setup
│   ├── http/                 # HTTP handlers and middleware
│   ├── model/                # Data models
│   ├── repository/           # Data access layer (PostgreSQL)
│   ├── service/              # Business logic layer
│   └── storage/              # Object storage layer (MinIO)
├── Dockerfile                # Docker build instructions
├── go.mod                    # Go module definition
└── .env                      # Environment variables (not tracked)
```

## Requirements

- Go 1.22 or higher
- PostgreSQL
- MinIO
- Docker (optional)

## Installation & Setup

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd docapi
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Configure environment variables:**
   Create a `.env` file in the root directory and populate it with the required values (see [Environment Variables](#environment-variables)).

## Running the Application

### Database Setup

Before running the application, make sure you have created the necessary table in your PostgreSQL database. You can use the following DDL:

```sql
-- PostgreSQL DDL
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS documents (
  id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  filename     TEXT        NOT NULL,
  storage_path TEXT        NOT NULL UNIQUE,
  size         BIGINT      NOT NULL CHECK (size >= 0),
  content_type TEXT        NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index (optional)
CREATE INDEX IF NOT EXISTS idx_documents_filename ON documents (filename);
CREATE INDEX IF NOT EXISTS idx_documents_content_type ON documents (content_type);
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents (created_at);
```

### Local Development

To run the application locally:
```bash
go run cmd/api/main.go
```

### Using Docker

1. **Build the Docker image:**
   ```bash
   docker build -t docapi .
   ```

2. **Run the container:**
   ```bash
   docker run -p 8080:8080 --env-file .env docapi
   ```

## Environment Variables

The application is configured using environment variables. You can set these in your shell or in a `.env` file.

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Port the server will listen on | `8080` |
| `DB_HOST` | PostgreSQL host | |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | |
| `DB_PASSWORD` | PostgreSQL password | |
| `DB_NAME` | PostgreSQL database name | |
| `DB_SSLMODE` | PostgreSQL SSL mode | `disable` |
| `DB_MAX_OPEN_CONNS` | Max open DB connections | `10` |
| `DB_MAX_IDLE_CONNS` | Max idle DB connections | `5` |
| `DB_CONN_MAX_LIFETIME_SEC` | DB connection max lifetime (sec) | `300` |
| `MINIO_ENDPOINT` | MinIO server endpoint | |
| `MINIO_ACCESS_KEY` | MinIO access key | |
| `MINIO_SECRET_KEY` | MinIO secret key | |
| `MINIO_BUCKET` | MinIO bucket name | |
| `MINIO_USE_SSL` | Use SSL for MinIO | `false` |

## API Documentation

The API documentation is automatically generated using Swagger. Once the application is running, you can access the Swagger UI at:
`http://localhost:8080/swagger/index.html`

To regenerate the documentation after making changes to the annotations:
```bash
swag init -g cmd/api/main.go
```
