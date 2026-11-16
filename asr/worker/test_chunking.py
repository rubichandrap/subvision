"""Tests for chunked decoding. No model, no numpy: a fake decoder stands in.

Run from the repository root:

    python3 -m unittest discover -s asr/worker -p 'test_*.py'
"""

from __future__ import annotations

import random
import unittest
from dataclasses import dataclass

from chunking import ChunkOptions, decode_chunked, plan_chunks, stitch

SAMPLE_RATE = 100  # fake rate: one "sample" is 10 ms, so every plan is exact
FRAME = 0.08  # the encoder frame the real model times words in


@dataclass(frozen=True)
class Word:
    text: str
    start: float
    end: float


def audio(seconds: float) -> list[int]:
    """Stand-in for a sample array: slices carry their own position."""
    return list(range(round(seconds * SAMPLE_RATE)))


def opts(chunk: float, overlap: float) -> ChunkOptions:
    return ChunkOptions(chunk_seconds=chunk, overlap_seconds=overlap)


def make_truth(seed: int, seconds: float) -> list[Word]:
    """Frame-aligned words: lyric-like runs, abutting words, long silences."""
    rng = random.Random(seed)
    words: list[Word] = []
    t = round(rng.uniform(0.5, 3.0) / FRAME) * FRAME
    index = 0
    while True:
        duration = rng.choice([1, 2, 2, 3, 4]) * FRAME
        if t + duration > seconds - 0.2:
            return words
        words.append(Word(f"w{index}", round(t, 2), round(t + duration, 2)))
        index += 1
        t += duration + rng.choice([0, 0, 0, 1, 1, 2, 4, 12, 40]) * FRAME


def garble(text: str) -> str:
    """A misheard word: never equal to any other word of the truth."""
    return text[::-1] + "~"


class FakeDecoder:
    """Hears the truth perfectly mid-chunk and badly near a chunk's edges.

    Within edge_zone of an edge that is not the true start or end of the
    audio, a word comes back garbled or not at all, the way a real decoder
    mishears a word cut in half. Every call also lands on its own frame-grid
    phase, so two chunks time the same word up to a frame apart.
    """

    def __init__(self, truth: list[Word], seconds: float, seed: int = 0, edge_zone: float = 0.6):
        self.truth = truth
        self.seconds = round(seconds * SAMPLE_RATE) / SAMPLE_RATE  # what audio() really holds
        self.edge_zone = edge_zone
        self.rng = random.Random(seed)
        self.calls: list[tuple[float, float]] = []

    def __call__(self, chunk: list[int]) -> list[Word]:
        start = chunk[0] / SAMPLE_RATE
        end = start + len(chunk) / SAMPLE_RATE
        self.calls.append((start, end))
        phase = self.rng.choice([-FRAME / 2, 0.0, FRAME / 2])
        heard: list[Word] = []
        for word in self.truth:
            if word.end <= start or word.start >= end:
                continue
            cut = word.start < start or word.end > end
            near_left = start > 1e-6 and word.start - start < self.edge_zone
            near_right = end < self.seconds - 1e-6 and end - word.end < self.edge_zone
            text = word.text
            lo, hi = word.start, word.end
            if cut:
                if self.rng.random() < 0.5:
                    continue
                text = garble(word.text)
                lo, hi = max(lo, start), min(hi, end)
            elif near_left or near_right:
                if self.rng.random() < 0.3:
                    continue
                text = garble(word.text)
            heard.append(Word(text, round(lo - start + phase, 4), round(hi - start + phase, 4)))
        return heard


def texts(words: list[Word]) -> list[str]:
    return [word.text for word in words]


def join(a: list[Word], b: list[Word]) -> list[Word]:
    """The usual two chunks: 0-10 s and 8-18 s, overlapping from 8 to 10 s."""
    return stitch([(0.0, 10.0, a), (8.0, 18.0, b)])


