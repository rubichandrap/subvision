export interface IWord {
  text: string;
  start: number;
  end: number;
}

export interface ISegment {
  start: number;
  end: number;
  text: string;
  /** Timed Words returned by the ASR worker, including native token timings. */
  words: IWord[];
}
