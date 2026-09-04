'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import * as tus from 'tus-js-client';
import { ArrowLeft, Clapperboard, FileVideo2, Loader2, Wand2 } from 'lucide-react';

import { AnimationPicker } from '@/components/editor/animation-picker';
import { FramePicker } from '@/components/editor/frame-picker';
import { FramePreview } from '@/components/editor/frame-preview';
import { StylePanel } from '@/components/editor/style-panel';
import { TrimTimeline } from '@/components/editor/trim-timeline';
import { AppShell } from '@/components/app-shell';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useToast } from '@/hooks/use-toast';
import { env } from '@/configs/env';
import { formatTime, buildEditSpecPayload, DEFAULT_EDIT_SPEC, type EditSpecState } from '@/lib/edit-spec';import { takePendingVideo } from '@/lib/pending-video';

// The editor: one video, four decisions (Frame, Trim, Animation, Style), one
// upload. Nothing renders here — the preview is the browser compositor; the
// server applies the same numbers in its single render pass.

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  return `${Math.round(bytes / (1024 * 1024))} MB`;
}

function Editor() {
  const router = useRouter();
  const { toast } = useToast();
  const searchParams = useSearchParams();
  const videoRef = React.useRef<HTMLVideoElement | null>(null);

  const [video, setVideo] = React.useState<{ file: File; previewUrl: string } | null>(null);
  const [missing, setMissing] = React.useState(false);
  const [duration, setDuration] = React.useState(0);
  const [playhead, setPlayhead] = React.useState(0);
  const [spec, setSpec] = React.useState<EditSpecState>(DEFAULT_EDIT_SPEC);
  const [uploadProgress, setUploadProgress] = React.useState<number | null>(null);

  React.useEffect(() => {
    const pending = takePendingVideo(searchParams.get('file'));
    if (pending) {
      setVideo({ file: pending.file, previewUrl: pending.previewUrl });
    } else {
      setMissing(true);
    }
  }, [searchParams]);

  // The preview loops inside the trim window.
  const handleTimeUpdate = (event: React.SyntheticEvent<HTMLVideoElement>) => {
    const time = event.currentTarget.currentTime;
    setPlayhead(time);
    if (spec.trim.end > spec.trim.start && time >= spec.trim.end) {
      event.currentTarget.currentTime = spec.trim.start;
    }
  };

  const handleLoadedMetadata = (event: React.SyntheticEvent<HTMLVideoElement>) => {
    const videoDuration = event.currentTarget.duration;
    if (Number.isFinite(videoDuration) && videoDuration > 0) {
      setDuration(videoDuration);
      setSpec((current) => ({ ...current, trim: { start: 0, end: videoDuration } }));
    }
  };

  const patchSpec = (changes: Partial<EditSpecState>) =>
    setSpec((current) => ({ ...current, ...changes }));

  const uploading = uploadProgress !== null;

  const handleGenerate = () => {
    if (!video || uploading) return;
    const videoElement = videoRef.current;
    if (videoElement) videoElement.pause();

    const { payload } = buildEditSpecPayload(spec);

    const upload = new tus.Upload(video.file, {
      // The Edit Spec rides as tus metadata; tus-js-client base64-encodes the
      // value and the server decodes it into plain JSON.
      endpoint: `${env.serverUrl}/files`,
      retryDelays: [0, 3000, 5000, 10000, 20000],
      metadata: {
        filename: video.file.name,
        filetype: video.file.type || 'video/mp4',
        editSpec: payload,
      },
      onError: (error) => {
        setUploadProgress(null);
        toast({
          title: 'Upload failed',
          description: error.message || 'Something went wrong while uploading.',
          variant: 'destructive',
        });
      },
      onProgress: (bytesUploaded, bytesTotal) => {
        setUploadProgress((bytesUploaded / bytesTotal) * 100);
      },
      onSuccess: () => {
        setUploadProgress(null);
        const fileId = upload.url ? upload.url.split('/').pop() : null;
        if (!fileId) {
          toast({
            title: 'Upload finished, but tracking is unavailable',
            description: 'The server did not return a process id for this upload.',
            variant: 'destructive',
          });
          return;
        }
        toast({
          title: 'Upload complete',
          description: 'Your video is in the pipeline. Watch it in the gallery.',
        });
        router.push(`/processes/${fileId}`);
      },
    });

    setUploadProgress(0);
    upload.start();
  };

  if (missing) {
    return (
      <AppShell>
        <div className="mx-auto flex w-full max-w-lg flex-col items-center px-4 py-24 text-center">
          <Clapperboard className="h-10 w-10 text-muted-foreground" />
          <h1 className="mt-4 font-display text-2xl font-semibold">
            That video is no longer in this session
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            The editor keeps your selection in memory only. Pick the video
            again to start fresh.
          </p>
          <Button asChild className="mt-6">
            <Link href="/">
              <ArrowLeft className="h-4 w-4" />
              Choose a video
            </Link>
          </Button>
        </div>
      </AppShell>
    );
  }

  if (!video) {
    return (
      <AppShell>
        <div className="flex items-center justify-center py-32">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </AppShell>
    );
  }

  return (
    <AppShell>
      <div className="mx-auto w-full max-w-6xl px-4 pb-10 pt-6">
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <Button variant="ghost" size="icon" asChild aria-label="Back to home">
            <Link href="/">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div className="min-w-0">
            <h1 className="truncate font-display text-lg font-semibold leading-tight">
              {video.file.name}
            </h1>
            <p className="text-xs text-muted-foreground">
              {formatBytes(video.file.size)}
              {duration > 0 && <> · {formatTime(duration)}</>}
              {spec.trim.end > spec.trim.start &&
                duration > 0 &&
                spec.trim.end - spec.trim.start < duration &&
                <> · trimmed to {formatTime(spec.trim.end - spec.trim.start)}</>}
            </p>
          </div>
          <div className="ml-auto flex items-center gap-3">
            {uploadProgress !== null && (
              <div className="hidden w-40 sm:block">
                <Progress value={uploadProgress} />
              </div>
            )}
            <Button onClick={handleGenerate} disabled={uploading || duration <= 0}>
              {uploading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Uploading {Math.round(uploadProgress)}%
                </>
              ) : (
                <>
                  <Wand2 className="h-4 w-4" />
                  Generate subtitles
                </>
              )}
            </Button>
          </div>
        </div>

        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="flex min-h-[420px] flex-col gap-4 lg:min-h-[560px]">
            <div
              className="flex min-h-0 flex-1 items-center justify-center rounded-2xl border border-border/60 p-4 sm:p-6"
              style={{
                backgroundImage:
                  'radial-gradient(circle at 1px 1px, var(--border) 1px, transparent 0)',
                backgroundSize: '24px 24px',
              }}
            >
              <FramePreview
                src={video.previewUrl}
                frame={spec.frame}
                style={spec.style}
                animation={spec.animation}
                videoRef={videoRef}
                onLoadedMetadata={handleLoadedMetadata}
                onTimeUpdate={handleTimeUpdate}
                onPan={(panX, panY) =>
                  patchSpec({ frame: { ...spec.frame, panX, panY } })
                }
                onFreeRatio={(ratio) =>
                  patchSpec({ frame: { ...spec.frame, preset: 'free', ratio } })
                }
                disabled={uploading}
              />
            </div>
            <TrimTimeline
              duration={duration}
              trim={spec.trim}
              playhead={playhead}
              videoRef={videoRef}
              onTrimChange={(trim) => patchSpec({ trim })}
              disabled={uploading}
            />
          </div>

          <Card className="h-fit p-4 lg:sticky lg:top-24">
            <Tabs defaultValue="frame">
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="frame">Frame</TabsTrigger>
                <TabsTrigger value="animation">Animation</TabsTrigger>
                <TabsTrigger value="style">Style</TabsTrigger>
              </TabsList>
              <TabsContent value="frame" className="mt-4">
                <FramePicker
                  frame={spec.frame}
                  onChange={(frame) => patchSpec({ frame })}
                  disabled={uploading}
                />
              </TabsContent>
              <TabsContent value="animation" className="mt-4">
                <AnimationPicker
                  value={spec.animation}
                  onChange={(animation) => patchSpec({ animation })}
                  disabled={uploading}
                />
                <p className="mt-3 rounded-lg border border-border/60 bg-card/60 p-2.5 text-xs leading-relaxed text-muted-foreground">
                  The preview shows your caption style. Words appear once transcription finishes.
                </p>
              </TabsContent>
              <TabsContent value="style" className="mt-4">
                <StylePanel
                  style={spec.style}
                  animation={spec.animation}
                  onChange={(style) => patchSpec({ style })}
                  disabled={uploading}
                />
              </TabsContent>
            </Tabs>
            <div className="mt-4 flex items-center gap-2 border-t border-border/60 pt-3">
              <Badge variant="secondary" className="font-normal">
                <FileVideo2 className="mr-1 h-3 w-3" />
                {spec.frame.preset === 'free'
                  ? `${spec.frame.ratio.toFixed(2)}:1`
                  : spec.frame.preset}
              </Badge>
              <Badge variant="secondary" className="font-normal">
                {spec.animation === 'random' ? 'random animation' : spec.animation}
              </Badge>
              {spec.trim.end > spec.trim.start && (
                <Badge variant="secondary" className="font-normal">
                  {formatTime(spec.trim.start)}–{formatTime(spec.trim.end)}
                </Badge>
              )}
            </div>
          </Card>
        </div>
      </div>
    </AppShell>
  );
}

export default function EditorPage() {
  return (
    // useSearchParams needs a Suspense boundary during prerendering.
    <React.Suspense
      fallback={
        <AppShell>
          <div className="flex items-center justify-center py-32">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        </AppShell>
      }
    >
      <Editor />
    </React.Suspense>
  );
}
