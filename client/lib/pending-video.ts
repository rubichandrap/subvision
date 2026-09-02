// Hands the chosen video from the home dropzone to the editor. File objects
// cannot survive a route change in storage, so they live in this module-level
// registry keyed by an id that travels in the URL; the editor takes the entry
// (consuming it) and owns the object URL from then on.

interface PendingVideo {
  file: File;
  previewUrl: string;
}

const pending = new Map<string, PendingVideo>();
const MAX_PENDING = 3;

export function setPendingVideo(file: File): string {
  const id = crypto.randomUUID();
  pending.set(id, { file, previewUrl: URL.createObjectURL(file) });
  while (pending.size > MAX_PENDING) {
    const oldest = pending.keys().next().value;
    if (oldest === undefined) break;
    discardPendingVideo(oldest);
  }
  return id;
}

export function takePendingVideo(id: string | null): PendingVideo | null {
  if (!id) return null;
  const entry = pending.get(id) ?? null;
  if (entry) pending.delete(id);
  return entry;
}

export function discardPendingVideo(id: string): void {
  const entry = pending.get(id);
  if (entry) {
    URL.revokeObjectURL(entry.previewUrl);
    pending.delete(id);
  }
}
