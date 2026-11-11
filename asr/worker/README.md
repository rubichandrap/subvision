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

## Environment

| Variable | Description |
| --- | --- |
| `PARAKEET_MODEL_DIR` | Directory containing the four model files. Set to `/models` in the container. |
| `PARAKEET_TIMEOUT_SECONDS` | Server-side request ceiling; set on the server, not the worker. |
| `ASR_MAX_BODY_BYTES` | Request body cap; default 256 MiB. |
| `ASR_HOST` / `ASR_PORT` | Bind address and port; defaults `0.0.0.0:8081`. |
| `ASR_THREADS` | ONNX runtime thread count. |
