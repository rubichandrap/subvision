import { ISegment } from "./types";

// One definition of the VFX Job contract for this runtime: the queue name and
// the payload shape the vfx service consumes. The server's publisher mirrors
// it in server/internal/vfxjob (Edit Spec validation in
// server/internal/editspec) — a change here must be made there too.

export const VFX_JOBS_QUEUE = "vfx_jobs";

// The queue a rejected or repeatedly failing VFX Job is dead-lettered into.
// The vfx_jobs queue carries dead-letter arguments routing here, so every
// declare of vfx_jobs must pass the same arguments.
export const VFX_JOBS_DEAD_QUEUE = "vfx_jobs_dead";

export const JOB_COMPLETED_QUEUE = "job_completed";

export const JOB_FAILED_QUEUE = "job_failed";

// ─── Edit Spec ─────────────────────────────────────────────────────────────
// The creative configuration the user set in the editor: trim window, target
// frame, animation, and subtitle style. The server validates it before
// publishing; this side parses it defensively again — a malformed job fails
// loudly, it is never silently reinterpreted. A job without an Edit Spec
// renders with the defaults below (the pre-editor behavior).

export const FRAME_PRESET_RATIOS: Record<string, number> = {
  "9:16": 9 / 16,
  "4:5": 4 / 5,
  "1:1": 1,
  "16:9": 16 / 9,
};

export const FRAME_PRESET_FREE = "free";

// The animations the templates registry can render. "random" is resolved
// client-side and never crosses the wire.
export const ANIMATIONS = ["fade", "slide", "karaoke", "pop"] as const;
export type Animation = (typeof ANIMATIONS)[number];

// Fonts bundled with the vfx service (via @fontsource); anything else would
// render in a fallback face.
export const FONT_FAMILIES = [
  "Montserrat",
  "Inter",
  "Poppins",
  "Oswald",
  "Bebas Neue",
  "Anton",
] as const;
export type FontFamily = (typeof FONT_FAMILIES)[number];

export interface Trim {
  /** Rendered window start, seconds. */
  start: number;
  /** Rendered window end, seconds; 0 renders to the end of the video. */
  end: number;
}

export interface Frame {
  /** Which Frame Preset the user picked; "free" carries a custom ratio. */
  preset: string;
  /** Target aspect ratio, width / height. */
  ratio: number;
  /** Cover zoom, 1..5. */
  zoom: number;
  /** Pan across the overflow, -1..1, positive toward the right/bottom. */
  panX: number;
  panY: number;
}

export interface SubtitleStyle {
  fontFamily: FontFamily;
  /** Relative to the frame height; 1 is the designed default. */
  fontSizeScale: number;
  color: string;
  /** Stroke width in px at a 1080-height reference; 0 = off. */
  outlineWidth: number;
  outlineColor: string;
  /** Distance from the frame bottom to the caption block, frame-height fraction. */
  bottomMargin: number;
  /** "none" renders raw text; "box" draws a rounded plate behind it. */
  background: "none" | "box";
  backgroundOpacity: number;
  uppercase: boolean;
  /** Swipe/active-word color of the karaoke and pop animations. */
  highlightColor: string;
}

export interface EditSpec {
  trim: Trim;
  frame: Frame;
  animation: Animation;
  style: SubtitleStyle;
}

export const DEFAULT_STYLE: SubtitleStyle = {
  fontFamily: "Montserrat",
  fontSizeScale: 1,
  color: "#FFFFFF",
  outlineWidth: 0,
  outlineColor: "#000000",
  bottomMargin: 0.12,
  background: "none",
  backgroundOpacity: 0.5,
  uppercase: false,
  highlightColor: "#FACC15",
};

export interface VfxJobPayload {
  uploadId: string;
  objectKey: string;
  segments: ISegment[];
  editSpec?: EditSpec;
}

export interface JobCompletedEvent {
  uploadId: string;
  outputKey: string;
}

export interface JobFailedEvent {
  uploadId: string;
  reason: string;
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

function isHexColor(value: unknown, field: string): void {
  if (typeof value !== "string" || !/^#[0-9A-Fa-f]{6}$/.test(value)) {
    throw new ContractError(`"${field}" must be a #RRGGBB color`);
  }
}

function inRange(
  value: unknown,
  field: string,
  min: number,
  max: number
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    value < min ||
    value > max
  ) {
    throw new ContractError(
      `"${field}" must be a number within [${min}, ${max}]`
    );
  }
  return value;
}

