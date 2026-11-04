# ---------- STAGE 1: Build Go binary ----------
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

# go-sqlite3 needs cgo; ffmpeg is only needed at runtime but keeping the
# builder lean avoids pulling a C++ inference toolchain.
RUN apk add --no-cache git build-base ffmpeg

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1

RUN go build -o subvision ./cmd/subvision

# ---------- STAGE 2: Runtime ----------
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ffmpeg ca-certificates fontconfig ttf-dejavu

COPY --from=builder /app/subvision .

RUN chmod +x ./subvision

EXPOSE 8080

CMD ["./subvision"]
