import { config } from "dotenv";

config();

function int(value: string | undefined, fallback: number): number {
  const parsed = value ? parseInt(value, 10) : NaN;
  return Number.isFinite(parsed) ? parsed : fallback;
}

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
  // How many jobs the consumer processes concurrently.
  rabbitmqPrefetch: int(process.env.RABBITMQ_PREFETCH, 1),
  // How many attempts a job gets before it dead-letters.
  rabbitmqMaxAttempts: int(process.env.RABBITMQ_MAX_ATTEMPTS, 3),
  // Render options; the composition id of the subtitle template to render.
  renderFps: int(process.env.RENDER_FPS, 30),
  renderWidth: int(process.env.RENDER_WIDTH, 1920),
  renderHeight: int(process.env.RENDER_HEIGHT, 1080),
  renderTemplate: process.env.RENDER_TEMPLATE || 'karaoke',
}
