import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Channel, ConsumeMessage } from "amqplib";

import {
  RabbitmqSubscriberService,
  VFX_JOBS_DEAD_QUEUE,
} from "./rabbitmq-subscriber-service";
import {
  JobCompletedEvent,
  JobFailedEvent,
  VFX_JOBS_QUEUE,
  VfxJobPayload,
} from "../../contract";

const job = {
  uploadId: "u1",
  objectKey: "uploads/u1",
  segments: [{ start: 0, end: 1.5, text: "hello" }],
};

class FakeChannel {
  published: Array<{
    exchange: string;
    routingKey: string;
    content: Buffer;
    headers?: Record<string, unknown>;
  }> = [];
  acked: ConsumeMessage[] = [];
  nacked: Array<{ requeue: boolean }> = [];
  consumer?: (msg: ConsumeMessage | null) => Promise<void>;

  async assertQueue(): Promise<void> {}
  prefetch(): void {}
  async consume(
    _queue: string,
    handler: (msg: ConsumeMessage | null) => Promise<void>
  ): Promise<{ consumerTag: string }> {
    this.consumer = handler;
    return { consumerTag: "test" };
  }
  ack(msg: ConsumeMessage): void {
    this.acked.push(msg);
  }
  nack(_msg: ConsumeMessage, _allUpTo: boolean, requeue: boolean): void {
    this.nacked.push({ requeue });
  }
  publish(
    exchange: string,
    routingKey: string,
    content: Buffer,
    options?: { headers?: Record<string, unknown> }
  ): void {
    this.published.push({
      exchange,
      routingKey,
      content,
      headers: options?.headers,
    });
  }
}

class FakeEvents {
  completed: JobCompletedEvent[] = [];
  failed: JobFailedEvent[] = [];

  async publishCompleted(event: JobCompletedEvent): Promise<void> {
    this.completed.push(event);
  }
  async publishFailed(event: JobFailedEvent): Promise<void> {
    this.failed.push(event);
  }
}

function makeMessage(body: unknown, headers?: Record<string, unknown>): ConsumeMessage {
  return {
    content: Buffer.from(typeof body === "string" ? body : JSON.stringify(body)),
    properties: { headers },
  } as unknown as ConsumeMessage;
}

async function subscribe(
  channel: FakeChannel,
  renderJob: (job: VfxJobPayload) => Promise<{ uploadId: string; outputKey: string }>,
  events: FakeEvents,
  maxAttempts = 3
) {
  const subscriber = new RabbitmqSubscriberService(
    channel as unknown as Channel
  );
  await subscriber.subscribeToVfxJobs(renderJob, events, {
    maxAttempts,
    prefetch: 1,
  });
}

describe("vfx job consumer", () => {
  it("a successful render publishes JobCompleted and acks", async () => {
    const channel = new FakeChannel();
    const events = new FakeEvents();
    await subscribe(channel, async (job) => ({ uploadId: job.uploadId, outputKey: "outputs/u1" }), events);

    const msg = makeMessage(job);
    await channel.consumer!(msg);

    assert.deepEqual(events.completed, [
      { uploadId: "u1", outputKey: "outputs/u1" },
    ]);
    assert.deepEqual(events.failed, []);
    assert.equal(channel.acked.length, 1);
    assert.equal(channel.nacked.length, 0);
    assert.equal(channel.published.length, 0);
  });

  it("a failed job is requeued with its attempt count", async () => {
    const channel = new FakeChannel();
    const events = new FakeEvents();
    await subscribe(
      channel,
      async () => {
        throw new Error("transient failure");
      },
      events
    );

    await channel.consumer!(makeMessage(job));

    assert.equal(channel.published.length, 1);
    assert.equal(channel.published[0]!.exchange, "");
    assert.equal(channel.published[0]!.routingKey, VFX_JOBS_QUEUE);
    assert.deepEqual(JSON.parse(channel.published[0]!.content.toString()), job);
    assert.equal(channel.published[0]!.headers?.["x-retry-count"], 1);
    assert.equal(channel.acked.length, 1);
    assert.deepEqual(events.completed, []);
    assert.deepEqual(events.failed, []);
  });

  it("a repeatedly failing job dead-letters with a JobFailed event", async () => {
    const channel = new FakeChannel();
    const events = new FakeEvents();
    await subscribe(
      channel,
      async () => {
        throw new Error("still broken");
      },
      events,
      3
    );

    // two attempts already burned
    await channel.consumer!(makeMessage(job, { "x-retry-count": 2 }));

    assert.deepEqual(events.failed, [
      { uploadId: "u1", reason: "still broken" },
    ]);
    assert.deepEqual(events.completed, []);
    assert.equal(channel.nacked.length, 1);
    assert.equal(channel.nacked[0]!.requeue, false);
    assert.equal(channel.published.length, 0);
    assert.equal(channel.acked.length, 0);
  });

  it("a malformed payload fails loudly and dead-letters", async () => {
    const channel = new FakeChannel();
    const events = new FakeEvents();
    await subscribe(channel, async (job) => ({ uploadId: job.uploadId, outputKey: "outputs/u1" }), events);

    // valid JSON, invalid contract: segments missing
    await channel.consumer!(makeMessage({ uploadId: "u9", objectKey: "uploads/u9" }));

    assert.equal(events.failed.length, 1);
    assert.equal(events.failed[0]!.uploadId, "u9");
    assert.match(events.failed[0]!.reason, /segments/);
    assert.equal(channel.nacked.length, 1);
    assert.equal(channel.nacked[0]!.requeue, false);
    assert.deepEqual(events.completed, []);
  });

  it("a malformed payload without an upload id dead-letters unreported", async () => {
    const channel = new FakeChannel();
    const events = new FakeEvents();
    await subscribe(channel, async (job) => ({ uploadId: job.uploadId, outputKey: "outputs/u1" }), events);

    await channel.consumer!(makeMessage("definitely not json"));

    assert.deepEqual(events.failed, []);
    assert.equal(channel.nacked.length, 1);
  });
});
