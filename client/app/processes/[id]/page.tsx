import { FileVideo } from 'lucide-react';
import Link from 'next/link';

import { ProcessDetails } from '@/components/process-details';
import { Button } from '@/components/ui/button';

export default function ProcessPage({ params }: { params: { id: string } }) {
  return (
    <div className="flex flex-col min-h-screen">
      <header className="border-b">
        <div className="container flex items-center h-16 px-4 mx-auto">
          <Link href="/" className="flex items-center gap-2 font-semibold">
            <FileVideo className="w-5 h-5 text-teal-500" />
            <span>Subvision</span>
          </Link>
          <nav className="ml-auto">
            <Link href="/processes">
              <Button variant="ghost">My Processes</Button>
            </Link>
          </nav>
        </div>
      </header>
      <main className="flex-1 container px-4 py-8 mx-auto">
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              Process Details
            </h1>
            <p className="text-gray-500">
              Track the status of your video subtitle generation process.
            </p>
          </div>
          <ProcessDetails processId={params.id} />
        </div>
      </main>
      <footer className="border-t">
        <div className="container flex flex-col items-center justify-between gap-4 py-6 px-4 mx-auto md:flex-row">
          <p className="text-sm text-gray-500">
            © 2025 Subvision. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
