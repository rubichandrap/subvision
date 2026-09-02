import { ProcessDetails } from '@/components/process-details';
import { AppShell } from '@/components/app-shell';

export default async function ProcessPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <AppShell>
      <div className="mx-auto w-full max-w-4xl px-4 py-10">
        <ProcessDetails processId={id} />
      </div>
    </AppShell>
  );
}
