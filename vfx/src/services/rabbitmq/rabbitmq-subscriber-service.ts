import { Channel, ConsumeMessage } from "amqplib";
import { VFX_JOBS_QUEUE, VfxJobPayload, parseVfxJob } from "../../contract";

export class RabbitmqSubscriberService {
  constructor(private readonly channel: Channel) {}

  // subscribeToVfxJobs consumes VFX Jobs from the contract's queue and hands
  // each parsed job to the handler. A malformed payload fails loudly and is
  // discarded — it can never succeed, so requeueing it would only wedge the
  // consumer.
  public async subscribeToVfxJobs(
    handler: (job: VfxJobPayload) => Promise<void>
  ) {
    await this.channel.assertQueue(VFX_JOBS_QUEUE, { durable: true });

    this.channel.prefetch(3); // well, my PC only can afford this around of processing, maybe less sometimes

    console.log(`[Subscriber] Waiting for messages in queue "${VFX_JOBS_QUEUE}"`);

    this.channel.consume(
      VFX_JOBS_QUEUE,
      async (msg: ConsumeMessage | null) => {
        if (!msg) return;

        let job: VfxJobPayload;
        try {
          job = parseVfxJob(JSON.parse(msg.content.toString()));
        } catch (error) {
          console.error(
            `[Subscriber] Malformed VFX Job on "${VFX_JOBS_QUEUE}", discarding it`,
            { error, body: msg.content.toString() }
          );
          this.channel.nack(msg, false, false);
          return;
        }

        console.log(`[Subscriber] Received job:`, job);

        try {
          await handler(job);
          this.channel.ack(msg);
          console.log(`[Subscriber] Job processed and acked:`, job);
        } catch (error) {
          console.error(`[Subscriber] Error processing job:`, error);
          // Failure policy (requeue / dead-letter) lands with the render
          // module; for now the job fails loudly rather than wedging the
          // consumer on an unacked message.
          this.channel.nack(msg, false, false);
        }
      },
      { noAck: false }
    );
  }
}
