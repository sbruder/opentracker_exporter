FROM python:3-alpine

WORKDIR /usr/src/opentracker_exporter

COPY requirements.txt .

RUN apk add --no-cache --virtual .build-deps \
        build-base \
        libxml2-dev \
        libxslt-dev \
    && pip3 install --no-cache-dir -r requirements.txt \
    && apk del .build-deps \
    && apk add --no-cache \
        libxml2 \
        libxslt

COPY opentracker_exporter.py .

ENTRYPOINT ["/usr/src/opentracker_exporter/opentracker_exporter.py"]

EXPOSE 9574
