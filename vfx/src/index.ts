import { env } from "./config/env";
import { renderImagesFromTemplate } from "./render";
import { s3Service } from "./services/storage/s3-service";
import { rabbitmqService } from "./services/rabbitmq/rabbitmq-service";
import { RabbitmqJobEventPublisher } from "./services/rabbitmq/job-event-publisher";
import { RabbitmqSubscriberService } from "./services/rabbitmq/rabbitmq-subscriber-service";
import {
  RenderModule,
  combineFramesWithFFmpeg,
  type RenderOptions,
} from "./services/render-module";

async function main() {
  // init all the 3rd services
  await s3Service.connect();
  const channel = await rabbitmqService.connect();

  // the render module owns paths, options, ffmpeg and the Output upload
  const renderOptions: RenderOptions = {
    fps: env.renderFps,
    width: env.renderWidth,
    height: env.renderHeight,
  };
  const renderModule = new RenderModule(
    s3Service,
    (segments, framesDir) =>
      renderImagesFromTemplate(segments, env.renderTemplate, framesDir, renderOptions),
    combineFramesWithFFmpeg,
    {
      tmpDir: env.tmpDir,
      bucket: env.s3Bucket,
      options: renderOptions,
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
});
