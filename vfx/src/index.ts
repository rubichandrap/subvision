import { env } from "./config/env";
import { AnimateService } from "./services/animate-service";
import { minioService } from "./services/minio/minio-service";
import { rabbitmqService } from "./services/rabbitmq/rabbitmq-service";
import { RabbitmqSubscriberService } from "./services/rabbitmq/rabbitmq-subscriber-service";
import { ensureDirs } from "./utils/ensure-dirs";

async function main() {
  const tmpDir = env.tmpDir;
  const videoTmpDir = `${tmpDir}/videos`;
  const framesTmpDir = `${tmpDir}/frames`;
  const outputTmpDir = `${tmpDir}/outputs`;

  ensureDirs(tmpDir, videoTmpDir, framesTmpDir, outputTmpDir);

  // init all the 3rd services
  await minioService.connect();
  const channel = await rabbitmqService.connect();

  // services
  const animateService = new AnimateService();

  // init rabbitmq subscribers
  const rabbitmqSubscriberService = new RabbitmqSubscriberService(channel);

  // subscribe
  await rabbitmqSubscriberService.subscribeToGenerateVfx(animateService);

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
