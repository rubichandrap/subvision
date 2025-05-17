# Subvision

**Subvision** is an AI-powered subtitle generator. Upload your video, and Subvision will automatically generate subtitles using AI, track your process, and let you download the result when it's ready.

---

## Features

- Video upload with resumable uploads (tus protocol)
- Automatic subtitle generation using AI
- Track processing status
- Download processed videos with subtitles
- Modern web UI (Next.js, Tailwind CSS)
- Backend with Gin, MinIO, RabbitMQ, tusd

---

## Architecture

- **Client:** Next.js app ([client/](client))
- **Server:** Go backend ([server/](server))
- **MinIO:** S3-compatible object storage for uploads
- **RabbitMQ:** Job queue for processing
- **tusd:** Resumable upload server

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/)

### 1. Clone the repository

```sh
git clone https://github.com/yourusername/subvision.git
cd subvision
```

### 2. Configure environment variables

Copy the example environment files and edit them as needed:

```sh
cp server/.env.example server/.env
cp client/.env.example client/.env
```

- Edit `server/.env` for backend configuration (see [server/.env.example](server/.env.example))
- Edit `client/.env` for frontend configuration (see [client/.env.example](client/.env.example))

### 3. Start all services

```sh
docker-compose up --build
```

### 4. Access the app

- **Frontend:** [http://localhost:3000](http://localhost:3000)
- **Backend API:** [http://localhost:8000](http://localhost:8000)
- **MinIO Console:** [http://localhost:9001](http://localhost:9001) (user: `minio`, pass: `minio123`)
- **RabbitMQ Console:** [http://localhost:15672](http://localhost:15672) (user: `guest`, pass: `guest`)

---

## Development

### Client

```sh
cd client
pnpm install
pnpm dev
```

### Server

```sh
cd server
go mod tidy
go run ./cmd/subvision/main.go
```

---

## Folder Structure

```
/client   # Next.js frontend
/server   # Go backend
```

---

## Environment Variables

### Server (`server/.env.example`)

```env
PORT=8000
TMP_DIR=/tmp

CLIENT_URL=http://localhost:3000

MINIO_HOST=minio
MINIO_PORT=9000
MINIO_ACCESS_KEY=minio
MINIO_SECRET_KEY=minio123
MINIO_BUCKET=subvision

RABBITMQ_HOST=rabbitmq
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
```

### Client (`client/.env.example`)

```env
NEXT_PUBLIC_SERVER_URL=http://localhost:8000
```

---

## License

MIT

---

**Made with ❤️ for creators.**
