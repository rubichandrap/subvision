import { Channel } from "amqplib";
import {
  JOB_COMPLETED_QUEUE,
  JOB_FAILED_QUEUE,
  JobCompletedEvent,
  JobFailedEvent,
} from "../../contract";

// JobEventPublisher reports render outcomes to the server.
export interface JobEventPublisher {
  publishCompleted(event: JobCompletedEvent): Promise<void>;
  publishFailed(event: JobFailedEvent): Promise<void>;
}

export class RabbitmqJobEventPublisher implements JobEventPublisher {
  constructor(private readonly channel: Channel) {}

  async publishCompleted(event: JobCompletedEvent): Promise<void> {
    await this.channel.assertQueue(JOB_COMPLETED_QUEUE, { durable: true });
    this.channel.publish(
      "",
      JOB_COMPLETED_QUEUE,
      Buffer.from(JSON.stringify(event)),
      { contentType: "application/json", persistent: true }
    );
  }

  async publishFailed(event: JobFailedEvent): Promise<void> {
    await this.channel.assertQueue(JOB_FAILED_QUEUE, { durable: true });
    this.channel.publish("", JOB_FAILED_QUEUE, Buffer.from(JSON.stringify(event)), {
      contentType: "application/json",
      persistent: true,
    });
  }
}
