# ---------- STAGE 1: Build ----------
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ffmpeg

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o subvision ./cmd/subvision

# ---------- STAGE 2: Runtime ----------
FROM alpine:latest

WORKDIR /app

RUN ls -lah /app

RUN apk add --no-cache ffmpeg ca-certificates

# Copy binary + make sure it's executable
COPY --from=builder /app/subvision .
RUN chmod +x ./subvision && ls -lah ./subvision

COPY --from=builder /app/.env .env

EXPOSE 8000

CMD ["./subvision"]
