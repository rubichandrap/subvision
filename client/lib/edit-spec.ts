// The client-side Edit Spec model: what the editor configures and what rides
// to the server as the "editSpec" tus metadata key. The wire shape mirrors
// vfx/src/contract.ts and server/internal/editspec; the builder only ever
// produces values inside the validated ranges, and resolves the "random"
// animation choice to one concrete animation before anything is serialized.

export type AnimationName = 'fade' | 'slide' | 'karaoke' | 'pop';
export type AnimationChoice = AnimationName | 'random';

export type FramePresetName = '9:16' | '4:5' | '1:1' | '16:9' | 'free';

export type CaptionFont =
  | 'Montserrat'
  | 'Inter'
  | 'Poppins'
  | 'Oswald'
  | 'Bebas Neue'
  | 'Anton';

export interface FrameState {
  preset: FramePresetName;
  /** width / height */
  ratio: number;
  zoom: number;
  panX: number;
  panY: number;
}

export interface TrimState {
  start: number;
  end: number;
}

export interface SubtitleStyleState {
  fontFamily: CaptionFont;
  fontSizeScale: number;
  color: string;
  outlineWidth: number;
  outlineColor: string;
  bottomMargin: number;
  background: 'none' | 'box';
  backgroundOpacity: number;
  uppercase: boolean;
  highlightColor: string;
}

export interface CaptionsState {
  wordsPerPage: number;
}

export interface EditSpecState {
  frame: FrameState;
  trim: TrimState;
  animation: AnimationChoice;
  style: SubtitleStyleState;
  captions: CaptionsState;
}

export const FONT_FAMILIES: CaptionFont[] = [
  'Montserrat',
  'Inter',
  'Poppins',
  'Oswald',
  'Bebas Neue',
  'Anton',
];

export const FRAME_PRESETS: Array<{ name: FramePresetName; ratio: number; label: string }> = [
  { name: '9:16', ratio: 9 / 16, label: 'Reels · TikTok · Shorts' },
  { name: '4:5', ratio: 4 / 5, label: 'Instagram feed' },
  { name: '1:1', ratio: 1, label: 'Square post' },
  { name: '16:9', ratio: 16 / 9, label: 'YouTube' },
];

export const ANIMATION_OPTIONS: Array<{ value: AnimationChoice; label: string; hint: string }> = [
  { value: 'fade', label: 'Fade', hint: 'Text fades in, holds, then fades out.' },
  { value: 'slide', label: 'Slide', hint: 'Text slides in from the left.' },
  { value: 'karaoke', label: 'Karaoke', hint: 'Each spoken word lights up in turn.' },
  { value: 'pop', label: 'Pop', hint: 'Words appear one by one, current word highlighted.' },
  { value: 'random', label: 'Random', hint: 'Picks one of the four when you generate.' },
];

export const COLOR_SWATCHES = [
  '#FFFFFF',
  '#000000',
  '#FACC15',
  '#A3E635',
  '#22D3EE',
  '#F472B6',
  '#F87171',
  '#4ADE80',
];

export const DEFAULT_WORDS_PER_PAGE = 4;
export const MIN_WORDS_PER_PAGE = 2;
export const MAX_WORDS_PER_PAGE = 8;

export const DEFAULT_EDIT_SPEC: EditSpecState = {
  frame: { preset: '9:16', ratio: 9 / 16, zoom: 1, panX: 0, panY: 0 },
  trim: { start: 0, end: 0 },
  animation: 'pop',
  captions: { wordsPerPage: DEFAULT_WORDS_PER_PAGE },
  style: {
    fontFamily: 'Montserrat',
    fontSizeScale: 1,
    color: '#FFFFFF',
    outlineWidth: 0,
    outlineColor: '#000000',
    bottomMargin: 0.12,
    background: 'none',
    backgroundOpacity: 0.5,
    uppercase: true,
    highlightColor: '#FACC15',
  },
};

export function resolveAnimation(choice: AnimationChoice): AnimationName {
  if (choice !== 'random') return choice;
  const animations: AnimationName[] = ['fade', 'slide', 'karaoke', 'pop'];
  return animations[Math.floor(Math.random() * animations.length)]!;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

export function formatTime(seconds: number): string {
  const total = Math.max(0, seconds);
  const minutes = Math.floor(total / 60);
  const rest = total - minutes * 60;
  return `${minutes}:${rest.toFixed(1).padStart(4, '0')}`;
}

// buildEditSpecPayload serializes the editor state into the metadata value
// the server validates. It carries the concrete ratio (for "free" frames the
// dragged one), the trim window, the resolved animation, the Caption Page
// size, and the full style — the contract requires every style field.
export function buildEditSpecPayload(state: EditSpecState): {
  animation: AnimationName;
  payload: string;
} {
  const trimEnd = state.trim.end > state.trim.start ? state.trim.end : 0;
  const spec = {
    trim: { start: Number(state.trim.start.toFixed(2)), end: Number(trimEnd.toFixed(2)) },
    frame: {
      preset: state.frame.preset,
      ratio: Number(state.frame.ratio.toFixed(4)),
      zoom: Number(clamp(state.frame.zoom, 1, 5).toFixed(2)),
      panX: Number(clamp(state.frame.panX, -1, 1).toFixed(3)),
      panY: Number(clamp(state.frame.panY, -1, 1).toFixed(3)),
    },
    animation: resolveAnimation(state.animation),
    captions: {
      wordsPerPage: Math.round(
        clamp(state.captions.wordsPerPage, MIN_WORDS_PER_PAGE, MAX_WORDS_PER_PAGE)
      ),
    },
    style: {
      fontFamily: state.style.fontFamily,
      fontSizeScale: Number(clamp(state.style.fontSizeScale, 0.5, 2).toFixed(2)),
      color: state.style.color,
      outlineWidth: Number(clamp(state.style.outlineWidth, 0, 32).toFixed(1)),
      outlineColor: state.style.outlineColor,
      bottomMargin: Number(clamp(state.style.bottomMargin, 0, 0.8).toFixed(3)),
      background: state.style.background,
      backgroundOpacity: Number(clamp(state.style.backgroundOpacity, 0, 1).toFixed(2)),
      uppercase: state.style.uppercase,
      highlightColor: state.style.highlightColor,
    },
  };
  return { animation: spec.animation, payload: JSON.stringify(spec) };
}