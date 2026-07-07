# syntax=docker/dockerfile:1
FROM python:3.13-slim AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    cmake \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml README.md ./
COPY src ./src
COPY proto ./proto
COPY models ./models

RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install --upgrade pip setuptools wheel \
    && pip install ".[dev]" \
    && python -m models.download_model --output models/yolov8n.onnx --force

# -----------------------------------------------------------------------------
FROM python:3.13-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    MODEL_PATH=/app/models/yolov8n.onnx \
    HOST=0.0.0.0 \
    PORT=8080 \
    GRPC_PORT=50051

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid 1000 astra && \
    useradd --uid 1000 --gid astra --shell /bin/false --create-home astra

WORKDIR /app

COPY --from=builder /opt/venv /opt/venv
COPY --from=builder /build/models /app/models
COPY --from=builder /build/src /app/src
COPY --from=builder /build/proto /app/proto
COPY --from=builder /build/pyproject.toml /app/pyproject.toml
COPY --from=builder /build/README.md /app/README.md

ENV PATH="/opt/venv/bin:$PATH" \
    PYTHONPATH="/app/src:/app/proto"

USER astra

EXPOSE 8080 50051

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/health')" \
    || exit 1

CMD ["python", "-m", "src.main"]
