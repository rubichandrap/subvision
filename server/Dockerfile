# ---------- STAGE 1: Build Go binary ----------
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git build-base cmake ffmpeg

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/third_party/whisper.cpp/bindings/go

RUN make whisper

ENV CGO_ENABLED=1 \
  C_INCLUDE_PATH=/app/third_party/whisper.cpp/include:/app/third_party/whisper.cpp/ggml/include \
  LIBRARY_PATH=/app/third_party/whisper.cpp/build_go/src:/app/third_party/whisper.cpp/build_go/ggml/src

WORKDIR /app

RUN go build -o subvision ./cmd/subvision

# ---------- STAGE 2: Runtime ----------
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ffmpeg ca-certificates

COPY --from=builder /app/subvision .

RUN chmod +x ./subvision

COPY --from=builder /app/.env .env

COPY --from=builder /app/third_party/whisper.cpp/bindings/go/models/ third_party/whisper.cpp/bindings/go/models

EXPOSE 8080

CMD ["./subvision"]
