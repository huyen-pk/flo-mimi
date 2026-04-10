FROM python:3.11-slim

WORKDIR /app

ENV PIP_DEFAULT_TIMEOUT=120 \
	PIP_DISABLE_PIP_VERSION_CHECK=1 \
	PIP_NO_CACHE_DIR=1

COPY requirements.txt .
RUN python -m pip install --upgrade pip setuptools wheel \
	&& pip install --retries 10 --timeout 120 --progress-bar off -r requirements.txt

COPY app ./app

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8010"]