function parseEditSpec(value: unknown): EditSpec | undefined {
  if (value === undefined) return undefined;
  if (!isObject(value)) {
    throw new ContractError(`"editSpec" must be an object`);
  }

  const trim = value["trim"];
  if (!isObject(trim)) {
    throw new ContractError(`"editSpec.trim" must be an object`);
  }
  const start = trim["start"];
  const end = trim["end"];
  if (typeof start !== "number" || !Number.isFinite(start) || start < 0) {
    throw new ContractError(
      `"editSpec.trim.start" must be a non-negative number`
    );
  }
  if (
    typeof end !== "number" ||
    !Number.isFinite(end) ||
    (end !== 0 && end <= start)
  ) {
    throw new ContractError(
      `"editSpec.trim.end" must be 0 (to the end) or greater than trim.start`
    );
  }

  const frame = value["frame"];
  if (!isObject(frame)) {
    throw new ContractError(`"editSpec.frame" must be an object`);
  }
  const preset = frame["preset"];
  const ratio = frame["ratio"];
  if (typeof preset !== "string") {
    throw new ContractError(`"editSpec.frame.preset" must be a string`);
  }
  if (
    typeof ratio !== "number" ||
    !Number.isFinite(ratio) ||
    ratio <= 0 ||
    ratio > 3.5
  ) {
    throw new ContractError(
      `"editSpec.frame.ratio" must be a number within (0, 3.5]`
    );
  }
  const expected = FRAME_PRESET_RATIOS[preset];
  if (expected !== undefined && Math.abs(ratio - expected) > expected * 0.01) {
    throw new ContractError(
      `"editSpec.frame.ratio" ${ratio} does not match preset "${preset}" (${expected})`
    );
  }
  if (expected === undefined && preset !== FRAME_PRESET_FREE) {
    throw new ContractError(
      `"editSpec.frame.preset" must be one of 9:16, 4:5, 1:1, 16:9, free`
    );
  }

  const animation = value["animation"];
  if (
    typeof animation !== "string" ||
    !ANIMATIONS.includes(animation as Animation)
  ) {
    throw new ContractError(
      `"editSpec.animation" must be one of ${ANIMATIONS.join(", ")}`
    );
  }

  const style = value["style"];
  if (!isObject(style)) {
    throw new ContractError(`"editSpec.style" must be an object`);
  }
  const fontFamily = style["fontFamily"];
  if (
    typeof fontFamily !== "string" ||
    !FONT_FAMILIES.includes(fontFamily as FontFamily)
  ) {
    throw new ContractError(
      `"editSpec.style.fontFamily" must be one of ${FONT_FAMILIES.join(", ")}`
    );
  }
  isHexColor(style["color"], "editSpec.style.color");
  isHexColor(style["outlineColor"], "editSpec.style.outlineColor");
  isHexColor(style["highlightColor"], "editSpec.style.highlightColor");
  const background = style["background"];
  if (background !== "none" && background !== "box") {
    throw new ContractError(`"editSpec.style.background" must be "none" or "box"`);
  }

  return {
    trim: { start, end },
    frame: {
      preset,
      ratio,
      zoom: inRange(frame["zoom"], "editSpec.frame.zoom", 1, 5),
      panX: inRange(frame["panX"], "editSpec.frame.panX", -1, 1),
      panY: inRange(frame["panY"], "editSpec.frame.panY", -1, 1),
    },
    animation: animation as Animation,
    style: {
      fontFamily: fontFamily as FontFamily,
      fontSizeScale: inRange(
        style["fontSizeScale"],
        "editSpec.style.fontSizeScale",
        0.5,
        2
      ),
      color: style["color"] as string,
      outlineWidth: inRange(
        style["outlineWidth"],
        "editSpec.style.outlineWidth",
        0,
        32
      ),
      outlineColor: style["outlineColor"] as string,
      bottomMargin: inRange(
        style["bottomMargin"],
        "editSpec.style.bottomMargin",
        0,
        0.8
      ),
      background,
      backgroundOpacity: inRange(
        style["backgroundOpacity"],
        "editSpec.style.backgroundOpacity",
        0,
        1
      ),
      uppercase: style["uppercase"] === true,
      highlightColor: style["highlightColor"] as string,
    },
  };
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
  const editSpec = parseEditSpec(raw["editSpec"]);
  if (editSpec) {
    payload.editSpec = editSpec;
  }
  return payload;
}

// extractUploadId recovers the upload id from an unparsable payload when one
// is present, so a malformed job can still be reported as failed instead of
// leaving its process hanging in-flight forever.
export function extractUploadId(raw: unknown): string | undefined {
  if (isObject(raw) && typeof raw["uploadId"] === "string") {
    return raw["uploadId"];
  }
  return undefined;
}
