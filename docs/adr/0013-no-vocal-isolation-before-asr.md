# ADR-0013: No vocal isolation before ASR

Date: 2026-11-16 · Status: accepted (negative result)

## Context

Chunked decoding (ADR-0012) made Parakeet steadier over music but left wrong and missing sung words, with stretches of 30 s and more where no word is recognized. The next idea was to separate the vocals from the accompaniment first, with Demucs (`htdemucs`, vocals stem), and decode only the vocals.

Measured on one song, the reporter's private 3:12 music video (media and transcripts stay local; only aggregates are recorded here). The original audio was extracted at 44.1 kHz stereo with the production flags, separated with Demucs 4.0.1 `htdemucs --two-stems=vocals`, and the vocals stem resampled to 16 kHz mono. The vocals stem is as loud as the mix at its peaks (−3.2 vs −2.5 dBFS), so level is not a confound. Both the mix and the vocals were decoded by the shipped worker code (`asr/worker` mounted into the `subvision-asr` image) at five start offsets (0, 2.7, 5.3, 8.2, 11.1 s), chunked (20 s / 4 s) and in a single pass. The mix decodes reproduce ADR-0012 exactly at its three offsets (147 / 142 / 150 words and 39 / 35 / 31 s longest gap chunked; 106 / 151 / 129 words single pass); the two extra offsets account for the wider ranges in the table.

Scoring follows ADR-0012: a word is corroborated when at least half of the other decodes contain the same word within 0.5 s. "Within" scores the five decodes of one variant against each other; "pooled" scores all 20 decodes (mix and vocals alike) against each other, so each variant is partly corroborated by the other. Longest gap is the longest stretch with no recognized word, per decode.

| variant | words per offset | F1 within (range) | corroborated words | F1 pooled (range) | longest gap |
| --- | --- | --- | --- | --- | --- |
| mix, chunked (shipped) | 120–150 | 0.84 (0.81–0.86) | 126 | 0.72 (0.71–0.74) | 31–55 s |
| mix, single pass | 106–151 | 0.83 (0.73–0.89) | 126 | 0.72 (0.63–0.80) | 24–38 s |
| vocals, chunked | 114–166 | 0.79 (0.75–0.86) | 127 | 0.71 (0.67–0.77) | 22–54 s |
| vocals, single pass | 123–152 | 0.83 (0.76–0.86) | 129 | 0.71 (0.68–0.74) | 17–34 s |

Isolation does not let Parakeet hear more of the song. The vocals decodes find no more corroborated words than the mix (127–129 against 126) and agree with the other decodes no better (pooled F1 0.71 against 0.72). Chunked vocals look somewhat less consistent than chunked mix (within F1 0.79 against 0.84; word count 114–166), but the ranges overlap, so that is a hint, not a finding. Vocals decodes do have shorter longest gaps in a single pass (17–34 s against 24–38 s), yet chunked they do not (22–54 s against 31–55 s), and the gaps move as much between start offsets as between variants. Whatever limits recognition here, removing the accompaniment did not lift it on this song. Only this one song was measured, so this says nothing about other music or about speech.

The cost would be large. On the same 3:12 song, CPU only:

- Demucs took 101 s wall and 359 s CPU time limited to 4 CPUs (the worker's `ASR_THREADS` default), and 118 s wall / 786 s CPU when left to use all 12 cores of a busy machine; peak RSS was 1.7 GB both times. That is about 7 times the 15 s Parakeet needs to decode the song chunked, and on its own exceeds the server's default `PARAKEET_TIMEOUT_SECONDS` of 60 s before decoding starts.
- 1.7 GB of RAM on top of the Parakeet worker does not fit the 2–4 GB this deployment has free.
- The scratch image with PyTorch (CPU), torchaudio, Demucs and ffmpeg was 2.25 GB, against 350 MB for `subvision-asr`; the `htdemucs` weights are a further 84 MB inside it.
- Demucs 4.0.1 declares the MIT licence. The licence of the pretrained `htdemucs` weights and any attribution they need were not checked, since nothing ships; anyone reopening this must check them first.

The long-speech video and the committed fixture were not re-measured: those checks guard against a regression after a win, and there was no win.

## Decision

Do not add vocal isolation to the ASR path, on the evidence of this one song. The worker keeps decoding the extracted mix, chunked as in ADR-0012. No setting is added.

## Amendment: opt-in switch

After this was recorded the user asked to be able to try isolation from the app, so it exists as an off-by-default switch (`ASR_VOCAL_ISOLATION` in the root `.env`; it builds Demucs into the ASR image and applies it in the worker). It is one path with one setting, as ADR-0010 allows, not a second engine. The decision above stands: the measurements do not support turning it on, and the 16 kHz mono input the app gives the worker is weaker than the 44.1 kHz stereo measured here (one run on the song gave 108 words, 69% corroborated by the mix decodes, and 1 wrong word in 22 on the clean-speech fixture, against 0 without isolation).

## Alternatives considered

- **A stronger separator (`htdemucs_ft`, about 4x the compute of `htdemucs`).** Not measured. Cleaner stems would have to turn "no gain" into a clear gain to justify a cost already far past the timeout and the memory budget.
- **Isolation as an opt-in setting.** It would fit ADR-0010 as one path with a switch, but a switch that buys nothing measurable and costs minutes per song is not worth carrying.

## Consequences

- Wrong and missing sung words remain, as recorded in ADR-0012. Ensembling several chunkings is the next idea left to measure.
- The scratch measurement scripts were not committed; the method above is enough to repeat it.
