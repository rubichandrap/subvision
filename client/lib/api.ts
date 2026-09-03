import { env } from '@/configs/env';

// The client reads Process state only through the server's status API and
// downloads only via the API-provided URL — it invents nothing the server owns.

export type ProcessStage = 'uploaded' | 'transcribing' | 'rendering' | 'done' | 'failed';

export const IN_FLIGHT_STAGES: ReadonlySet<ProcessStage> = new Set([
  'uploaded',
  'transcribing',
  'rendering',
]);

export type Process = {
  id: string;
  filename: string;
  stage: ProcessStage;
  createdAt: string;
  updatedAt: string;
  downloadUrl?: string;
  reason?: string;
};

export class ProcessNotFoundError extends Error {
  constructor(id: string) {
    super(`no process with id "${id}"`);
    this.name = 'ProcessNotFoundError';
  }
}

type JSendSuccess<T> = { status: 'success'; data: T };

async function request<T>(path: string): Promise<T> {
  const res = await fetch(`${env.serverUrl}${path}`);
  if (res.status === 404) {
    throw new Error('not found');
  }
  if (!res.ok) {
    throw new Error(`status API returned ${res.status}`);
  }
  const body = (await res.json()) as JSendSuccess<T>;
  return body.data;
}

export async function fetchJobs(): Promise<Process[]> {
  const data = await request<{ jobs: Process[] }>('/jobs');
  return data.jobs;
}

// Job ids come from tusd's S3 store and contain a literal "+" (e.g.
// "<hex>+<base64>"). A "+" reaches fetchJob either raw or percent-encoded,
// depending on whether Next.js decoded the route param — normalizing here
// means the API is called exactly once-encoded no matter which it was.
function normalizeJobId(id: string): string {
  try {
    return decodeURIComponent(id);
  } catch {
    return id;
  }
}

export async function fetchJob(id: string): Promise<Process> {
  try {
    return await request<Process>(`/jobs/${encodeURIComponent(normalizeJobId(id))}`);
  } catch (error) {
    if (error instanceof Error && error.message === 'not found') {
      throw new ProcessNotFoundError(id);
    }
    throw error;
  }
}

// The URL the API provides for the rendered Output; the server streams it as
// an attachment.
export function downloadUrl(process: Process): string {
  if (!process.downloadUrl) {
    throw new Error(`process ${process.id} has no download URL yet`);
  }
  return `${env.serverUrl}${process.downloadUrl}`;
}
