# CashLens Backend

Financial tracking application backend built with Go.

**Production URL:** `https://cashlensbackend-production.up.railway.app`

## Tech Stack

- **Language**: Go 1.22+
- **HTTP Router**: Chi v5
- **Database**: PostgreSQL with pgx/v5
- **Migrations**: Goose v3
- **Authentication**: JWT (golang-jwt/jwt/v5) with refresh token rotation
- **Rate Limiting**: httprate (per-IP, per-endpoint)
- **External APIs**: Google Gemini AI (receipt scanning), Xendit (payments), Telegram Bot
- **Config**: godotenv

## Project Structure

```
cashlens_backend/
├── cmd/server/          # Application entry point
├── internal/
│   ├── config/          # Configuration loading
│   ├── database/        # Database connection pooling & migrations
│   ├── errors/          # Custom error types & JSON responses
│   ├── handlers/        # HTTP handlers (REST endpoints)
│   ├── logger/          # Structured logging (slog)
│   ├── middleware/      # Auth, CORS, rate limit, security headers, quota
│   ├── models/          # Domain models & request/response structs
│   ├── pkg/
│   │   ├── validator/   # Input validation (go-playground/validator)
│   │   └── xendit/      # Xendit payment gateway client
│   ├── repository/      # Data access layer (SQL queries)
│   ├── service/         # Business logic layer
│   └── telegram/        # Telegram bot service
└── migrations/          # Database migrations (Goose)
```

## Setup

1. **Install Dependencies**
   ```bash
   go mod tidy
   ```

2. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Run Migrations**
   ```bash
   make migrate-up
   ```

4. **Start Server**
   ```bash
   make run
   ```

## API Endpoints

### Public Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/auth/register` | User registration (seeds default categories) |
| `POST` | `/api/v1/auth/login` | User login |
| `POST` | `/api/v1/auth/refresh` | Refresh access token |
| `POST` | `/api/v1/webhooks/payment` | Xendit payment webhook (signature-verified) |

### Protected Endpoints (Requires JWT Bearer Token)
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/auth/me` | Get current user profile |
| `POST` | `/api/v1/auth/logout` | Revoke refresh tokens |
| `GET` | `/api/v1/subscription` | Get subscription status & quota usage |
| `POST` | `/api/v1/payments/create-invoice` | Create Xendit payment invoice |
| `POST` | `/api/v1/categories` | Create category |
| `GET` | `/api/v1/categories` | List categories |
| `GET` | `/api/v1/categories/{id}` | Get category |
| `PUT` | `/api/v1/categories/{id}` | Update category |
| `DELETE` | `/api/v1/categories/{id}` | Delete category |
| `POST` | `/api/v1/transactions` | Create transaction |
| `GET` | `/api/v1/transactions` | List transactions |
| `GET` | `/api/v1/transactions/{id}` | Get transaction |
| `GET` | `/api/v1/transactions/date-range` | List transactions by date range |
| `PUT` | `/api/v1/transactions/{id}` | Update transaction |
| `DELETE` | `/api/v1/transactions/{id}` | Delete transaction (soft-delete) |
| `POST` | `/api/v1/budgets` | Create budget |
| `GET` | `/api/v1/budgets` | List budgets |
| `GET` | `/api/v1/budgets/{id}` | Get budget |
| `PUT` | `/api/v1/budgets/{id}` | Update budget |
| `DELETE` | `/api/v1/budgets/{id}` | Delete budget |
| `POST` | `/api/v1/drafts` | Create draft transaction |
| `GET` | `/api/v1/drafts` | List drafts |
| `GET` | `/api/v1/drafts/{id}` | Get draft |
| `POST` | `/api/v1/drafts/{id}/confirm` | Confirm draft → transaction |
| `DELETE` | `/api/v1/drafts/{id}` | Delete draft |
| `POST` | `/api/v1/receipts/scan` | Scan receipt with Gemini AI |

### Example: Subscription Status
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/subscription
```
```json
{
  "data": {
    "tier": "free",
    "quota": {
      "transactions_used": 12,
      "transactions_limit": 50,
      "scans_used": 3,
      "scans_limit": 5
    }
  }
}
```

