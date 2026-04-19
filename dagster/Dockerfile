FROM python:3.11-slim

WORKDIR /opt/dagster/app

ENV PIP_DEFAULT_TIMEOUT=120 \
	PIP_DISABLE_PIP_VERSION_CHECK=1 \
	PIP_NO_CACHE_DIR=1

COPY requirements.txt .
RUN python -m pip install --upgrade pip setuptools wheel \
	&& pip install --retries 10 --timeout 120 --progress-bar off -r requirements.txt

COPY . .

ENV DAGSTER_HOME=/opt/dagster/dagster_home

RUN mkdir -p /opt/dagster/dagster_home

COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

CMD ["/usr/local/bin/docker-entrypoint.sh"]