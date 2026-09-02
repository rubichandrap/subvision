import { ISegment } from "./types";

// One definition of the VFX Job contract for this runtime: the queue name and
// the payload shape the vfx service consumes. The server's publisher mirrors
// it in server/internal/vfxjob — a change here must be made there too.

export const VFX_JOBS_QUEUE = "vfx_jobs";

export interface VfxJobPayload {
  uploadId: string;
  objectKey: string;
  segments: ISegment[];
  /**
   * Reserved by the contract; consumers must not act on it.
   */
  animationType?: string;
}

export class ContractError extends Error {
  constructor(detail: string) {
    super(`invalid vfx job payload: ${detail}`);
    this.name = "ContractError";
  }
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function requireString(
  source: Record<string, unknown>,
  field: string
): string {
  const value = source[field];
  if (typeof value !== "string" || value === "") {
    throw new ContractError(`"${field}" must be a non-empty string`);
  }
  return value;
}

function parseSegments(value: unknown): ISegment[] {
  if (!Array.isArray(value)) {
    throw new ContractError(`"segments" must be an array`);
  }
  return value.map((entry, index) => {
    if (!isObject(entry)) {
      throw new ContractError(`segments[${index}] must be an object`);
    }
    const { start, end, text } = entry;
    if (typeof start !== "number" || !Number.isFinite(start)) {
      throw new ContractError(`segments[${index}].start must be a number`);
    }
    if (typeof end !== "number" || !Number.isFinite(end)) {
      throw new ContractError(`segments[${index}].end must be a number`);
    }
    if (typeof text !== "string") {
      throw new ContractError(`segments[${index}].text must be a string`);
    }
    return { start, end, text };
  });
}

// parseVfxJob validates a raw message body against the contract and throws a
// ContractError naming the violation — a malformed job fails loudly, it is
// never silently reinterpreted.
export function parseVfxJob(raw: unknown): VfxJobPayload {
  if (!isObject(raw)) {
    throw new ContractError("expected a JSON object");
  }
  const payload: VfxJobPayload = {
    uploadId: requireString(raw, "uploadId"),
    objectKey: requireString(raw, "objectKey"),
    segments: parseSegments(raw["segments"]),
  };
  if (typeof raw["animationType"] === "string") {
    payload.animationType = raw["animationType"];
  }
  return payload;
}
