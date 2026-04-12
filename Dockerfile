# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/server

# Stage 2: Run
FROM alpine:3.19

WORKDIR /app

# Install CA certificates (needed for HTTPS calls to Gemini, Telegram, Neon)
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/bin/server .

EXPOSE 8080

CMD ["./server"]