### Example: Create Invoice
```bash
curl -X POST http://localhost:8080/api/v1/payments/create-invoice \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"plan":"monthly"}'
```
```json
{
  "data": {
    "payment_url": "https://invoice.xendit.co/...",
    "invoice_id": "cashlens-abc12345-1713000000",
    "expires_at": "2026-05-13T12:00:00Z",
    "amount": 15000,
    "plan": "monthly"
  }
}
```

## Security Features

### Authentication
- JWT access tokens (24h expiry)
- Refresh token rotation with token-family compromise detection
- Session revocation on logout

### Rate Limiting
| Endpoint Group | Limit | Purpose |
|----------------|-------|---------|
| Auth (login/register) | 20 req / 5 min | Brute-force protection |
| Protected endpoints | 100 req / 1 min | General abuse prevention |
| Health check | No limit | Monitoring |

### Quota Enforcement
| Tier | Transactions/Month | AI Scans/Month |
|------|-------------------|----------------|
| Free | 50 | 5 |
| Premium | Unlimited | Unlimited |

- Atomic SQL check+increment prevents TOCTOU race conditions
- Quota checked before resource-intensive operations

### Input Validation
- All requests validated using `go-playground/validator`
- Field-level validation errors returned in JSON `details` field
- Password: minimum 8 characters
- Email: RFC-compliant format validation

### Security Headers (Production)
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security` (HSTS)
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy: default-src 'none'`

### CORS
- Development: Permissive (`*`)
- Production: Domain whitelist with wildcard subdomain support
- Startup warning if production origins are unconfigured

### Request Size Limits
- JSON endpoints: 1MB max
- Receipt uploads: 10MB max (multipart)

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `ENVIRONMENT` | Environment (`development`/`production`) | `development` | No |
| `DATABASE_URL` | PostgreSQL connection string | - | Yes |
| `JWT_SECRET` | JWT signing secret | - | Yes |
| `JWT_EXPIRATION` | Access token expiration | `24h` | No |
| `JWT_REFRESH_EXPIRATION` | Refresh token expiration | `168h` (7 days) | No |
| `RATE_LIMIT_REQUESTS` | Max requests for protected endpoints | `100` | No |
| `RATE_LIMIT_WINDOW` | Time window for protected endpoints | `1m` | No |
| `RATE_LIMIT_AUTH_REQUESTS` | Max requests for auth endpoints | `20` | No |
| `RATE_LIMIT_AUTH_WINDOW` | Time window for auth endpoints | `5m` | No |
| `LOG_FORMAT` | Log format (`text` or `json`) | `text` | No |
| `GEMINI_API_KEY` | Google Gemini API key (receipt scanning) | - | No |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | - | No |
| `XENDIT_WEBHOOK_TOKEN` | Xendit callback token (webhook verification) | - | No |
| `XENDIT_SECRET_KEY` | Xendit secret key (invoice creation) | - | No |
| `XENDIT_SUCCESS_URL` | Redirect URL after successful payment | - | No |
| `XENDIT_FAILURE_URL` | Redirect URL after failed payment | - | No |

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
  "time": "2026-04-13T10:30:05Z",
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

## Database Schema

**11 tables:** `users`, `categories`, `transactions`, `draft_transactions`, `budgets`, `user_chat_links`, `refresh_tokens`, `user_quotas`, `usage_logs`, `subscription_events`, `pending_invoices`

See `docs/backend/DATABASE_SCHEMA.md` for full schema documentation.

## Deployment

**Production:** Railway (`https://cashlensbackend-production.up.railway.app`)
- Auto-migrations on server startup via Goose
- PostgreSQL provisioned on Railway (public proxy endpoint)
- Multi-stage Dockerfile (Alpine base)

```bash
# Deploy to Railway
railway up
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

## Documentation

| Document | Description |
|----------|-------------|
| `docs/ARCHITECTURE_OVERVIEW.md` | System architecture & data flow |
| `docs/backend/BACKEND_PLANNING.md` | Implementation roadmap |
| `docs/backend/DATABASE_SCHEMA.md` | Complete PostgreSQL schema |
| `docs/PAYMENT_INTEGRATION.md` | Payment integration guide |
| `docs/SECURITY_IMPROVEMENTS.md` | Security hardening summary |
| `docs/future_project/MASTER_ROADMAP.md` | Full feature roadmap |
| `apidog-openapi.yaml` | OpenAPI specification (import to Bruno/Postman) |
