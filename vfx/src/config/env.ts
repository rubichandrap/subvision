import { config } from "dotenv";

config();

export const env = {
  tmpDir: process.env.TMP_DIR || '/tmp',
  minioHost: process.env.MINIO_HOST || 'minio',
  minioPort: process.env.MINIO_PORT ? parseInt(process.env.MINIO_PORT, 10) : 9000,
  minioAccessKey: process.env.MINIO_ACCESS_KEY || 'minio',
  minioSecretKey: process.env.MINIO_SECRET_KEY || 'minio123',
  minioBucket: process.env.MINIO_BUCKET || 'subvision',
  rabbitmqHost: process.env.RABBITMQ_HOST || 'rabbitmq',
  rabbitmqPort: process.env.RABBITMQ_PORT ? parseInt(process.env.RABBITMQ_PORT, 10) : 5672,
  rabbitmqUser: process.env.RABBITMQ || 'guest',
  rabbitmqPassword: process.env.RABBITMQ_PASSWORD || 'guest',
}
