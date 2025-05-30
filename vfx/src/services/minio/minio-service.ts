import fs from "fs";
import { Client, ClientOptions } from "minio";
import path from "path";
import { env } from "../../config/env";
import { ensureDirs } from "../../utils/ensure-dirs";

class MinioService {
  private static instance: MinioService;
  private client: Client | null = null;

  constructor(private readonly clientOptions: ClientOptions) {}

  static getInstance(): MinioService {
    if (!MinioService.instance) {
      MinioService.instance = new MinioService({
        endPoint: env.minioHost,
        port: env.minioPort,
        useSSL: false,
        accessKey: env.minioAccessKey,
        secretKey: env.minioSecretKey,
      });
    }
    return MinioService.instance;
  }

  async connect(): Promise<void> {
    this.client = new Client(this.clientOptions);
  }

  async downloadFile(
    bucketName: string,
    objectKey: string,
    filePath: string
  ): Promise<string> {
    const writeStream = fs.createWriteStream(filePath);

    if (!this.client) {
      throw new Error("Minio client is not connected.");
    }

    return new Promise<string>((resolve, reject) => {
      this.client!.getObject(bucketName, objectKey)
        .then((dataStream: NodeJS.ReadableStream) => {
          dataStream.pipe(writeStream);

          writeStream.on("finish", () => resolve(filePath));
          writeStream.on("error", (error: Error) => reject(error));
        })
        .catch((err: Error) => reject(err));
    });
  }

  async uploadFile(
    bucketName: string,
    objectKey: string,
    filePath: string
  ): Promise<void> {
    ensureDirs(env.tmpDir, path.dirname(filePath));

    if (!this.client) {
      throw new Error("Minio client is not connected.");
    }

    // You can pass an empty object for metaData if not needed
    await this.client.fPutObject(bucketName, objectKey, filePath, {});
  }
}

export const minioService = MinioService.getInstance();
