# CashLens Backend

Financial tracking application backend built with Go.

## Tech Stack

- **Language**: Go 1.22+
- **HTTP Router**: Chi v5
- **Database**: PostgreSQL with pgx/v5
- **Migrations**: Goose v3
- **Authentication**: JWT (golang-jwt/jwt/v5)
- **Rate Limiting**: httprate
- **Config**: godotenv

## Project Structure

```
cashlens_backend/
├── cmd/server/          # Application entry point
├── internal/
│   ├── config/          # Configuration loading
│   ├── database/        # Database connection pooling
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # Custom middleware
│   ├── models/          # Domain models
│   ├── repository/      # Data access layer
│   └── service/         # Business logic layer
└── migrations/          # Database migrations
```

## Setup

1. **Install Dependencies**
   ```bash
   go mod tidy
   ```

2. **Start PostgreSQL**
   ```bash
   docker-compose up -d
   ```

3. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run Migrations**
   ```bash
   make migrate-up
   ```

5. **Start Server**
   ```bash
   make run
   ```

## API Endpoints

### Public Endpoints
- `GET /health` - Health check
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login

### Protected Endpoints (Requires JWT)
- `POST /api/v1/categories` - Create category
- `GET /api/v1/categories` - List categories
- `POST /api/v1/transactions` - Create transaction
- `GET /api/v1/transactions` - List transactions

## Development

```bash
# Run server
make run

# Build binary
make build

# Run tests
make test

# Create new migration
make migrate-create name=migration_name

# Migrate up
make migrate-up

# Migrate down
make migrate-down
```

## Testing

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "database": "healthy"
}
```