class PlanChunksTest(unittest.TestCase):
    def test_short_audio_is_one_chunk(self):
        self.assertEqual(plan_chunks(1500, SAMPLE_RATE, opts(20, 4)), [(0, 1500)])
        self.assertEqual(plan_chunks(2000, SAMPLE_RATE, opts(20, 4)), [(0, 2000)])

    def test_disabled_is_one_chunk(self):
        self.assertEqual(plan_chunks(50000, SAMPLE_RATE, opts(0, 0)), [(0, 50000)])

    def test_spans_cover_the_audio_with_the_overlap(self):
        for seconds in [20.01, 21.0, 35.99, 36.0, 36.01, 36.5, 60.0, 191.9, 600.0]:
            for chunk, overlap in [(20, 4), (10, 3), (30, 4), (15, 0), (20, 9.5)]:
                n = round(seconds * SAMPLE_RATE)
                spans = plan_chunks(n, SAMPLE_RATE, opts(chunk, overlap))
                lengths = [b - a for a, b in spans]
                with self.subTest(seconds=seconds, chunk=chunk, overlap=overlap):
                    self.assertEqual(spans[0][0], 0)
                    self.assertEqual(spans[-1][1], n)
                    self.assertLessEqual(max(lengths), round(chunk * SAMPLE_RATE))
                    for (a1, b1), (a2, b2) in zip(spans, spans[1:]):
                        self.assertGreater(a2, a1)
                        self.assertGreaterEqual(b1 - a2, round(overlap * SAMPLE_RATE))

    def test_chunks_are_even_and_no_more_numerous_than_needed(self):
        for seconds in [20.01, 36.5, 191.9]:
            n = round(seconds * SAMPLE_RATE)
            chunk, overlap = 2000, 400  # samples: 20 s and 4 s
            spans = plan_chunks(n, SAMPLE_RATE, opts(chunk / SAMPLE_RATE, overlap / SAMPLE_RATE))
            lengths = [b - a for a, b in spans]
            with self.subTest(seconds=seconds):
                self.assertLessEqual(max(lengths) - min(lengths), len(spans))
                fewer = len(spans) - 1
                self.assertLess(fewer * chunk - (fewer - 1) * overlap, n)

    def test_a_barely_long_file_gets_no_sliver_chunk(self):
        spans = plan_chunks(round(20.01 * SAMPLE_RATE), SAMPLE_RATE, opts(20, 4))
        self.assertEqual(len(spans), 2)
        self.assertEqual([b - a for a, b in spans], [1201, 1200])


class ChunkOptionsTest(unittest.TestCase):
    def test_rejects_nonsense(self):
        nan, inf = float("nan"), float("inf")
        for chunk, overlap in [
            (10, -1), (10, 5), (10, 6), (-5, 0),
            (nan, 0), (inf, 0), (10, nan), (10, inf),
            (1e-5, 0), (1.5, 0),  # too short to plan: a chunk is at least 2 s
        ]:
            with self.subTest(chunk=chunk, overlap=overlap):
                with self.assertRaises(ValueError):
                    opts(chunk, overlap)
        self.assertTrue(opts(2, 0.5).enabled)

    def test_zero_disables(self):
        self.assertFalse(opts(0, 0).enabled)
        self.assertFalse(opts(0, 7).enabled)
        self.assertTrue(opts(20, 4).enabled)


