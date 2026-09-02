// The object-key invariant from CONTEXT.md: keys are always <prefix>/<id>,
// the id being the last path segment. Derive keys from these helpers only.
export const objectUploadPrefix = "uploads/";
export const objectOutputPrefix = "outputs/";

export function uploadKey(id: string): string {
  return `${objectUploadPrefix}${id}`;
}

export function outputKey(id: string): string {
  return `${objectOutputPrefix}${id}`;
}
