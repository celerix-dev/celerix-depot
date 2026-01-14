# Stage 1: Build the frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build the backend
FROM golang:1.25.1-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Copy the built frontend to the backend directory for embedding
COPY --from=frontend-builder /app/frontend/dist ./cmd/depot/dist
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o depot ./cmd/depot/main.go

# Stage 3: Final image
FROM alpine:latest
RUN apk add --no-cache ca-certificates sqlite
WORKDIR /app
COPY --from=backend-builder /app/depot .
RUN mkdir -p data/uploads
EXPOSE 8080
ENV PORT=8080
ENV DB_PATH=/app/data/depot.db
ENV STORAGE_DIR=/app/data/uploads
CMD ["./depot"]
