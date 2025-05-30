import { Channel, ConsumeMessage } from "amqplib";
import { AnimateService } from "../animate-service";

export class RabbitmqSubscriberService {
  constructor(private readonly channel: Channel) {}

  public async subscribeToGenerateVfx(animateService: AnimateService) {
    const queue = "generate_vfx_queue";
    await this.channel.assertQueue(queue, { durable: true });

    this.channel.prefetch(3); // well, my PC only can afford this around of processing, maybe less sometimes

    console.log(`[Subscriber] Waiting for messages in queue "${queue}"`);

    this.channel.consume(
      queue,
      async (msg: ConsumeMessage | null) => {
        if (!msg) return;

        const payload = JSON.parse(msg.content.toString()) as Record<
          string,
          any
        >;
        console.log(`[Subscriber] Received job:`, payload);

        const { objectKey, segments, animationType } = payload;

        try {
          await animateService.create(objectKey, segments, animationType);

          // TODO: maybe we should send some publish event after the process done(?)
          // for now it's not necessary, but I think it will be a very good idea
          // if we can notify another service or even the user that the process has been done

          this.channel.ack(msg);
          console.log(`[Subscriber] Job processed and acked:`, payload);
        } catch (error) {
          console.error(`[Subscriber] Error processing job:`, error);
        }
      },
      { noAck: false }
    );
  }
}
