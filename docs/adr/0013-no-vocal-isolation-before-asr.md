# ADR-0013: No vocal isolation before decoding

Date: 2026-11-16 · Status: accepted (negative result)

## Context

Chunked decoding (ADR-0012) made Parakeet steadier over music but left wrong and missing sung words, with stretches of 30 s and more where no word is recognized. The next idea was to separate the vocals from the accompaniment first, with Demucs (`htdemucs`, vocals stem), and decode only the vocals.

Measured on the reporter's private 3:12 music video (media and transcripts stay local; only aggregates are recorded here). The original audio was extracted at 44.1 kHz stereo with the production flags, separated with Demucs 4.0.1 `htdemucs --two-stems=vocals`, and the vocals stem resampled to 16 kHz mono. The vocals stem is as loud as the mix at its peaks (−3.2 vs −2.5 dBFS), so level is not a confound. Both the mix and the vocals were decoded by the shipped worker code (`asr/worker` mounted into the `subvision-asr` image) at five start offsets (0, 2.7, 5.3, 8.2, 11.1 s), chunked (20 s / 4 s) and in a single pass. The mix decodes reproduce ADR-0012 exactly at the shared offsets (147 / 142 / 150 words chunked; 106 / 151 / 129 single pass).

Scoring follows ADR-0012: a word is corroborated when at least half of the other decodes contain the same word within 0.5 s. "Within" scores the five decodes of one variant against each other; "pooled" scores all 20 decodes against each other. Longest gap is the longest stretch with no recognized word, per decode.

| variant | words per offset | F1 within (range) | corroborated words | F1 pooled (range) | longest gap |
| --- | --- | --- | --- | --- | --- |
| mix, chunked (shipped) | 120–150 | 0.84 (0.81–0.86) | 126 | 0.72 (0.71–0.74) | 31–55 s |
| mix, single pass | 106–151 | 0.83 (0.73–0.89) | 126 | 0.72 (0.63–0.80) | 24–38 s |
| vocals, chunked | 114–166 | 0.79 (0.75–0.86) | 127 | 0.71 (0.67–0.77) | 22–54 s |
| vocals, single pass | 123–152 | 0.83 (0.76–0.86) | 129 | 0.71 (0.68–0.74) | 17–34 s |

Isolation does not let Parakeet hear more of the song. The vocals decodes find no more corroborated words than the mix (127–129 against 126), agree with the other decodes no better (pooled F1 0.71 against 0.72), and chunked vocals are less consistent than chunked mix (within F1 0.79 against 0.84; word count varies more, 114–166). Every range overlaps. What limits recognition here is how the model hears sung words, not the accompaniment around them.

The cost would be large. On the same 3:12 song, CPU only:

- Demucs took 101 s wall and 359 s CPU time limited to 4 CPUs (the worker's `ASR_THREADS` default), and 118 s wall / 786 s CPU when left to use all 12 cores of a busy machine; peak RSS was 1.7 GB both times. That is about 7 times the 15 s Parakeet needs to decode the song chunked, and on its own exceeds the server's default `PARAKEET_TIMEOUT_SECONDS` of 60 s before decoding starts.
- 1.7 GB of RAM on top of the Parakeet worker does not fit the 2–4 GB this deployment has free.
- The image would grow from 350 MB to about 2.25 GB (PyTorch CPU is 720 MB installed), plus 84 MB of `htdemucs` weights.
- Demucs code is MIT-licensed. The licence of the pretrained weights was not examined further, since nothing ships.

The long-speech video and the committed fixture were not re-measured: with no gain on the song there is nothing to weigh a speech regression against.

## Decision

Do not add vocal isolation to the ASR path. The worker keeps decoding the extracted mix, chunked as in ADR-0012. No setting is added.

## Alternatives considered

- **A stronger separator (`htdemucs_ft`, about 4x the compute of `htdemucs`).** Not measured. Cleaner stems would have to turn "no gain" into a clear gain to justify a cost already far past the timeout and the memory budget.
- **Isolation as an opt-in setting.** It would fit ADR-0010 as one path with a switch, but a switch that buys nothing measurable and costs minutes per song is not worth carrying.

## Consequences

- Wrong and missing sung words remain, as recorded in ADR-0012. Ensembling several chunkings is the next idea left to measure.
- The scratch measurement scripts were not committed; the method above is enough to repeat it.
