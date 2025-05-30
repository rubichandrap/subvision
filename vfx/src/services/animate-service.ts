import { env } from "@/config/env";
import { objectOutputPrefix, objectUploadPrefix } from "@/config/storage";
import { renderImagesFromTemplate } from "@/render";
import { ISegment } from "@/types";
import { spawn } from "child_process";
import path from "path";
import { minioService } from "./minio/minio-service";

export class AnimateService {
  async create(objectKey: string, segments: ISegment[], animationType: string) {
    const id = objectKey.split("/").pop();

    if (!id) {
      throw new Error("Invalid object key");
    }

    // download the file
    const videoPath = path.join(process.cwd(), env.tmpDir, "videos");
    await minioService.downloadFile(
      env.minioBucket,
      `${objectUploadPrefix}/${id}`,
      videoPath
    );

    // render images
    const framePath = path.join(process.cwd(), env.tmpDir, "frames", id);
    await renderImagesFromTemplate(segments, animationType, framePath);

    // combine the image frames with the original video
    const outputPath = path.join(process.cwd(), env.tmpDir, "outputs");
    await this.combineImagesWithFFmpeg(videoPath, framePath, outputPath);

    // upload the video to minio
    await minioService.uploadFile(
      env.minioBucket,
      `${objectOutputPrefix}/${id}`,
      outputPath
    );
  }

  private async combineImagesWithFFmpeg(
    videoPath: string,
    framePath: string,
    outputPath: string
  ) {
    const ffmpeg = spawn("ffmpeg", [
      "-framerate",
      "30",
      "-i",
      path.join(framePath, "element-%03d.png"), // overlay
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
          resolve(outputPath);
        } else {
          reject(new Error(`FFmpeg process failed with code ${code}`));
        }
      });
    });
  }
}
