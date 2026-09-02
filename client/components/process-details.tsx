'use client';

import Link from 'next/link';
import {
  ArrowLeft,
  CheckCircle,
  Download,
  FileVideo,
  Loader2,
  XCircle,
} from 'lucide-react';

import { StageBadge, STAGE_LABEL } from '@/components/process-stage';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { useProcess } from '@/hooks/use-process-polling';
import { IN_FLIGHT_STAGES, downloadUrl, type ProcessStage } from '@/lib/api';

// The pipeline stages, in the order the server moves a job through them.
const PIPELINE_STAGES: ProcessStage[] = [
  'uploaded',
  'transcribing',
  'rendering',
  'done',
];

export function ProcessDetails({ processId }: { processId: string }) {
  const { process, notFound, error } = useProcess(processId);

  const handleDownload = () => {
    if (!process?.downloadUrl) return;
    window.location.href = downloadUrl(process);
  };

  if (notFound) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <FileVideo className="w-12 h-12 text-gray-300 mb-4" />
          <h3 className="text-lg font-medium">Process not found</h3>
          <p className="text-gray-500 text-center mt-2 mb-6">
            The process ID you&apos;re looking for doesn&apos;t exist.
          </p>
          <Link href="/processes">
            <Button>View All Processes</Button>
          </Link>
        </CardContent>
      </Card>
    );
  }

  if (error && !process) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <FileVideo className="w-12 h-12 text-gray-300 mb-4" />
          <h3 className="text-lg font-medium">Could not load this process</h3>
          <p className="text-gray-500 text-center mt-2">{error}</p>
        </CardContent>
      </Card>
    );
  }

  if (!process) {
    return (
      <div className="flex justify-center items-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-teal-500" />
      </div>
    );
  }

  const inFlight = IN_FLIGHT_STAGES.has(process.stage);

  return (
    <div className="grid gap-6">
      <Link
        href="/processes"
        className="flex items-center text-sm text-teal-600 hover:underline"
      >
        <ArrowLeft className="w-4 h-4 mr-1" />
        Back to all processes
      </Link>

      <Card>
        <CardHeader>
          <div className="flex justify-between items-start">
            <div>
              <CardTitle>{process.filename || process.id}</CardTitle>
              <CardDescription>Process ID: {process.id}</CardDescription>
            </div>
            <StageBadge stage={process.stage} />
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {process.stage === 'failed' && process.reason && (
            <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-600">
              This process failed: {process.reason}
            </div>
          )}

          <div>
            <h3 className="text-sm font-medium mb-4">Pipeline Stages</h3>
            <div className="space-y-4">
              {PIPELINE_STAGES.map((stage) => {
                const reached =
                  PIPELINE_STAGES.indexOf(stage) <=
                  PIPELINE_STAGES.indexOf(process.stage);
                const isCurrent = stage === process.stage;
                return (
                  <div key={stage} className="flex items-center">
                    {isCurrent && inFlight && (
                      <Loader2 className="w-4 h-4 text-amber-500 animate-spin mr-2" />
                    )}
                    {isCurrent && process.stage === 'failed' && (
                      <XCircle className="w-4 h-4 text-red-500 mr-2" />
                    )}
                    {reached && !isCurrent && (
                      <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                    )}
                    {!reached && !isCurrent && (
                      <CheckCircle className="w-4 h-4 text-gray-300 mr-2" />
                    )}
                    <span
                      className={`text-sm ${
                        !reached && !isCurrent ? 'text-gray-400' : ''
                      }`}
                    >
                      {STAGE_LABEL[stage]}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="text-sm text-gray-500">
            Started: {new Date(process.createdAt).toLocaleString()}
          </div>
        </CardContent>
        <CardFooter>
          {process.stage === 'done' && process.downloadUrl && (
            <Button className="w-full" onClick={handleDownload}>
              <Download className="w-4 h-4 mr-2" />
              Download Video with Subtitles
            </Button>
          )}
          {inFlight && (
            <Button className="w-full" disabled>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Processing — check back later
            </Button>
          )}
          {process.stage === 'failed' && (
            <Button className="w-full" variant="destructive" disabled>
              <XCircle className="w-4 h-4 mr-2" />
              Processing Failed
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  );
}
