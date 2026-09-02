import { config } from "dotenv";

config();

export const env = {
  tmpDir: process.env.TMP_DIR || '/tmp',
  s3Endpoint: process.env.S3_ENDPOINT || 'http://rustfs:9000',
  s3AccessKey: process.env.S3_ACCESS_KEY || 'rustfs',
  s3SecretKey: process.env.S3_SECRET_KEY || 'rustfs123',
  s3Bucket: process.env.S3_BUCKET || 'subvision',
  rabbitmqHost: process.env.RABBITMQ_HOST || 'rabbitmq',
  rabbitmqPort: process.env.RABBITMQ_PORT ? parseInt(process.env.RABBITMQ_PORT, 10) : 5672,
  rabbitmqUser: process.env.RABBITMQ_USER || 'guest',
  rabbitmqPassword: process.env.RABBITMQ_PASSWORD || 'guest',
}
