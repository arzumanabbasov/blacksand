import { fileURLToPath } from 'node:url';

// import.meta.filename is absent in Node 20.9 (the current E2B base image).
export function modulePath(meta) {
  return fileURLToPath(meta.url);
}