class DecodeChunkedTest(unittest.TestCase):
    def test_short_audio_decodes_once_untouched(self):
        decoder = FakeDecoder(make_truth(1, 15), 15)
        got = decode_chunked(audio(15), SAMPLE_RATE, decoder, opts(20, 4))
        self.assertEqual(len(decoder.calls), 1)
        self.assertEqual(got, FakeDecoder(make_truth(1, 15), 15)(audio(15)))

    def test_disabled_decodes_once(self):
        decoder = FakeDecoder(make_truth(2, 90), 90)
        decode_chunked(audio(90), SAMPLE_RATE, decoder, opts(0, 0))
        self.assertEqual(decoder.calls, [(0.0, 90.0)])

    def test_long_audio_is_decoded_in_overlapping_chunks(self):
        decoder = FakeDecoder(make_truth(3, 100), 100)
        decode_chunked(audio(100), SAMPLE_RATE, decoder, opts(20, 4))
        self.assertGreater(len(decoder.calls), 5)
        for (_, end), (start, _) in zip(decoder.calls, decoder.calls[1:]):
            self.assertGreaterEqual(end - start, 4 - 1e-9)
        self.assertTrue(all(end - start <= 20 + 1e-9 for start, end in decoder.calls))

    def test_stitched_words_equal_the_truth(self):
        for seed in range(40):
            seconds = random.Random(seed).uniform(25, 140)
            truth = make_truth(seed, seconds)
            for chunk, overlap in [(20, 4), (10, 3), (30, 4), (15, 2)]:
                decoder = FakeDecoder(truth, seconds, seed=seed)
                got = decode_chunked(audio(seconds), SAMPLE_RATE, decoder, opts(chunk, overlap))
                with self.subTest(seed=seed, chunk=chunk, overlap=overlap):
                    self.assertEqual(len(decoder.calls) > 1, seconds > chunk)
                    self.assertEqual(texts(got), texts(truth))

    def test_stitched_times_stay_within_a_frame_of_the_truth(self):
        for seed in range(20):
            seconds = random.Random(seed).uniform(25, 140)
            truth = make_truth(seed, seconds)
            decoder = FakeDecoder(truth, seconds, seed=seed)
            got = decode_chunked(audio(seconds), SAMPLE_RATE, decoder, opts(20, 4))
            with self.subTest(seed=seed):
                for want, have in zip(truth, got):
                    # one frame of grid phase, plus up to a frame of repair
                    # where two chunks' words meet
                    self.assertAlmostEqual(have.start, want.start, delta=FRAME + 1e-6)
                    self.assertAlmostEqual(have.end, want.end, delta=FRAME + 1e-6)

    def test_words_are_ordered_and_never_overlap(self):
        for seed in range(40):
            seconds = random.Random(seed).uniform(25, 140)
            decoder = FakeDecoder(make_truth(seed, seconds), seconds, seed=seed)
            got = decode_chunked(audio(seconds), SAMPLE_RATE, decoder, opts(20, 4))
            with self.subTest(seed=seed):
                for word in got:
                    self.assertLess(word.start, word.end)
                for before, after in zip(got, got[1:]):
                    self.assertLessEqual(before.end, after.start + 1e-9)


