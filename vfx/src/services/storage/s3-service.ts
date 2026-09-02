import fs from "fs";
import path from "path";
import { Readable } from "stream";
import {
  GetObjectCommand,
  PutObjectCommand,
  S3Client,
  S3ClientConfig,
} from "@aws-sdk/client-s3";
import { env } from "../../config/env";
import { ensureDirs } from "../../utils/ensure-dirs";

// The SDK types the body as a union; in Node it is a readable stream.
type ObjectBody = Readable;

class S3Service {
  private static instance: S3Service;
  private client: S3Client | null = null;

  constructor(private readonly clientOptions: S3ClientConfig) {}

  static getInstance(): S3Service {
    if (!S3Service.instance) {
      S3Service.instance = new S3Service({
        endpoint: env.s3Endpoint,
        region: "us-east-1",
        forcePathStyle: true,
        credentials: {
          accessKeyId: env.s3AccessKey,
          secretAccessKey: env.s3SecretKey,
        },
      });
    }
    return S3Service.instance;
  }

  async connect(): Promise<void> {
    this.client = new S3Client(this.clientOptions);
  }

  async downloadFile(
    bucketName: string,
    objectKey: string,
    filePath: string
  ): Promise<string> {
    if (!this.client) {
      throw new Error("S3 client is not connected.");
    }

    const response = await this.client.send(
      new GetObjectCommand({ Bucket: bucketName, Key: objectKey })
    );
    if (!response.Body) {
      throw new Error(`Object ${objectKey} has an empty body.`);
    }

    const writeStream = fs.createWriteStream(filePath);

    return new Promise<string>((resolve, reject) => {
      (response.Body as ObjectBody)
        .pipe(writeStream)
        .on("finish", () => resolve(filePath))
        .on("error", (error: Error) => reject(error));
    });
  }

  async uploadFile(
    bucketName: string,
    objectKey: string,
    filePath: string
  ): Promise<void> {
    ensureDirs(env.tmpDir, path.dirname(filePath));

    if (!this.client) {
      throw new Error("S3 client is not connected.");
    }

    await this.client.send(
      new PutObjectCommand({
        Bucket: bucketName,
        Key: objectKey,
        Body: fs.createReadStream(filePath),
      })
    );
  }
}

export const s3Service = S3Service.getInstance();
