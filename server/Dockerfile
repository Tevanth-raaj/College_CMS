# Backend Dockerfile - Multi-stage build
FROM golang:1.24-alpine AS builder

# Install build dependencies including Chrome/Chromium for PDF generation
RUN apk add --no-cache \
    git \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Production stage
FROM alpine:latest

# Install runtime dependencies for Chrome/Chromium and PDF generation
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont \
    font-noto-emoji \
    wget

# Set Chrome executable path for chromedp
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/bin/chromium-browser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Create necessary directories
RUN mkdir -p /app/uploads /app/pdfs

# Expose port
EXPOSE 5000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:5000/api/health || exit 1
# Run the application
CMD ["./main"]
