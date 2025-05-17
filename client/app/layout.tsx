import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Subvision | The AI Subtitle Generator',
  description:
    "Upload your video, and we'll automatically generate subtitles using AI. Track your process and download the result when it's ready.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
