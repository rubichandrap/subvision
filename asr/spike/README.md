# ASR spike: Parakeet TDT v3

This is a disposable spike harness, **not** production code. It runs
`sherpa-onnx` NVIDIA Parakeet TDT 0.6B v3 int8 on a 16 kHz mono WAV and emits
JSON with Subvision-compatible `segments` plus the raw token timings used to
derive the words.

## Install

```sh
uv venv /tmp/opencode/asr-spike-venv --python 3.11
uv pip install --python /tmp/opencode/asr-spike-venv/bin/python \
  --index-url https://pypi.org/simple \
  -r asr/spike/requirements.txt
```

If `uv` uses a private index on your machine, set:

```sh
export UV_INDEX_URL=https://pypi.org/simple
```

## Model

Download the pinned Parakeet TDT 0.6B v3 int8 files to a directory outside the
repo:

```sh
mkdir -p /tmp/opencode/parakeet-tdt-0.6b-v3-int8
cd /tmp/opencode/parakeet-tdt-0.6b-v3-int8
for f in encoder.int8.onnx decoder.int8.onnx joiner.int8.onnx tokens.txt; do
  curl -L --fail --retry 3 -o "$f" \
    "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8/resolve/main/$f"
done
```

## Run

```sh
PARAKEET_MODEL_DIR=/tmp/opencode/parakeet-tdt-0.6b-v3-int8
/tmp/opencode/asr-spike-venv/bin/python asr/spike/parakeet.py \
  server/testdata/onset-fixture/fixture.wav \
  --model-dir "$PARAKEET_MODEL_DIR" \
  --output /tmp/opencode/parakeet-fixture.json
```

## Test your own video locally

Do **not** commit private media. Extract audio to a temporary file, run the
spike, then compare only the resulting aggregate metrics:

```sh
ffmpeg -hide_banner -y \
  -i /path/to/your-video.mp4 \
  -vn -ac 1 -ar 16000 -acodec pcm_s16le \
  /tmp/opencode/your-video.wav

/tmp/opencode/asr-spike-venv/bin/python asr/spike/parakeet.py \
  /tmp/opencode/your-video.wav \
  --model-dir "$PARAKEET_MODEL_DIR" \
  --output /tmp/opencode/your-video.json
```

To compare against the committed baseline transcript, use:

```sh
/tmp/opencode/asr-spike-venv/bin/python asr/spike/compare.py \
  server/testdata/onset-fixture/segments.json \
  /tmp/opencode/your-video.json \
  --reference asr/spike/reference.txt
```

The JSON contains:

- `text`: raw recognized text;
- `segments`: Subvision-shaped Segments with Timed Words;
- `tokens`: raw sherpa-onnx token timings;
- `words`: the grouped Timed Words used to build Segments.

## Notes

- This tool is English-first and does not translate.
- It does not touch production ingestion.
- Model weights and virtualenvs are intentionally kept outside the repo.
