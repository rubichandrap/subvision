'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { CheckCircle, Clock, FileVideo, Loader2, XCircle } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { IN_FLIGHT_STAGES, fetchJobs, type Process, type ProcessStage } from '@/lib/api';

const STAGE_LABEL: Record<ProcessStage, string> = {
  uploaded: 'Uploaded',
  transcribing: 'Transcribing',
  rendering: 'Rendering',
  done: 'Completed',
  failed: 'Failed',
};

function StageBadge({ stage }: { stage: ProcessStage }) {
  if (stage === 'done') {
    return (
      <div className="flex items-center text-green-500 text-sm font-medium">
        <CheckCircle className="w-4 h-4 mr-1" />
        Completed
      </div>
    );
  }
  if (stage === 'failed') {
    return (
      <div className="flex items-center text-red-500 text-sm font-medium">
        <XCircle className="w-4 h-4 mr-1" />
        Failed
      </div>
    );
  }
  return (
    <div className="flex items-center text-amber-500 text-sm font-medium">
      <Loader2 className="w-4 h-4 mr-1 animate-spin" />
      {STAGE_LABEL[stage]}
    </div>
  );
}

export function ProcessList() {
  const [processes, setProcesses] = useState<Process[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    // Poll the status API while any process is still in flight, so the list
    // moves through the real pipeline stages instead of a fake progress bar.
    const poll = async () => {
      try {
        const jobs = await fetchJobs();
        if (cancelled) return;
        setProcesses(jobs);
        setError(null);
        if (jobs.some((p) => IN_FLIGHT_STAGES.has(p.stage))) {
          timer = setTimeout(poll, 3000);
        }
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Failed to load processes');
        timer = setTimeout(poll, 5000);
      }
    };

    poll();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, []);

  if (processes === null && !error) {
    return (
      <div className="flex justify-center items-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-teal-500" />
      </div>
    );
  }

  if (error && processes === null) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <FileVideo className="w-12 h-12 text-gray-300 mb-4" />
          <h3 className="text-lg font-medium">Could not load your processes</h3>
          <p className="text-gray-500 text-center mt-2">{error}</p>
        </CardContent>
      </Card>
    );
  }

  if (processes !== null && processes.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <FileVideo className="w-12 h-12 text-gray-300 mb-4" />
          <h3 className="text-lg font-medium">No processes found</h3>
          <p className="text-gray-500 text-center mt-2 mb-6">
            You haven&apos;t uploaded any videos for processing yet.
          </p>
          <Link href="/">
            <Button>Upload a Video</Button>
          </Link>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="grid gap-4">
      {(processes ?? []).map((process) => (
        <Card key={process.id}>
          <CardHeader className="pb-2">
            <div className="flex justify-between items-start">
              <div>
                <CardTitle className="text-lg">
                  {process.filename || process.id}
                </CardTitle>
                <CardDescription>Process ID: {process.id}</CardDescription>
              </div>
              <StageBadge stage={process.stage} />
            </div>
          </CardHeader>
          <CardContent>
            {process.stage === 'failed' && process.reason && (
              <div className="text-sm text-red-500 mt-1">{process.reason}</div>
            )}
            <div className="text-sm text-gray-500 mt-2 flex items-center">
              <Clock className="w-4 h-4 mr-1" />
              Started: {new Date(process.createdAt).toLocaleString()}
            </div>
          </CardContent>
          <CardFooter>
            <Link href={`/processes/${process.id}`} className="w-full">
              <Button variant="outline" className="w-full">
                View Details
              </Button>
            </Link>
          </CardFooter>
        </Card>
      ))}
    </div>
  );
}
