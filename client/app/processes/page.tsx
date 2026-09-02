import { ProcessGallery } from '@/components/process-gallery';
import { AppShell } from '@/components/app-shell';

export default function ProcessesPage() {
  return (
    <AppShell>
      <div className="mx-auto w-full max-w-6xl px-4 py-10">
        <div className="mb-6">
          <h1 className="font-display text-3xl font-bold tracking-tight">Gallery</h1>
          <p className="mt-1.5 text-muted-foreground">
            Every video you have captioned — hover a finished one to preview it.
          </p>
        </div>
        <ProcessGallery />
      </div>
    </AppShell>
  );
}
