import { Suspense } from 'react';
import Link from 'next/link';
import { FileVideo } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { UploadForm } from '@/components/upload-form';

export default function Home() {
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
      <main className="flex-1">
        <section className="py-12 md:py-24 lg:py-32">
          <div className="container px-4 mx-auto space-y-12 text-center">
            <div className="space-y-4">
              <h1 className="text-3xl font-bold tracking-tighter sm:text-4xl md:text-5xl">
                Add Subtitles to Your Videos
              </h1>
              <p className="max-w-[700px] mx-auto text-gray-500 md:text-xl">
                Upload your video, and we'll automatically generate subtitles
                using AI. Track your process and download the result when it's
                ready.
              </p>
            </div>
            <div className="mx-auto max-w-2xl">
              <Suspense fallback={<div>Loading upload form...</div>}>
                <UploadForm />
              </Suspense>
            </div>
          </div>
        </section>
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
