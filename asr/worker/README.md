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
179e50c43d1a9de79c8a24149a2f9bac6eb5981823f2a2ed88d655b24248db4e  decoder.int8.onnx
acfc2b4456377e15d04f0243af540b7fe7c992f8d898d751cf134c3a55fd2247  encoder.int8.onnx
3164c13fc2821009440d20fcb5fdc78bff28b4db2f8d0f0b329101719c0948b3  joiner.int8.onnx
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
