export interface IWord {
  text: string;
  start: number;
  end: number;
}

export interface ISegment {
  start: number;
  end: number;
  text: string;
  /** Timed Words transcribed by whisper, built server-side from its token timestamps. */
  words: IWord[];
}
