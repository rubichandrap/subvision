import { env } from "./config/env";
import { renderOverlayFrames } from "./render";
import { s3Service } from "./services/storage/s3-service";
import { rabbitmqService } from "./services/rabbitmq/rabbitmq-service";
import { RabbitmqJobEventPublisher } from "./services/rabbitmq/job-event-publisher";
import { RabbitmqSubscriberService } from "./services/rabbitmq/rabbitmq-subscriber-service";
import {
  RenderModule,
  combineFramesWithFFmpeg,
  type RenderOptions,
} from "./services/render-module";

// Broker readiness is not part of compose's healthcheck (it pings the
// process, not the AMQP listener), so startup retries instead of crashing
// into `restart: always`.
const CONNECT_RETRIES = 10;
const CONNECT_RETRY_DELAY_MS = 3000;

async function connectWithRetry(): Promise<ReturnType<typeof rabbitmqService.connect>> {
  for (let attempt = 1; attempt <= CONNECT_RETRIES; attempt++) {
    try {
      return await rabbitmqService.connect();
    } catch (error) {
      if (attempt === CONNECT_RETRIES) throw error;
      console.warn(
        `[RabbitMQ] Connect attempt ${attempt}/${CONNECT_RETRIES} failed, retrying in ${CONNECT_RETRY_DELAY_MS}ms:`,
        (error as Error).message
      );
      await new Promise((resolve) => setTimeout(resolve, CONNECT_RETRY_DELAY_MS));
    }
  }
  throw new Error("unreachable");
}

async function main() {
  // init all the 3rd services
  await s3Service.connect();
  const channel = await connectWithRetry();

  // the render module owns paths, options, ffmpeg and the Output upload
  const renderOptions: RenderOptions = {
    fps: env.renderFps,
    width: env.renderWidth,
    height: env.renderHeight,
  };
  const renderModule = new RenderModule(
    s3Service,
    (request) => renderOverlayFrames(request),
    combineFramesWithFFmpeg,
    {
      tmpDir: env.tmpDir,
      bucket: env.s3Bucket,
      options: renderOptions,
      // The fallback template for jobs without an Edit Spec; jobs with one
      // carry their animation in the payload.
      template: env.renderTemplate,
    }
  );

  // init rabbitmq subscribers
  const events = new RabbitmqJobEventPublisher(channel);
  const rabbitmqSubscriberService = new RabbitmqSubscriberService(channel);

  // subscribe
  await rabbitmqSubscriberService.subscribeToVfxJobs(
    (job) => renderModule.run(job),
    events,
    { maxAttempts: env.rabbitmqMaxAttempts, prefetch: env.rabbitmqPrefetch }
  );

  // graceful shutdown
  process.once("SIGINT", async () => {
    console.log("\n[App] Caught SIGINT. Cleaning up...");
    await rabbitmqService.close();
    process.exit(0);
  });
}

main().catch((error) => {
  console.error("Error starting app:", error);
  process.exit(1);
});
