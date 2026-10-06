# ---- Frontend build ----
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci || npm install
COPY frontend/ ./
RUN npm run build

# ---- Backend build ----
FROM golang:1.23-alpine AS backend-build
RUN apk add --no-cache git ca-certificates gcc musl-dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /regpilot ./cmd/server

# ---- Runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget docker-cli \
    && adduser -D -u 1000 regpilot \
    && mkdir -p /data /app/static \
    && chown -R regpilot:regpilot /data /app

WORKDIR /app
COPY --from=backend-build /regpilot /app/regpilot
COPY --from=frontend-build /app/frontend/dist /app/static
COPY docs/openapi.json /app/static/openapi.json
COPY docs/openapi.json /app/docs/openapi.json

ENV APP_PORT=8080 \
    DATABASE_PATH=/data/regpilot.db \
    APP_ENV=production \
    STATIC_DIR=/app/static

EXPOSE 8080
VOLUME ["/data"]
USER regpilot

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/app/regpilot"]
