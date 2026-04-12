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
- `GET /api/v1/auth/me` - Get current user profile
- `POST /api/v1/categories` - Create category
- `GET /api/v1/categories` - List categories
- `GET /api/v1/categories/{id}` - Get category
- `PUT /api/v1/categories/{id}` - Update category
- `DELETE /api/v1/categories/{id}` - Delete category
- `POST /api/v1/transactions` - Create transaction
- `GET /api/v1/transactions` - List transactions
- `GET /api/v1/transactions/{id}` - Get transaction
- `GET /api/v1/transactions/date-range` - List transactions by date range
- `PUT /api/v1/transactions/{id}` - Update transaction
- `DELETE /api/v1/transactions/{id}` - Delete transaction
- `POST /api/v1/budgets` - Create budget
- `GET /api/v1/budgets` - List budgets
- `GET /api/v1/budgets/{id}` - Get budget
- `PUT /api/v1/budgets/{id}` - Update budget
- `DELETE /api/v1/budgets/{id}` - Delete budget
- `POST /api/v1/drafts` - Create draft transaction
- `GET /api/v1/drafts` - List drafts
- `GET /api/v1/drafts/{id}` - Get draft
- `POST /api/v1/drafts/{id}/confirm` - Confirm draft
- `DELETE /api/v1/drafts/{id}` - Delete draft
- `POST /api/v1/receipts/scan` - Scan receipt with AI

## Security Features

### Rate Limiting
- **Auth endpoints**: 20 requests per 5 minutes (prevents brute force)
- **Protected endpoints**: 100 requests per 1 minute (authenticated users)
- **Health check**: No rate limiting

### Input Validation
- All requests validated using `go-playground/validator`
- Field-level validation errors returned in JSON format
- Password requirements: minimum 8 characters
- Email format validation

### Error Handling
- Consistent JSON error response format
- Proper HTTP status codes
- User-friendly error messages

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `ENVIRONMENT` | Environment (`development`/`production`) | `development` | No |
| `DATABASE_URL` | PostgreSQL connection string | - | Yes |
| `JWT_SECRET` | JWT signing secret | - | Yes |
| `JWT_EXPIRATION` | Access token expiration | `24h` | No |
| `RATE_LIMIT_REQUESTS` | Max requests for protected endpoints | `100` | No |
| `RATE_LIMIT_WINDOW` | Time window for protected endpoints | `1m` | No |
| `RATE_LIMIT_AUTH_REQUESTS` | Max requests for auth endpoints | `20` | No |
| `RATE_LIMIT_AUTH_WINDOW` | Time window for auth endpoints | `5m` | No |
| `LOG_FORMAT` | Log format (`text` or `json`) | `text` | No |
| `GEMINI_API_KEY` | Google Gemini API key (receipt scanning) | - | No |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | - | No |

### Structured Logging

The application uses Go's `log/slog` for structured logging:

**Development Mode** (`ENVIRONMENT=development`):
- Text format with source file/line numbers
- Debug level enabled
- Easy to read during development

**Production Mode** (`ENVIRONMENT=production`):
- JSON format recommended for log aggregation
- Info level enabled
- Includes request IDs for correlation

**Example JSON Log Entry:**
```json
{
  "time": "2026-04-12T10:30:05Z",
  "level": "INFO",
  "msg": "request started",
  "request_id": "host.example.com/random-0001",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "remote_addr": "192.168.1.100:12345"
}
```

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
