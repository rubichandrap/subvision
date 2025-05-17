'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  ArrowLeft,
  CheckCircle,
  Clock,
  Download,
  FileVideo,
  Loader2,
} from 'lucide-react';

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
import { useToast } from '@/hooks/use-toast';

// This is a mock implementation - in a real app, this would come from your backend
type ProcessStage = {
  name: string;
  status: 'completed' | 'in_progress' | 'pending';
  startedAt?: string;
  completedAt?: string;
};

type ProcessDetails = {
  id: string;
  filename: string;
  status: 'processing' | 'completed';
  progress: number;
  createdAt: string;
  stages: ProcessStage[];
};

export function ProcessDetails({ processId }: { processId: string }) {
  const { toast } = useToast();
  const [process, setProcess] = useState<ProcessDetails | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Simulate fetching process details from the backend
    setTimeout(() => {
      // Check if there are any processes in localStorage
      const storedProcesses = localStorage.getItem('subvision_processes');
      let foundProcess = null;

      if (storedProcesses) {
        const processes = JSON.parse(storedProcesses);
        foundProcess = processes.find((p: any) => p.id === processId);
      }

      if (foundProcess) {
        // Add stages to the process
        foundProcess.stages = [
          {
            name: 'Video Upload',
            status: 'completed',
            startedAt: new Date(foundProcess.createdAt).toISOString(),
            completedAt: new Date(
              new Date(foundProcess.createdAt).getTime() + 120000
            ).toISOString(),
          },
          {
            name: 'Audio Extraction',
            status: foundProcess.progress >= 30 ? 'completed' : 'pending',
            startedAt:
              foundProcess.progress >= 30
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 180000
                  ).toISOString()
                : undefined,
            completedAt:
              foundProcess.progress >= 30
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 240000
                  ).toISOString()
                : undefined,
          },
          {
            name: 'Speech Recognition',
            status:
              foundProcess.progress >= 60
                ? 'completed'
                : foundProcess.progress >= 30
                  ? 'in_progress'
                  : 'pending',
            startedAt:
              foundProcess.progress >= 30
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 300000
                  ).toISOString()
                : undefined,
            completedAt:
              foundProcess.progress >= 60
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 420000
                  ).toISOString()
                : undefined,
          },
          {
            name: 'Subtitle Generation',
            status:
              foundProcess.progress >= 80
                ? 'completed'
                : foundProcess.progress >= 60
                  ? 'in_progress'
                  : 'pending',
            startedAt:
              foundProcess.progress >= 60
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 480000
                  ).toISOString()
                : undefined,
            completedAt:
              foundProcess.progress >= 80
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 540000
                  ).toISOString()
                : undefined,
          },
          {
            name: 'Video Rendering',
            status:
              foundProcess.progress >= 100
                ? 'completed'
                : foundProcess.progress >= 80
                  ? 'in_progress'
                  : 'pending',
            startedAt:
              foundProcess.progress >= 80
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 600000
                  ).toISOString()
                : undefined,
            completedAt:
              foundProcess.progress >= 100
                ? new Date(
                    new Date(foundProcess.createdAt).getTime() + 720000
                  ).toISOString()
                : undefined,
          },
        ];

        setProcess(foundProcess);
      } else {
        // If no process exists, create a sample one
        const now = new Date();
        const sampleProcess: ProcessDetails = {
          id: processId,
          filename: 'sample_video.mp4',
          status: Math.random() > 0.5 ? 'completed' : 'processing',
          progress:
            Math.random() > 0.5 ? 100 : Math.floor(Math.random() * 80) + 10,
          createdAt: new Date(now.getTime() - 3600000).toISOString(),
          stages: [],
        };

        // Add stages to the process
        sampleProcess.stages = [
          {
            name: 'Video Upload',
            status: 'completed',
            startedAt: new Date(now.getTime() - 3600000).toISOString(),
            completedAt: new Date(now.getTime() - 3480000).toISOString(),
          },
          {
            name: 'Audio Extraction',
            status: sampleProcess.progress >= 30 ? 'completed' : 'pending',
            startedAt:
              sampleProcess.progress >= 30
                ? new Date(now.getTime() - 3420000).toISOString()
                : undefined,
            completedAt:
              sampleProcess.progress >= 30
                ? new Date(now.getTime() - 3360000).toISOString()
                : undefined,
          },
          {
            name: 'Speech Recognition',
            status:
              sampleProcess.progress >= 60
                ? 'completed'
                : sampleProcess.progress >= 30
                  ? 'in_progress'
                  : 'pending',
            startedAt:
              sampleProcess.progress >= 30
                ? new Date(now.getTime() - 3300000).toISOString()
                : undefined,
            completedAt:
              sampleProcess.progress >= 60
                ? new Date(now.getTime() - 3180000).toISOString()
                : undefined,
          },
          {
            name: 'Subtitle Generation',
            status:
              sampleProcess.progress >= 80
                ? 'completed'
                : sampleProcess.progress >= 60
                  ? 'in_progress'
                  : 'pending',
            startedAt:
              sampleProcess.progress >= 60
                ? new Date(now.getTime() - 3120000).toISOString()
                : undefined,
            completedAt:
              sampleProcess.progress >= 80
                ? new Date(now.getTime() - 3060000).toISOString()
                : undefined,
          },
          {
            name: 'Video Rendering',
            status:
              sampleProcess.progress >= 100
                ? 'completed'
                : sampleProcess.progress >= 80
                  ? 'in_progress'
                  : 'pending',
            startedAt:
              sampleProcess.progress >= 80
                ? new Date(now.getTime() - 3000000).toISOString()
                : undefined,
            completedAt:
              sampleProcess.progress >= 100
                ? new Date(now.getTime() - 2880000).toISOString()
                : undefined,
          },
        ];

        setProcess(sampleProcess);

        // Store this process in localStorage
        if (storedProcesses) {
          const processes = JSON.parse(storedProcesses);
          processes.push(sampleProcess);
          localStorage.setItem(
            'subvision_processes',
            JSON.stringify(processes)
          );
        } else {
          localStorage.setItem(
            'subvision_processes',
            JSON.stringify([sampleProcess])
          );
        }
      }

      setLoading(false);

      // If the process is still in progress, simulate progress updates
      if (
        foundProcess &&
        foundProcess.status === 'processing' &&
        foundProcess.progress < 100
      ) {
        const interval = setInterval(() => {
          setProcess((prev) => {
            if (!prev || prev.progress >= 100) {
              clearInterval(interval);
              return prev;
            }

            const newProgress = Math.min(prev.progress + 5, 100);
            const newProcess: ProcessDetails = {
              ...prev,
              progress: newProgress,
              status: newProgress >= 100 ? 'completed' : 'processing',
            };

            // Update stages based on progress
            newProcess.stages = newProcess.stages.map((stage) => {
              if (
                stage.name === 'Audio Extraction' &&
                newProgress >= 30 &&
                stage.status !== 'completed'
              ) {
                return {
                  ...stage,
                  status: 'completed',
                  completedAt: new Date().toISOString(),
                };
              }
              if (stage.name === 'Speech Recognition') {
                if (newProgress >= 30 && stage.status === 'pending') {
                  return {
                    ...stage,
                    status: 'in_progress',
                    startedAt: new Date().toISOString(),
                  };
                }
                if (newProgress >= 60 && stage.status !== 'completed') {
                  return {
                    ...stage,
                    status: 'completed',
                    completedAt: new Date().toISOString(),
                  };
                }
              }
              if (stage.name === 'Subtitle Generation') {
                if (newProgress >= 60 && stage.status === 'pending') {
                  return {
                    ...stage,
                    status: 'in_progress',
                    startedAt: new Date().toISOString(),
                  };
                }
                if (newProgress >= 80 && stage.status !== 'completed') {
                  return {
                    ...stage,
                    status: 'completed',
                    completedAt: new Date().toISOString(),
                  };
                }
              }
              if (stage.name === 'Video Rendering') {
                if (newProgress >= 80 && stage.status === 'pending') {
                  return {
                    ...stage,
                    status: 'in_progress',
                    startedAt: new Date().toISOString(),
                  };
                }
                if (newProgress >= 100 && stage.status !== 'completed') {
                  return {
                    ...stage,
                    status: 'completed',
                    completedAt: new Date().toISOString(),
                  };
                }
              }
              return stage;
            });

            // Update the process in localStorage
            const storedProcesses = localStorage.getItem('subvision_processes');
            if (storedProcesses) {
              const processes = JSON.parse(storedProcesses);
              const updatedProcesses = processes.map((p: any) =>
                p.id === newProcess.id
                  ? {
                      ...p,
                      progress: newProgress,
                      status: newProgress >= 100 ? 'completed' : 'processing',
                    }
                  : p
              );
              localStorage.setItem(
                'subvision_processes',
                JSON.stringify(updatedProcesses)
              );
            }

            return newProcess;
          });
        }, 3000);

        return () => clearInterval(interval);
      }
    }, 1000);
  }, [processId]);

  const handleDownload = () => {
    toast({
      title: 'Download started',
      description: 'Your video with subtitles is being downloaded.',
    });

    // In a real implementation, this would trigger a download from your backend
    // Here we're just simulating the action
    setTimeout(() => {
      toast({
        title: 'Download complete',
        description: 'Your video has been downloaded successfully.',
      });
    }, 2000);
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-teal-500" />
      </div>
    );
  }

  if (!process) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <FileVideo className="w-12 h-12 text-gray-300 mb-4" />
          <h3 className="text-lg font-medium">Process not found</h3>
          <p className="text-gray-500 text-center mt-2 mb-6">
            The process ID you're looking for doesn't exist.
          </p>
          <Link href="/processes">
            <Button>View All Processes</Button>
          </Link>
        </CardContent>
      </Card>
    );
  }

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
              <CardTitle>{process.filename}</CardTitle>
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
        <CardContent className="space-y-6">
          {process.status === 'processing' && (
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Overall Progress</span>
                <span>{process.progress}%</span>
              </div>
              <Progress value={process.progress} className="h-2" />
            </div>
          )}

          <div>
            <h3 className="text-sm font-medium mb-4">Processing Stages</h3>
            <div className="space-y-4">
              {process.stages.map((stage) => (
                <div
                  key={stage.name}
                  className="grid grid-cols-[1fr_auto] gap-2"
                >
                  <div>
                    <div className="flex items-center">
                      {stage.status === 'completed' ? (
                        <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                      ) : stage.status === 'in_progress' ? (
                        <Loader2 className="w-4 h-4 text-amber-500 animate-spin mr-2" />
                      ) : (
                        <Clock className="w-4 h-4 text-gray-300 mr-2" />
                      )}
                      <span
                        className={`text-sm ${stage.status === 'pending' ? 'text-gray-500' : ''}`}
                      >
                        {stage.name}
                      </span>
                    </div>
                    {stage.startedAt && (
                      <div className="text-xs text-gray-500 ml-6 mt-1">
                        Started: {new Date(stage.startedAt).toLocaleString()}
                      </div>
                    )}
                    {stage.completedAt && (
                      <div className="text-xs text-gray-500 ml-6 mt-1">
                        Completed:{' '}
                        {new Date(stage.completedAt).toLocaleString()}
                      </div>
                    )}
                  </div>
                  <div className="text-xs text-gray-500">
                    {stage.status === 'completed'
                      ? 'Completed'
                      : stage.status === 'in_progress'
                        ? 'In Progress'
                        : 'Pending'}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="text-sm text-gray-500">
            Started: {new Date(process.createdAt).toLocaleString()}
          </div>
        </CardContent>
        <CardFooter>
          {process.status === 'completed' ? (
            <Button className="w-full" onClick={handleDownload}>
              <Download className="w-4 h-4 mr-2" />
              Download Video with Subtitles
            </Button>
          ) : (
            <Button className="w-full" disabled>
              <Clock className="w-4 h-4 mr-2" />
              Processing in Progress
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  );
}
