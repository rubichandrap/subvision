import { ProcessGallery } from '@/components/process-gallery';
import { AppShell } from '@/components/app-shell';

export default function ProcessesPage() {
  return (
    <AppShell>
      <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
        <div className="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end border-b border-border/60 pb-6">
          <div>
            <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              <span className="h-1.5 w-1.5 rounded-full bg-primary" />
              <span>Studio Archive</span>
            </div>
            <h1 className="mt-1 font-display text-3xl font-bold tracking-tight sm:text-4xl">
              Video Gallery
            </h1>
            <p className="mt-1.5 text-sm text-muted-foreground">
              Manage your captioned videos, monitor in-flight rendering jobs, or download final MP4s.
            </p>
          </div>
        </div>
        <ProcessGallery />
      </div>
    </AppShell>
  );
}
