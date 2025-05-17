'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { CheckCircle, Clock, FileVideo, Loader2 } from 'lucide-react';

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

// This is a mock implementation - in a real app, this would come from your backend
type Process = {
  id: string;
  filename: string;
  status: 'processing' | 'completed';
  progress: number;
  createdAt: string;
};

export function ProcessList() {
  const [processes, setProcesses] = useState<Process[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Simulate fetching processes from the backend
    setTimeout(() => {
      // Check if there are any processes in localStorage
      const storedProcesses = localStorage.getItem('subvision_processes');

      if (storedProcesses) {
        setProcesses(JSON.parse(storedProcesses));
      } else {
        // If no processes exist, create some sample data
        const sampleProcesses: Process[] = [
          {
            id: 'process_sample1',
            filename: 'interview.mp4',
            status: 'completed',
            progress: 100,
            createdAt: new Date(Date.now() - 86400000).toISOString(),
          },
          {
            id: 'process_sample2',
            filename: 'presentation.mp4',
            status: 'processing',
            progress: 65,
            createdAt: new Date(Date.now() - 3600000).toISOString(),
          },
        ];

        setProcesses(sampleProcesses);
        localStorage.setItem(
          'subvision_processes',
          JSON.stringify(sampleProcesses)
        );
      }

      setLoading(false);
    }, 1000);
  }, []);

  if (loading) {
    return (
      <div className="flex justify-center items-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-teal-500" />
      </div>
    );
  }

  if (processes.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <FileVideo className="w-12 h-12 text-gray-300 mb-4" />
          <h3 className="text-lg font-medium">No processes found</h3>
          <p className="text-gray-500 text-center mt-2 mb-6">
            You haven't uploaded any videos for processing yet.
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
      {processes.map((process) => (
        <Card key={process.id}>
          <CardHeader className="pb-2">
            <div className="flex justify-between items-start">
              <div>
                <CardTitle className="text-lg">{process.filename}</CardTitle>
                <CardDescription>Process ID: {process.id}</CardDescription>
              </div>
              {process.status === 'completed' ? (
                <div className="flex items-center text-green-500 text-sm font-medium">
                  <CheckCircle className="w-4 h-4 mr-1" />
                  Completed
                </div>
              ) : (
                <div className="flex items-center text-amber-500 text-sm font-medium">
                  <Clock className="w-4 h-4 mr-1" />
                  Processing
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {process.status === 'processing' && (
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>Progress</span>
                  <span>{process.progress}%</span>
                </div>
                <Progress value={process.progress} className="h-2" />
              </div>
            )}
            <div className="text-sm text-gray-500 mt-2">
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