class StitchTest(unittest.TestCase):
    def test_nothing_to_stitch(self):
        self.assertEqual(stitch([]), [])
        words = [Word("a", 0.0, 0.2)]
        self.assertEqual(stitch([(0.0, 10.0, words)]), words)

    def test_words_without_an_overlap_are_concatenated(self):
        a = [Word("one", 1.0, 1.3)]
        b = [Word("two", 11.0, 11.3)]
        self.assertEqual(texts(stitch([(0.0, 10.0, a), (10.0, 20.0, b)])), ["one", "two"])

    def test_same_words_in_the_overlap_appear_once(self):
        a = [Word("one", 1.0, 1.3), Word("two", 9.0, 9.3), Word("three", 9.6, 9.9)]
        b = [Word("two", 9.0, 9.3), Word("three", 9.6, 9.9), Word("four", 14.0, 14.3)]
        got = join(a, b)
        self.assertEqual(texts(got), ["one", "two", "three", "four"])

    def test_same_word_timed_a_frame_apart_appears_once(self):
        a = [Word("love", 9.00, 9.08)]
        b = [Word("love", 9.08, 9.16)]
        self.assertEqual(texts(join(a, b)), ["love"])

    def test_case_and_punctuation_do_not_stop_words_matching(self):
        a = [Word("Love,", 9.0, 9.3), Word("me", 9.4, 9.6)]
        b = [Word("love", 9.0, 9.3), Word("me.", 9.4, 9.6), Word("now", 14.0, 14.3)]
        self.assertEqual(len(join(a, b)), 3)

    def test_repeated_words_stay_repeated(self):
        a = [Word("no", 9.0, 9.2), Word("no", 9.3, 9.5), Word("no", 9.6, 9.8)]
        b = [Word("no", 9.04, 9.24), Word("no", 9.34, 9.54), Word("no", 9.64, 9.84)]
        self.assertEqual(texts(join(a, b)), ["no", "no", "no"])

    def test_same_text_far_apart_in_time_is_not_the_same_word(self):
        a = [Word("love", 8.5, 8.8)]
        b = [Word("love", 9.5, 9.8)]  # a second "love", a second later, still inside the overlap
        self.assertEqual(texts(join(a, b)), ["love", "love"])

    def test_word_cut_by_a_chunk_edge_loses_to_the_chunk_that_heard_it_whole(self):
        a = [Word("lo", 9.7, 10.0)]  # cut by the end of the first chunk
        b = [Word("love", 9.7, 10.2)]
        self.assertEqual(texts(join(a, b)), ["love"])

    def test_a_word_both_chunks_heard_comes_from_the_one_with_more_context(self):
        a = [Word("do.", 10.0, 10.0)]  # zero length, at the end of the first chunk
        b = [Word("do.", 9.92, 10.08)]
        self.assertEqual(join(a, b), b)

    def test_each_side_of_the_overlap_comes_from_its_own_interior(self):
        a = [Word("a1", 8.2, 8.4), Word("a2", 9.8, 10.0)]
        b = [Word("b1", 8.2, 8.4), Word("b2", 9.8, 10.0)]
        self.assertEqual(texts(join(a, b)), ["a1", "b2"])

    def test_a_word_only_the_far_side_of_an_overlap_heard_is_dropped(self):
        # The other chunk has more audio around that stretch and heard nothing
        # there; measured on real music these are the least corroborated words.
        a = [Word("a1", 8.2, 8.4), Word("far", 9.5, 9.7)]
        b = [Word("later", 12.0, 12.3)]
        self.assertEqual(texts(join(a, b)), ["a1", "later"])
        a = [Word("early", 2.0, 2.3)]
        b = [Word("far", 8.4, 8.6), Word("b2", 9.4, 9.6)]
        self.assertEqual(texts(join(a, b)), ["early", "b2"])

    def test_copies_of_a_word_timed_a_few_tenths_apart_still_cross_over(self):
        # Real timings: the later chunk starts at 4.33 s and times the word
        # right at its own edge 0.2 s late. The chunks also disagree about the
        # next word ("America" or "Americans,"), so the shared word is the only anchor.
        a = [Word("fellow", 4.16, 4.64), Word("America", 4.64, 5.04)]
        b = [Word("Fellow", 4.49, 4.73), Word("Americans,", 4.81, 6.25), Word("ask", 6.25, 6.89)]
        got = stitch([(0.8, 6.33, a), (4.33, 9.87, b)])
        self.assertEqual(texts(got), ["fellow", "Americans,", "ask"])

    def test_a_run_of_identical_words_keeps_every_instance(self):
        # A "no" every 0.3 s. The second chunk starts just after the first one;
        # shifting an alignment by one instance would still look close in time.
        def run(ks, shift):
            return [Word("no", round(8.0 + 0.3 * k + shift, 2), round(8.2 + 0.3 * k + shift, 2)) for k in ks]

        for shift in (0.0, 0.04):
            got = stitch([(0.0, 10.0, run(range(0, 7), 0.0)), (8.05, 18.05, run(range(1, 21), shift))])
            self.assertEqual(len(got), 21, f"shift {shift}")
            for k, word in enumerate(got):
                self.assertAlmostEqual(word.start, 8.0 + 0.3 * k, delta=0.05, msg=f"word {k}, shift {shift}")

    def test_a_one_frame_word_that_two_chunks_time_a_frame_apart_keeps_a_duration(self):
        # The Go client rejects a sentence whose only word has no duration.
        a = [Word("Yes.", 8.60, 8.68)]
        b = [Word("Yes.", 8.52, 8.60), Word("I", 8.60, 8.68)]
        got = join(a, b)
        self.assertEqual(texts(got), ["Yes.", "I"])
        for word in got:
            self.assertGreater(word.end, word.start)
        for before, after in zip(got, got[1:]):
            self.assertLessEqual(before.end, after.start + 1e-9)

    def test_where_words_overlap_the_later_word_keeps_its_start(self):
        # A caption shows a word from its start, so the earlier word gives way.
        got = stitch([(0.0, 10.0, [Word("one", 8.0, 8.5), Word("two", 8.3, 8.6)])])
        self.assertEqual([(w.text, w.start, w.end) for w in got], [("one", 8.0, 8.3), ("two", 8.3, 8.6)])

    def test_a_short_word_inside_a_long_one_keeps_a_duration(self):
        a = [Word("mmm", 8.0, 8.9)]  # one chunk hears a long hum
        b = [Word("mmm", 8.0, 8.9), Word("hey", 8.6, 8.62)]  # the other puts a tiny word inside it
        got = join(a, b)
        for word in got:
            self.assertLessEqual(word.start, word.end)
        for before, after in zip(got, got[1:]):
            self.assertLessEqual(before.end, after.start + 1e-9)
        self.assertEqual(texts(got), ["mmm", "hey"])
        self.assertGreater(got[1].end, got[1].start)

    def test_words_that_arrive_out_of_order_still_come_out_valid(self):
        # Garbled input should never turn into a word that ends before it starts.
        a = [Word("a", 9.0, 9.4), Word("b", 8.2, 8.4)]  # the second ends before the first starts
        got = stitch([(0.0, 10.0, a)])
        self.assertEqual(texts(got), ["a", "b"])
        for word in got:
            self.assertLessEqual(word.start, word.end)
        for before, after in zip(got, got[1:]):
            self.assertLessEqual(before.end, after.start + 1e-9)

    def test_a_first_chunk_that_heard_nothing_follows_the_same_rule_as_any_other(self):
        x = [Word("x", 16.5, 16.8), Word("y", 19.0, 19.2)]  # "x" is in the half of the overlap nearest the later chunk's own edge
        self.assertEqual(texts(stitch([(0.0, 20.0, []), (16.0, 36.0, x)])), ["y"])
        self.assertEqual(
            texts(stitch([(0.0, 20.0, [Word("e", 1.0, 1.2)]), (16.0, 36.0, x)])), ["e", "y"]
        )

    def test_words_at_the_true_ends_of_the_audio_survive(self):
        a = [Word("first", 0.0, 0.2)]
        b = [Word("last", 17.8, 18.0)]
        self.assertEqual(texts(join(a, b)), ["first", "last"])

    def test_back_to_back_words_in_one_chunk_are_all_kept(self):
        a = [Word("x", 3.0, 3.0), Word("y", 3.0, 3.16), Word("z", 3.16, 3.32)]
        self.assertEqual(stitch([(0.0, 10.0, a)]), a)

    def test_where_two_chunks_meet_the_earlier_word_gives_way(self):
        a = [Word("a", 8.5, 8.8), Word("b", 9.0, 9.4)]  # b runs on to 9.4
        b = [Word("b", 9.0, 9.24), Word("c", 9.3, 9.6)]  # c starts at 9.3
        got = join(a, b)
        self.assertEqual(texts(got), ["a", "b", "c"])
        for before, after in zip(got, got[1:]):
            self.assertLessEqual(before.end, after.start + 1e-9)

    def test_three_chunks_overlapping_one_another(self):
        words = [Word(f"w{i}", i * 0.5, i * 0.5 + 0.3) for i in range(60)]
        chunks = [
            (0.0, 12.0, [w for w in words if w.end <= 12.0]),
            (6.0, 18.0, [w for w in words if 6.0 <= w.start and w.end <= 18.0]),
            (8.0, 30.0, [w for w in words if 8.0 <= w.start and w.end <= 30.0]),
        ]
        self.assertEqual(stitch(chunks), words)

    def test_input_is_not_mutated(self):
        a = [Word("a", 1.0, 1.2), Word("b", 9.0, 9.2)]
        b = [Word("b", 9.0, 9.2), Word("c", 12.0, 12.2)]
        keep = (list(a), list(b))
        join(a, b)
        self.assertEqual((a, b), keep)


if __name__ == "__main__":
    unittest.main()
