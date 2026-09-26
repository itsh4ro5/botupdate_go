# ============================================================
# Stage 1: Build React frontend
# ============================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app

COPY web/package.json web/package-lock.json* ./
RUN npm install --no-audit --no-fund

COPY web/ ./

RUN npm run build


# ============================================================
# Stage 2: Build Go backend
# ============================================================
FROM golang:1.26-alpine AS backend-builder

WORKDIR /app

# Keep Go compiler memory usage as low as possible
ENV GOMAXPROCS=1
ENV GOFLAGS="-p=1"
ENV GOMEMLIMIT=1200MiB

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Single-package compilation with stripped binary
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /tmp/bot \
    ./cmd/bot


# ============================================================
# Stage 3: Minimal Hugging Face runtime
# ============================================================
FROM alpine:3.22

RUN apk --no-cache add ca-certificates tzdata

# Hugging Face Spaces non-root user
RUN adduser -D -u 1000 appuser

USER 1000

WORKDIR /home/appuser

# Go backend
COPY --from=backend-builder --chown=1000:1000 /tmp/bot ./bot

# React frontend
COPY --from=frontend-builder --chown=1000:1000 /app/dist ./web/dist

# Hugging Face Spaces default port
EXPOSE 7860

# PORT is supplied by Hugging Face.
# Fiber application reads PORT itself.
CMD ["./bot"]
