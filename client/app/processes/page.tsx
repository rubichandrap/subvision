import { ProcessGallery } from '@/components/process-gallery';
import { AppShell } from '@/components/app-shell';

export default function ProcessesPage() {
  return (
    <AppShell>
      <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
        <div className="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end border-b-2 border-border pb-6">
          <div>
            <div className="inline-flex items-center gap-2 border-2 border-border bg-primary px-2.5 py-1 font-mono text-xs font-bold uppercase tracking-widest text-primary-foreground shadow-brutal-sm">
              <span>Archive</span>
            </div>
            <h1 className="mt-3 font-display text-3xl font-extrabold tracking-tight sm:text-4xl">
              Video Gallery
            </h1>
            <p className="mt-1.5 text-sm text-muted-foreground">
              Track rendering jobs, watch finished videos, or download the MP4s.
            </p>
          </div>
        </div>
        <ProcessGallery />
      </div>
    </AppShell>
  );
}
