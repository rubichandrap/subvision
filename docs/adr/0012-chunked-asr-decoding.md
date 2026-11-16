# ADR-0012: Decode long audio in overlapping chunks

Date: 2026-11-15 · Status: accepted · Issue: #78

## Context

#72 and #75 removed the Timing Drift on the reporter's music video, and they showed that Parakeet's word timestamps are sound: for words that a single pass and a chunked decode of the same audio both heard, start times differ by a median of 26–32 ms (over 1,086 song words and 2,766 speech words). What is left is what the model *hears*, and one decode pass over a long file is erratic.

Measured on two real local videos (the media and transcripts stay local; only aggregates are recorded here). Each decode was repeated on audio starting at different offsets, and every decode was scored against the others: a word is *corroborated* when at least half of the other decodes contain it within 0.5 s; precision is the share of a decode's words that are corroborated, recall the share of corroborated words it contains. There is no ground-truth transcript for the song, so this measures consistency, not accuracy.

3:12 music video, start offsets 0, 5.3 and 11.1 s:

| decode | words per offset | precision | recall | F1 (range) |
| --- | --- | --- | --- | --- |
| single pass | 106 / 151 / 129 | 0.62 | 0.63 | 0.62 (0.54–0.66) |
| 15 s chunks, 4 s overlap | 161 ± 3 | 0.68 | 0.88 | 0.77 (0.75–0.78) |
| **20 s, 4 s** | 147 / 142 / 150 | 0.73 | 0.85 | **0.79** (0.75–0.81) |
| 20 s, 8 s | 137 ± 21 | 0.75 | 0.81 | 0.78 (0.75–0.80) |
| 30 s, 4 s | 136 ± 2 | 0.72 | 0.77 | 0.74 (0.72–0.77) |
| 30 s, 8 s | 137 ± 2 | 0.76 | 0.81 | 0.78 (0.78–0.79) |

The single pass hears a different song depending on where the audio starts. Every chunked setting beats it, and their ranges overlap one another, so the exact numbers barely matter between 15 and 30 s. 20 s with 4 s overlap is the cheapest of the top group.

6:38 speech video: the single pass returned 945 words, 20 s/4 s chunks 995–1,003 and 30 s/8 s 1,010. Chunked decodes agreed with each other (text similarity 0.97–0.98) more than with the single pass (0.94–0.95), and only 7 of the 22 differing regions sat within 2 s of a join (about 5 would by chance). Where they differ, the single pass skips whole sentences (a stretch of about 15 s, and a list of places) and runs digits together ("gone17"); the chunked decode recovers them.

On the committed speech fixture, chunks of 6–10 s (many joins inside the speech) at six start offsets, 24 settings in all, gave no repeated or dropped word, no ordering violation, and start times within a mean 32 ms (max 173 ms) of the single pass. The only wording differences from the hand transcript were "Americans" heard as "America(n)" inside a chunk; the single pass does the same at some offsets.

Chunking is not a cure for music: the longest stretch with no recognized word is still 31–39 s on the song, and wrong words remain.

## Decision

The ASR Worker decodes audio longer than `ASR_CHUNK_SECONDS` (default 20) in the fewest equal chunks that overlap by at least `ASR_CHUNK_OVERLAP_SECONDS` (default 4), then stitches the words into one list. Audio that fits one chunk is decoded in one pass, exactly as before.

Stitching crosses over between neighbouring chunks at a word both heard, choosing the one nearest the middle of the overlap and taking that word from the chunk with more audio around it. Words are lined up by the cheapest in-order alignment: copies of a word (same text ignoring case and punctuation) pair at the cost of the time between their centers, and a word left unpaired costs 0.15 s, so copies pair only when closer than 0.3 s. Pricing the distance matters for runs of identical words ("no no no"), where counting matches alone prefers an alignment shifted by one instance. With no shared word it cuts at the middle.

Words from two chunks can meet a frame out of step. Where one starts inside the one before it, the earlier word gives way and the later word keeps its start, which is what a caption shows; if that would leave the earlier word under half a frame, the two share the overlap instead, because the Go client rejects a sentence whose only word has no duration. The list stays ordered with no overlap beyond the Go client's 1 ms tolerance.

Planning and stitching live in a pure-Python `chunking` module with the decode function injected; the tests use a fake decoder and need no model. The worker's request and its `segments` contract are unchanged, so the server needs no change. The response drops the raw `tokens` field (tokens cannot be stitched across chunks, and nothing read it) and gains `meta.chunks`. `ASR_CHUNK_SECONDS=0` turns chunking off, for comparing on your own files; it is the same engine and the same path with one setting changed, not a fallback in the sense of ADR-0010. A chunk is at least 2 s (or chunking is off), and the options are validated at startup.

## Alternatives considered

- **Single pass (status quo).** Erratic over music and skips sentences over long speech, as measured above.
- **Hard cuts with no overlap.** Every boundary cuts a word in half and gives both neighbours no context.
- **Cutting at silences.** Music has no reliable ones, and decoding only the speech windows found by silence detection was already tried and removed (ADR-0007).
- **Keeping whichever copy of a word sits farther from its chunk's edge.** Tried in tests first: one-frame words that two chunks time slightly differently survive twice. Crossing over at one shared word avoids that.
- **Keeping words only one chunk heard when the other heard nothing in the overlap.** Built, measured and removed. On the song those 80 words were the least corroborated in the word list (20% heard by no other decode against 9%; 38% strongly corroborated against 74%), so the chunk with more audio around a stretch is believed when it hears nothing.
- **Chunking in the Go server.** The worker owns inference (ADR-0010); keeping chunking behind its interface leaves the server and its contract untouched.
- **Vocal isolation before decoding.** Measured afterwards and not adopted: no gain on the song, at a large cost (ADR-0013).

## Consequences

- Overlap costs extra decoding, about `chunk / (chunk - overlap)` times the single pass: 1.25x with the defaults (1.23x measured on the song with two decoders sharing the CPU). Memory per pass is now bounded by the chunk, not the file.
- The server's `PARAKEET_TIMEOUT_SECONDS` (60 s by default) covers the whole file, so long videos need a larger value, about 25% more than before.
- The word list depends on the chunk grid, as the single pass already depended on where the audio starts; the defaults are measured, not guaranteed. Start times stay close to a single pass: on speech the median difference is 26-32 ms and almost all are within one model frame (80 ms).
- Stitching is not perfect on runs of identical words said back to back ("no no no"). In randomized tests with a decoder that hears every word exactly, unique words were never repeated or lost, while a vocabulary of two repeated words lost or repeated a word in about 0.1% of files, and with a decoder that mishears words at random about 4%. In the real decodes above no word was repeated or lost.
- Wrong and missing words over music remain. Ensembling several chunkings is the next thing to measure; vocal isolation was measured and did not help (ADR-0013).
