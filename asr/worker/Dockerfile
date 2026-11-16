FROM python:3.11-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

COPY requirements.txt ./
RUN pip install --no-cache-dir --index-url https://pypi.org/simple -r requirements.txt

# Optional vocal isolation (ADR-0013, experiment): build with
# --build-arg VOCAL_ISOLATION=1 to add Demucs (about 2 GB more) and bake its
# weights in; run with ASR_VOCAL_ISOLATION=1 to use it. Off by default.
ARG VOCAL_ISOLATION=0
ENV TORCH_HOME=/weights
RUN if [ "$VOCAL_ISOLATION" = "1" ]; then \
      apt-get update && apt-get install -y --no-install-recommends ffmpeg && rm -rf /var/lib/apt/lists/* \
      && pip install --no-cache-dir --index-url https://download.pytorch.org/whl/cpu torch==2.0.1 torchaudio==2.0.2 \
      && pip install --no-cache-dir --index-url https://pypi.org/simple demucs==4.0.1 numpy==1.26.4 \
      && python -c "from demucs.pretrained import get_model; get_model('htdemucs')" \
      && chmod -R a+rX /weights; \
    fi

RUN addgroup --system app && adduser --system --ingroup app app

COPY --chown=app:app main.py chunking.py isolation.py ./

USER app

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8081/healthz', timeout=3)"

CMD ["python", "main.py", "--host", "0.0.0.0", "--port", "8081"]
