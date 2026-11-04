# ASR worker: Parakeet TDT v3

This is a tiny HTTP worker for the Parakeet-first spike. It accepts a
`POST /transcribe` request with a 16 kHz mono PCM WAV body and returns
Subvision-compatible `segments` JSON.

## Run locally

```sh
uv venv /tmp/opencode/asr-worker-venv --python 3.11
uv pip install --python /tmp/opencode/asr-worker-venv/bin/python \
  --index-url https://pypi.org/simple \
  -r tools/asr-worker/requirements.txt

/tmp/opencode/asr-worker-venv/bin/python tools/asr-worker/main.py \
  --model-dir /tmp/opencode/parakeet-tdt-0.6b-v3-int8 \
  --host 127.0.0.1 \
  --port 8081
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

The image expects the pinned Parakeet model files mounted at `/models`:

```sh
docker build -t subvision-asr-worker tools/asr-worker
docker run --rm -p 8081:8081 \
  -v /tmp/opencode/parakeet-tdt-0.6b-v3-int8:/models:ro \
  subvision-asr-worker \
  python main.py --model-dir /models --host 0.0.0.0 --port 8081
```

## Notes

- This worker is optional. Set `PARAKEET_URL=http://127.0.0.1:8081/transcribe` to use it; leave it unset to keep the whisper.cpp path.
- It does not translate.
- Model weights and virtualenvs are intentionally kept outside the repo.
