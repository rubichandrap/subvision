# ASR worker: Parakeet TDT v3

This is Subvision's production speech-recognition worker. It accepts a
`POST /transcribe` request with a 16 kHz mono PCM WAV body and returns
Subvision-compatible `segments` JSON. Inference runs through sherpa-onnx with
NVIDIA Parakeet TDT 0.6B v3 int8. The worker does not translate.

The worker is internal-only: compose publishes no host port for it. It has no
authentication, so never expose it to an untrusted network.

## Model files

Download the pinned int8 ONNX export into a host directory, or let compose mount
`./models/parakeet`:

```sh
mkdir -p models/parakeet
huggingface-cli download csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8 \
  --local-dir models/parakeet
```

Expected files and SHA-256 values:

```text
0222a86147299d4dee90edf95b6d13827edb42ad8b7c9cf0b2582f030937c0b5  decoder.int8.onnx
1afbe69c1ffdabee94755f42e4daf1d93549131a3da11356bda84fcef0f2c479  encoder.int8.onnx
cbd02f3696e9944970111fb439d9c3fd365a80374565f13f2b9c7007d253fc98  joiner.int8.onnx
d58544679ea4bc6ac563d1f545eb7d474bd6cfa467f0a6e2c1dc1c7d37e3c35d  tokens.txt
```

The model is **CC-BY-4.0** and requires attribution; see
[THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md). Do not commit weights.

## Run locally

```sh
uv venv /tmp/opencode/asr-worker-venv --python 3.11
uv pip install --python /tmp/opencode/asr-worker-venv/bin/python \
  --index-url https://pypi.org/simple \
  -r asr/worker/requirements.txt

/tmp/opencode/asr-worker-venv/bin/python asr/worker/main.py \
  --model-dir models/parakeet \
  --host 127.0.0.1 \
  --port 8081
```

Then run the server with:

```sh
PARAKEET_URL=http://127.0.0.1:8081/transcribe \
  sh server/scripts/dev.sh
```

## Smoke test

```sh
curl --fail --silent --show-error \
  --header 'Content-Type: audio/wav' \
  --data-binary @server/testdata/onset-fixture/fixture.wav \
  http://127.0.0.1:8081/transcribe \
  | jq '.segments'
```

## Docker

The standard compose stack includes the worker and the server dependency:

```sh
PARAKEET_MODEL_DIR=/path/to/parakeet-tdt-0.6b-v3-int8 \
  docker compose up --build
```

The server gets `PARAKEET_URL=http://asr:8081/transcribe` from
`docker-compose.yml`.

For direct host testing only, you can publish the port; note that the worker is
unauthenticated and must not be exposed to untrusted networks:

```sh
docker build -t subvision-asr-worker asr/worker
docker run --rm -p 127.0.0.1:8081:8081 \
  -v /path/to/parakeet-tdt-0.6b-v3-int8:/models:ro \
  subvision-asr-worker
```

## Long audio

One Parakeet pass over a long file is erratic, especially over music: which
sung words it hears changes with where the audio starts, and short passes are
steadier (measurements in
[ADR-0012](../../docs/adr/0012-chunked-asr-decoding.md)). So audio longer
than `ASR_CHUNK_SECONDS` is split into equal chunks that overlap by
`ASR_CHUNK_OVERLAP_SECONDS`, each chunk is decoded on its own, and the Timed
Words are stitched back into one list. Where two chunks overlap, the list
crosses over at a word both heard, so each side comes from the chunk with more
audio around it and a word is not heard twice.

- Audio that fits one chunk is decoded in one pass, as before.
- The request and the `segments` contract are unchanged. The response also
  carries `meta.chunks`, the number of passes, and each request logs its audio
  length, chunk count, word count and decode time to stderr. The raw `tokens`
  field is gone: tokens cannot be stitched across chunks and nothing read them.
- `ASR_CHUNK_SECONDS=0` decodes every file in one pass, which is useful for
  comparing against your own videos.
- Overlap costs extra decoding: roughly `chunk / (chunk - overlap)` times the
  single-pass compute, 1.25x with the defaults. Long videos may need a larger
  `PARAKEET_TIMEOUT_SECONDS` on the server.

## Vocal isolation (optional experiment)

[ADR-0013](../../docs/adr/0013-no-vocal-isolation-before-asr.md) measured
Demucs (`htdemucs`) vocal isolation before Parakeet and found no gain on one
song, so it is **off by default**. It exists so you can try it on your own
videos: with `ASR_VOCAL_ISOLATION=1` the worker separates the vocals from the
16 kHz mono audio it receives and decodes that stem instead.

- Set `ASR_VOCAL_ISOLATION=1` in the root `.env` (see `.env.example`). Compose
  passes it to the image build as `VOCAL_ISOLATION` (which installs CPU PyTorch,
  Demucs and bakes the 84 MB weights in, about 2 GB more) and to the worker at
  run time. Rebuild the image after changing it:

  ```sh
  docker build --network host --build-arg VOCAL_ISOLATION=1 -t subvision-asr asr/worker
  docker compose up -d --no-build asr
  ```

- It is slow: about 100 s and up to ~3 GB of RAM for a 3-minute song, and RAM
  grows with the length of the file. Raise `PARAKEET_TIMEOUT_SECONDS` in `server/.env` (e.g. `900`) or every
  song fails at the default 60 s.
- A worker started with it on but an image built without it exits at startup
  with an error saying so. `meta.vocalIsolation` in the response says whether
  it was applied.
- To turn it off, set `ASR_VOCAL_ISOLATION=0` (or remove it) and restart the
  worker; no rebuild is needed, but rebuild without it to get the small image
  back.
- Not a second engine: the same decode path runs, on the vocals stem.

## Environment

| Variable | Description |
| --- | --- |
| `PARAKEET_MODEL_DIR` | Directory containing the four model files. Set to `/models` in the container. |
| `PARAKEET_TIMEOUT_SECONDS` | Server-side request ceiling; set on the server, not the worker. |
| `ASR_MAX_BODY_BYTES` | Request body cap; default 256 MiB. |
| `ASR_HOST` / `ASR_PORT` | Bind address and port; defaults `0.0.0.0:8081`. |
| `ASR_THREADS` | ONNX runtime thread count. |
| `ASR_VOCAL_ISOLATION` | `1` separates the vocals with Demucs before decoding; default `0`. Needs an image built with `VOCAL_ISOLATION=1`. |
| `ASR_CHUNK_SECONDS` | Longest audio decoded in one pass; longer files are chunked. Default `20`; `0` turns chunking off, otherwise at least `2`. |
| `ASR_CHUNK_OVERLAP_SECONDS` | Audio shared by neighbouring chunks, under half a chunk. Default `4`. |

## Tests

Chunk planning and stitching are pure Python and need no model or numpy:

```sh
python3 -m unittest discover -s asr/worker -p 'test_*.py'
```

`test_worker_model.py` runs the real model on the committed fixture, including
chunks of 6-10 s at six start offsets, and skips itself without a model. Run it
where the worker runs, with the image built as in [Docker](#docker):

```sh
docker build -t subvision-asr-worker asr/worker
docker run --rm --entrypoint python -w /repo/asr/worker \
  -v "$PWD":/repo:ro -v "$PWD/models/parakeet":/models:ro \
  -e PARAKEET_MODEL_DIR=/models subvision-asr-worker \
  -m unittest discover -p 'test_*.py'
```
