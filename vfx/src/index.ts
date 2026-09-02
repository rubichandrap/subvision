import { env } from "./config/env";
import { renderImagesFromTemplate } from "./render";
import { s3Service } from "./services/storage/s3-service";
import { rabbitmqService } from "./services/rabbitmq/rabbitmq-service";
import { RabbitmqJobEventPublisher } from "./services/rabbitmq/job-event-publisher";
import { RabbitmqSubscriberService } from "./services/rabbitmq/rabbitmq-subscriber-service";
import { RenderModule } from "./services/render-module";
import { spawn } from "child_process";

// combineFramesWithFFmpeg overlays the rendered subtitle frames onto the
// video; the render module owns the invocation through this combiner.
async function combineFramesWithFFmpeg(
  videoPath: string,
  framesDir: string,
  outputPath: string,
  fps: number
): Promise<void> {
  const ffmpeg = spawn("ffmpeg", [
    "-framerate",
    String(fps),
    "-i",
    `${framesDir}/element-%03d.png`, // overlay
    "-i",
    videoPath, // background
    "-filter_complex",
    "[1:v][0:v]overlay=0:0", // overlay on top
    "-c:v",
    "libx264",
    "-crf",
    "23",
    "-preset",
    "fast",
    "-c:a",
    "aac",
    "-shortest",
    outputPath,
  ]);

  // Log FFmpeg output for debugging
  ffmpeg.stdout.on("data", (data) => {
    console.log(`FFmpeg Output: ${data}`);
  });

  ffmpeg.stderr.on("data", (data) => {
    console.error(`FFmpeg Error: ${data}`);
  });

  // Handle FFmpeg process completion
  return new Promise((resolve, reject) => {
    ffmpeg.on("close", (code) => {
      if (code === 0) {
        console.log(`Video created successfully: ${outputPath}`);
        resolve();
      } else {
        reject(new Error(`FFmpeg process failed with code ${code}`));
      }
    });
  });
}

async function main() {
  // init all the 3rd services
  await s3Service.connect();
  const channel = await rabbitmqService.connect();

  // the render module owns paths, options, ffmpeg and the Output upload
  const renderModule = new RenderModule(
    s3Service,
    (segments, framesDir) =>
      renderImagesFromTemplate(segments, env.renderTemplate, framesDir, {
        fps: env.renderFps,
        width: env.renderWidth,
        height: env.renderHeight,
      }),
    combineFramesWithFFmpeg,
    {
      tmpDir: env.tmpDir,
      bucket: env.s3Bucket,
      options: { fps: env.renderFps, width: env.renderWidth, height: env.renderHeight },
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
