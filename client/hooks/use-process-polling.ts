'use client';

import { useEffect, useState } from 'react';

import { IN_FLIGHT_STAGES, ProcessNotFoundError, fetchJob, fetchJobs, type Process } from '@/lib/api';

// Polling against the status API: the client never invents state, it re-reads
// the server's while a job is in flight and stops once it reaches a terminal
// one. Both hooks schedule the next poll with a timeout chain, so there is
// exactly one request in the air at a time.

export function useProcessList() {
  const [processes, setProcesses] = useState<Process[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

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

  return { processes, error };
}

export function useProcess(id: string) {
  const [process, setProcess] = useState<Process | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let unknownAttempts = 0;

    const poll = async () => {
      try {
        const job = await fetchJob(id);
        if (cancelled) return;
        setProcess(job);
        setError(null);
        if (IN_FLIGHT_STAGES.has(job.stage)) {
          timer = setTimeout(poll, 2000);
        }
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ProcessNotFoundError) {
          // A 404 right after the upload may mean the server hasn't
          // recorded the process yet; retry briefly before giving up.
          unknownAttempts += 1;
          if (unknownAttempts >= 15) {
            setNotFound(true);
            return;
          }
          timer = setTimeout(poll, 2000);
          return;
        }
        setError(err instanceof Error ? err.message : 'Failed to load process');
        timer = setTimeout(poll, 5000);
      }
    };

    poll();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [id]);

  return { process, notFound, error };
}
