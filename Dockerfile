# Stage 1: Build React frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY internal/ internal/
COPY cmd/ cmd/
COPY static/ static/
COPY templates/ templates/
COPY --from=frontend /app/frontend/dist frontend/dist
RUN CGO_ENABLED=0 go build -o zrp ./cmd/zrp

# Stage 3: Runtime
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app

# Create directories for persistent data
RUN mkdir -p /app/data /app/uploads

# Copy binary and static assets
COPY --from=backend /app/zrp .
COPY --from=backend /app/static static/
COPY --from=backend /app/templates templates/
COPY --from=backend /app/frontend/dist frontend/dist/

# Expose port
EXPOSE 9000

# Set database path to persistent volume
CMD ["./zrp", "-db", "/app/data/zrp.db"]