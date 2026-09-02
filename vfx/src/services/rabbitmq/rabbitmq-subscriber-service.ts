import { Channel, ConsumeMessage } from "amqplib";
import {
  JobCompletedEvent,
  JobFailedEvent,
  VFX_JOBS_QUEUE,
  VfxJobPayload,
  extractUploadId,
  parseVfxJob,
} from "../../contract";
import { RenderOutcome } from "../render-module";
import { JobEventPublisher } from "./job-event-publisher";

// Queues a dead-lettered VFX Job lands in: the main queue routes rejected
// messages here through its dead-letter exchange, so a repeatedly failing job
// is parked for inspection instead of looping forever.
export const VFX_JOBS_DEAD_QUEUE = "vfx_jobs_dead";

const RETRY_COUNT_HEADER = "x-retry-count";

export interface VfxJobSubscriberOptions {
  // How many attempts a job gets before it dead-letters.
  maxAttempts: number;
  // How many jobs this consumer processes concurrently.
  prefetch: number;
}

export class RabbitmqSubscriberService {
  constructor(private readonly channel: Channel) {}

  // subscribeToVfxJobs consumes VFX Jobs from the contract's queue: a job
  // either renders and publishes a JobCompleted event, fails and is requeued
  // with a bounded number of attempts, or dead-letters with a JobFailed
  // event. A malformed payload can never succeed, so it fails loudly and
  // dead-letters immediately.
  public async subscribeToVfxJobs(
    renderJob: (job: VfxJobPayload) => Promise<RenderOutcome>,
    events: JobEventPublisher,
    options: VfxJobSubscriberOptions
  ) {
    await this.channel.assertQueue(VFX_JOBS_QUEUE, {
      durable: true,
      deadLetterExchange: "",
      deadLetterRoutingKey: VFX_JOBS_DEAD_QUEUE,
    });
    await this.channel.assertQueue(VFX_JOBS_DEAD_QUEUE, { durable: true });

    this.channel.prefetch(options.prefetch);

    console.log(`[Subscriber] Waiting for messages in queue "${VFX_JOBS_QUEUE}"`);

    this.channel.consume(
      VFX_JOBS_QUEUE,
      async (msg: ConsumeMessage | null) => {
        if (!msg) return;

        let job: VfxJobPayload;
        try {
          job = parseVfxJob(JSON.parse(msg.content.toString()));
        } catch (error) {
          await this.deadLetterMalformed(msg, error, events);
          return;
        }

        console.log(`[Subscriber] Received job:`, job);

        try {
          const outcome = await renderJob(job);
          await events.publishCompleted({
            uploadId: outcome.uploadId,
            outputKey: outcome.outputKey,
          } satisfies JobCompletedEvent);
          this.channel.ack(msg);
          console.log(`[Subscriber] Job completed and acked:`, job);
        } catch (error) {
          await this.requeueOrFail(msg, job, error, events, options.maxAttempts);
        }
      },
      { noAck: false }
    );
  }

  private async requeueOrFail(
    msg: ConsumeMessage,
    job: VfxJobPayload,
    error: unknown,
    events: JobEventPublisher,
    maxAttempts: number
  ) {
    const attempts = countAttempts(msg) + 1;
    const reason = errorMessage(error);

    if (attempts < maxAttempts) {
      console.error(
        `[Subscriber] Job for upload ${job.uploadId} failed (attempt ${attempts}/${maxAttempts}), requeueing:`,
        error
      );
      this.channel.publish("", VFX_JOBS_QUEUE, msg.content, {
        contentType: "application/json",
        persistent: true,
        headers: {
          ...(msg.properties.headers ?? {}),
          [RETRY_COUNT_HEADER]: attempts,
        },
      });
      this.channel.ack(msg);
      return;
    }

    console.error(
      `[Subscriber] Job for upload ${job.uploadId} failed ${maxAttempts} times, dead-lettering to "${VFX_JOBS_DEAD_QUEUE}":`,
      error
    );
    await events.publishFailed({
      uploadId: job.uploadId,
      reason,
    } satisfies JobFailedEvent);
    this.channel.nack(msg, false, false);
  }

  private async deadLetterMalformed(
    msg: ConsumeMessage,
    error: unknown,
    events: JobEventPublisher
  ) {
    console.error(
      `[Subscriber] Malformed VFX Job on "${VFX_JOBS_QUEUE}", dead-lettering to "${VFX_JOBS_DEAD_QUEUE}":`,
      error,
      `\nbody: ${msg.content.toString()}`
    );

    // Report the failure when the payload at least names its upload, so the
    // server can move the process to failed instead of it hanging in-flight.
    const body: unknown = safeJsonParse(msg.content.toString());
    const uploadId = extractUploadId(body);
    if (uploadId) {
      await events.publishFailed({
        uploadId,
        reason: errorMessage(error),
      } satisfies JobFailedEvent);
    }

    this.channel.nack(msg, false, false);
  }
}

function countAttempts(msg: ConsumeMessage): number {
  const value = msg.properties.headers?.[RETRY_COUNT_HEADER];
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function safeJsonParse(body: string): unknown {
  try {
    return JSON.parse(body);
  } catch {
    return undefined;
  }
}
