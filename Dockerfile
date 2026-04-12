# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Allow Go to auto-download the toolchain version required by go.mod
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Install git (needed by GOTOOLCHAIN=auto to fetch toolchains)
RUN apk --no-cache add git

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
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
