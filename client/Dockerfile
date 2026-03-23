# Frontend Dockerfile - Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm install

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM node:20-alpine

WORKDIR /app

# Install simple HTTP server to serve static files
RUN npm install -g serve

# Copy built application from builder
COPY --from=builder /app/build ./build

# Expose port
EXPOSE 80

# Start the application
# -s: Single page application mode (handles client-side routing)
# -l: Listen port
CMD ["serve", "-s", "build", "-l", "tcp://0.0.0.0:80"]