'use client';

import type React from 'react';

import { FileVideo, Upload } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import * as tus from 'tus-js-client';

import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { env } from '@/configs/env';
import { useToast } from '@/hooks/use-toast';

export function UploadForm() {
  const router = useRouter();
  const { toast } = useToast();
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [processId, setProcessId] = useState<string | null>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
    }
  };

  const handleUpload = async () => {
    if (!file) {
      toast({
        title: 'No file selected',
        description: 'Please select a video file to upload',
        variant: 'destructive',
      });
      return;
    }

    setUploading(true);
    setUploadProgress(0);

    console.log(`Starting upload to ${env.serverUrl}/files/`);

    // Create a new tus upload
    const upload = new tus.Upload(file, {
      endpoint: `${env.serverUrl}/files`,
      retryDelays: [0, 3000, 5000, 10000, 20000],
      metadata: {
        filename: file.name,
        filetype: file.type,
      },
      onError: (error) => {
        console.error('Upload failed:', error);
        toast({
          title: 'Upload failed',
          description: `Error: ${error.message || 'Unknown error'}`,
          variant: 'destructive',
        });
        setUploading(false);
      },
      onProgress: (bytesUploaded, bytesTotal) => {
        const percentage = (bytesUploaded / bytesTotal) * 100;
        setUploadProgress(percentage);
        console.log(`Upload progress: ${Math.round(percentage)}%`);
      },
      onSuccess: (payload) => {
        console.log("payload", payload);
        console.log('Upload completed successfully');

        // Get the upload URL which contains the file ID
        let uploadUrl = upload.url;
        console.log('Final upload URL:', uploadUrl)

        const fileId = uploadUrl ? uploadUrl.split('/').pop() : null;
        const processId =
          fileId || `process_${Math.random().toString(36).substring(2, 10)}`;

        setProcessId(processId);

        toast({
          title: 'Upload complete',
          description:
            'Your video has been uploaded and is now being processed.',
        });

        // Redirect to the process tracking page
        setTimeout(() => {
          router.push(`/processes/${processId}`);
        }, 1500);
      },
    });

    // Add event listeners for debugging
    upload.options.onAfterResponse = (req, res) => {
      const status = res.getStatus();
      const location = res.getHeader('Location');
      console.log(`Response: ${status} ${location || ''}`);

      // If we get a Location header with tusd:8080, log it but don't try to modify it directly
      // Our server should have already rewritten this header
      if (location && location.includes('tusd:8080')) {
        console.log(
          `Warning: Location header contains internal URL: ${location}`
        );
        console.log(`This should have been rewritten by the server.`);
      }
    };

    // Check if there are any previous uploads to continue
    try {
      const previousUploads = await upload.findPreviousUploads();
      if (previousUploads.length) {
        console.log('Found previous upload, resuming...');

        // Log the URL we're resuming from
        console.log('Resuming from URL:', previousUploads[0].uploadUrl);

        upload.resumeFromPreviousUpload(previousUploads[0]);
      }

      // Start the upload
      upload.start();
    } catch (error) {
      console.error('Error finding previous uploads:', error);
      // Start a new upload anyway
      upload.start();
    }
  };

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle>Upload Your Video</CardTitle>
        <CardDescription>
          We support MP4, MOV, AVI, and MKV formats up to 500MB.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div
          className="border-2 border-dashed rounded-lg p-8 text-center cursor-pointer hover:bg-gray-50 transition-colors"
          onClick={() => document.getElementById('file-upload')?.click()}
        >
          <input
            id="file-upload"
            type="file"
            accept="video/*"
            className="hidden"
            onChange={handleFileChange}
            disabled={uploading}
          />
          <FileVideo className="w-12 h-12 mx-auto text-gray-400 mb-4" />
          {file ? (
            <p className="text-sm font-medium">{file.name}</p>
          ) : (
            <p className="text-sm text-gray-500">
              Drag and drop your video here, or click to browse
            </p>
          )}
        </div>

        {uploading && (
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span>Uploading...</span>
              <span>{Math.round(uploadProgress)}%</span>
            </div>
            <Progress value={uploadProgress} className="h-2" />
          </div>
        )}
      </CardContent>
      <CardFooter>
        <Button
          className="w-full"
          onClick={handleUpload}
          disabled={!file || uploading}
        >
          {uploading ? (
            <span className="flex items-center gap-2">
              <Upload className="w-4 h-4 animate-pulse" />
              Uploading...
            </span>
          ) : (
            <span className="flex items-center gap-2">
              <Upload className="w-4 h-4" />
              Upload Video
            </span>
          )}
        </Button>
      </CardFooter>
    </Card>
  );
}
