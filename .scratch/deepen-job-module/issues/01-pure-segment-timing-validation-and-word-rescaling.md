Promoted: #46

## Parent

Part of #45

## What to build

The domain rules for checking segment timing and scaling word timestamps into edited windows move into pure, in-process functions in the transcriber module alongside the Segment and Word types. Non-finite times, negative starts, non-positive durations, and overlapping or out-of-order segment bounds are rejected with clear validation errors. Word timestamps inside an edited segment are scaled proportionally into the new window while preserving original relative offsets.

## Acceptance criteria

- [x] `transcriber.ValidateSegmentTiming` rejects non-finite values, negative starts, `end <= start`, and non-chronological overlap with a typed error
- [x] Valid segment sequences pass validation without error
- [x] `transcriber.RescaleWords` proportionally scales `TimedWord` timestamps into the edited segment window based on original whisper timestamps
- [x] Words outside modified windows or with unchanged segment boundaries retain their original timestamps
- [x] Pure unit tests cover all validation boundaries and word scaling calculations without database or HTTP harnesses

## Blocked by

- None — can start immediately.
