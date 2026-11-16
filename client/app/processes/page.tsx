import { ProcessGallery } from '@/components/process-gallery';
import { AppShell } from '@/components/app-shell';

export default function ProcessesPage() {
  return (
    <AppShell>
      <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
        <div className="mb-10 sm:mb-12">
          <p className="rise inline-flex items-center gap-2 rounded-full border border-border bg-foreground/[0.04] px-3 py-1 text-xs font-medium text-muted-foreground">
            <span className="h-1.5 w-1.5 rounded-full bg-primary" />
            Archive
          </p>
          <h1 className="rise rise-2 mt-5 text-5xl font-semibold leading-[0.92] tracking-[-0.05em] sm:text-7xl">
            Your <span className="text-gradient">videos</span>
          </h1>
          <p className="rise rise-3 mt-4 max-w-lg text-base leading-relaxed text-muted-foreground sm:text-lg">
            Track rendering jobs, watch finished videos, or download the MP4s.
          </p>
        </div>
        <ProcessGallery />
      </div>
    </AppShell>
  );
}
