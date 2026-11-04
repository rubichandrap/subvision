FROM python:3.11-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

COPY requirements.txt ./
RUN pip install --no-cache-dir --index-url https://pypi.org/simple -r requirements.txt

RUN addgroup --system app && adduser --system --ingroup app app

COPY --chown=app:app main.py ./

USER app

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8081/healthz', timeout=3)"

CMD ["python", "main.py", "--host", "0.0.0.0", "--port", "8081"]
