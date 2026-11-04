FROM python:3.11-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

COPY requirements.txt ./
RUN pip install --no-cache-dir --index-url https://pypi.org/simple -r requirements.txt

COPY main.py ./

EXPOSE 8081

CMD ["python", "main.py", "--host", "0.0.0.0", "--port", "8081"]
