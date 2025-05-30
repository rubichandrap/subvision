import { env } from "@/config/env";
import amqp, { Channel, ChannelModel, Options } from "amqplib";

class RabbitmqService {
  private static instance: RabbitmqService;
  private connection: ChannelModel | null = null;
  private channel: Channel | null = null;

  private constructor(private readonly connectOptions: Options.Connect) {}

  public static getInstance(): RabbitmqService {
    if (!RabbitmqService.instance) {
      RabbitmqService.instance = new RabbitmqService({
        hostname: env.rabbitmqHost,
        port: env.rabbitmqPort,
        username: env.rabbitmqUser,
        password: env.rabbitmqPassword,
      });
    }
    return RabbitmqService.instance;
  }

  public async connect(): Promise<Channel> {
    if (!this.connection || !this.channel) {
      this.connection = await amqp.connect(this.connectOptions);
      this.channel = await this.connection.createChannel();

      this.connection.on("error", (err) => {
        console.error("[RabbitMQ] Connection error:", err);
        this.connection = null;
        this.channel = null;
      });

      this.connection.on("close", () => {
        console.warn("[RabbitMQ] Connection closed.");
        this.connection = null;
        this.channel = null;
      });

      console.log("[RabbitMQ] Connected");
    }

    return this.channel;
  }

  public async close(): Promise<void> {
    await this.channel?.close();
    await this.connection?.close();
    console.log("[RabbitMQ] Connection closed cleanly");
  }
}

export const rabbitmqService = RabbitmqService.getInstance();
