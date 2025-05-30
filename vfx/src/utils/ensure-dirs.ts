import { mkdirSync } from 'fs';

export function ensureDirs(...paths: string[]): void {
  for (const path of paths) {
    try {
      mkdirSync(path, { recursive: true });
    } catch (err) {
      console.error(`Failed to create directory ${path}:`, err);
      process.exit(1);
    }
  }
}
