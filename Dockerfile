# syntax=docker/dockerfile:1

# ---- 1. Build the React SPA -------------------------------------------------
FROM node:24-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# ---- 2. Build the Go binary -------------------------------------------------
FROM golang:1.26-alpine AS api
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- 3. Minimal runtime -----------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=api /out/server /app/server
COPY --from=web /src/frontend/dist /app/static
ENV PORT=8080 STATIC_DIR=/app/static
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK NONE
ENTRYPOINT ["/app/server"]
