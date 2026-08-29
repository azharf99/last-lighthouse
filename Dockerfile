# ==========================================================
# All-In-One Production Multi-Stage Dockerfile (Alpine Linux)
# The Last Lighthouse (Server + Embedded Static Client)
# ==========================================================

# Stage 1: Build Client Assets
FROM node:22-alpine AS client-builder
WORKDIR /app
COPY client/package.json client/package-lock.json ./
RUN npm ci
COPY client/ ./
RUN npm run build

# Stage 2: Build Go Server
FROM golang:1.24-alpine AS server-builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /bin/lastlighthouse-server ./cmd/server

# Stage 3: Final Runner Container (Nginx Alpine Reverse Proxy + Server)
FROM alpine:3.21

# Install runtime dependencies: Nginx, CA certificates, tzdata, su-exec
RUN apk add --no-cache nginx ca-certificates tzdata \
    && addgroup -S appgroup -g 10001 \
    && adduser -S -u 10001 -G appgroup appuser

WORKDIR /app

# Copy Go binary
COPY --from=server-builder /bin/lastlighthouse-server /app/server

# Copy Frontend static assets to Nginx html folder
COPY --from=client-builder /app/dist /usr/share/nginx/html

# Copy Nginx configuration
COPY nginx/nginx.conf /etc/nginx/nginx.conf
COPY nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf

# Copy entrypoint startup script
COPY scripts/docker-entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 80 8080

ENTRYPOINT ["/app/entrypoint.sh"]
