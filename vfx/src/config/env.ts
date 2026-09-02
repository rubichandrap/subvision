import { config } from "dotenv";

config();

// Thin, typed loader for the environment contract documented in the README:
// required variables fail loudly at boot, naming the missing variable. No
// credential or URL ever falls back to a silent default.

function requireString(name: string): string {
  const value = process.env[name];
  if (value === undefined || value === "") {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function requireInt(name: string): number {
  const value = requireString(name);
  const parsed = parseInt(value, 10);
  if (!Number.isFinite(parsed)) {
    throw new Error(
      `Environment variable ${name} must be an integer, got ${JSON.stringify(value)}`
    );
  }
  return parsed;
}

function optionalInt(name: string, fallback: number): number {
  const value = process.env[name];
  if (value === undefined || value === "") {
    return fallback;
  }
  const parsed = parseInt(value, 10);
  if (!Number.isFinite(parsed)) {
    throw new Error(
      `Environment variable ${name} must be an integer, got ${JSON.stringify(value)}`
    );
  }
  return parsed;
}

export const env = {
  tmpDir: requireString("TMP_DIR"),
  s3Endpoint: requireString("S3_ENDPOINT"),
  s3AccessKey: requireString("S3_ACCESS_KEY"),
  s3SecretKey: requireString("S3_SECRET_KEY"),
  s3Bucket: requireString("S3_BUCKET"),
  rabbitmqHost: requireString("RABBITMQ_HOST"),
  rabbitmqPort: requireInt("RABBITMQ_PORT"),
  rabbitmqUser: requireString("RABBITMQ_USER"),
  rabbitmqPassword: requireString("RABBITMQ_PASSWORD"),
  // How many jobs the consumer processes concurrently.
  rabbitmqPrefetch: optionalInt("RABBITMQ_PREFETCH", 1),
  // How many attempts a job gets before it dead-letters.
  rabbitmqMaxAttempts: optionalInt("RABBITMQ_MAX_ATTEMPTS", 3),
  // Render options; the composition id of the subtitle template to render.
  renderFps: optionalInt("RENDER_FPS", 30),
  renderWidth: optionalInt("RENDER_WIDTH", 1920),
  renderHeight: optionalInt("RENDER_HEIGHT", 1080),
  renderTemplate: process.env["RENDER_TEMPLATE"] || "karaoke",
};
