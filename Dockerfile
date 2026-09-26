# Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# Build Backend
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot

# Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

# Hugging Face Spaces requires running as non-root user 1000
RUN adduser -D -u 1000 appuser
USER 1000
WORKDIR /home/appuser

# Copy compiled backend
COPY --from=backend-builder --chown=1000:1000 /app/bot .

# Copy compiled frontend
COPY --from=frontend-builder --chown=1000:1000 /app/dist ./web/dist

# Expose port and run
EXPOSE 7860
CMD ["./bot"]
