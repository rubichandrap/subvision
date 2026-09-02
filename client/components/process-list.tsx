'use client';

import Link from 'next/link';
import { Clock, FileVideo, Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { StageBadge } from '@/components/process-stage';
import { useProcessList } from '@/hooks/use-process-polling';

export function ProcessList() {
  const { processes, error } = useProcessList();

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
