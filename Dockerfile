# Multi-stage Dockerfile for musicd
# Stage 1: Build the frontend using Node.js
FROM node:20-alpine AS frontend-builder

# Enable corepack for yarn support
RUN corepack enable

# Set working directory for frontend build
WORKDIR /app/frontend

# # Copy frontend package files
# COPY frontend/package.json frontend/yarn.lock ./

# # Install frontend dependencies
# RUN corepack yarn install --frozen-lockfile

# Copy frontend source code
COPY frontend/ ./

# Build the frontend (equivalent to: cd ./frontend && corepack yarn install && corepack yarn generate)
RUN corepack yarn install && corepack yarn generate

# Stage 2: Build the Go backend
FROM golang:1.24-alpine AS backend-builder

# Install necessary build tools
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Copy built frontend files from previous stage
COPY --from=frontend-builder /app/frontend/.output/public/ ./cmd/musicd/ui/

# Build the Go application (equivalent to: mkdir -p ./bin && go build -o ./bin/musicd ./cmd/musicd)
RUN mkdir -p ./bin && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ./bin/musicd ./cmd/musicd

# Stage 3: Final runtime image
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=backend-builder /app/bin/musicd .

# Copy the database schema file
COPY --from=backend-builder /app/init.sql .

# Create necessary directories
RUN mkdir -p /music /playlists

# Set environment variables with defaults
ENV MUSIC_DIR=/music
ENV PLAYLISTS_DIR=/playlists
ENV DATABASE_URL=""
ENV PATH_PREFIX=""

# Expose port (adjust if needed)
EXPOSE 8080

# Run the application
CMD ["/root/musicd"]