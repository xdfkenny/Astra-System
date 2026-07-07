"""Entry point for the Astra ML Lane Intelligence service.

Exposes:
  - a gRPC LaneIntelService on ``GRPC_PORT`` (default 50051)
  - a FastAPI HTTP server with health/readiness/liveness on ``PORT`` (default 8080)
"""

from __future__ import annotations

import argparse
import logging
import os
import threading
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from pathlib import Path
from typing import TYPE_CHECKING

import grpc
import uvicorn
from fastapi import FastAPI

import health
from service import LaneIntelServicer, create_servicer

if TYPE_CHECKING:
    from model import QueueDepthEstimator

logger = logging.getLogger(__name__)

DEFAULT_MODEL_PATH = Path(os.environ.get("MODEL_PATH", "models/yolov8n.onnx"))
DEFAULT_HOST = os.environ.get("HOST", "0.0.0.0")
DEFAULT_PORT = int(os.environ.get("PORT", "8080"))
DEFAULT_GRPC_PORT = int(os.environ.get("GRPC_PORT", "50051"))
DEFAULT_GRPC_WORKERS = int(os.environ.get("GRPC_WORKERS", "10"))


class AppState:
    """Shared runtime state managed by the ASGI lifespan."""

    def __init__(self) -> None:
        self.estimator: QueueDepthEstimator | None = None
        self.model_path: Path = DEFAULT_MODEL_PATH
        self.grpc_server: grpc.Server | None = None


state = AppState()


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:
    """Load the ONNX model once when the application starts."""
    model_path = state.model_path

    if not model_path.exists():
        logger.warning(
            "Model %s not found; service is degraded. "
            "Run 'python -m models.download_model' to create a stub model.",
            model_path,
        )
    else:
        servicer = create_servicer(model_path=model_path)
        state.estimator = servicer.estimator
        logger.info("Loaded queue-depth estimator from %s", model_path)

    yield

    state.estimator = None


app = FastAPI(
    title="Astra ML Lane Intelligence",
    description="Local queue-depth estimation using YOLOv8n and ONNX Runtime.",
    version="0.1.0",
    lifespan=lifespan,
)
app.include_router(health.router)


def _start_grpc_server(
    model_path: Path,
    port: int,
    workers: int,
) -> grpc.Server:
    """Create and start the gRPC server in a background thread."""
    servicer = create_servicer(model_path=model_path)
    state.estimator = servicer.estimator

    server = grpc.server(
        thread_pool=threading.ThreadPoolExecutor(max_workers=workers),
    )
    from proto import lane_pb2_grpc

    lane_pb2_grpc.add_LaneIntelServiceServicer_to_server(servicer, server)
    bound_port = server.add_insecure_port(f"[::]:{port}")
    server.start()
    logger.info("gRPC server listening on port %d", bound_port)
    # Expose the bound port for tests and callers that pass port=0.
    server._astra_bound_port = bound_port  # type: ignore[attr-defined]
    return server


def _serve(
    model_path: Path,
    host: str,
    port: int,
    grpc_port: int,
    grpc_workers: int,
) -> None:
    """Start gRPC and HTTP servers and block until interrupted."""
    state.model_path = model_path
    state.grpc_server = _start_grpc_server(
        model_path=model_path,
        port=grpc_port,
        workers=grpc_workers,
    )

    uvicorn.run(
        "main:app",
        host=host,
        port=port,
        log_level="info",
        factory=False,
    )


def cli(argv: list[str] | None = None) -> int:
    """Command-line entry point."""
    parser = argparse.ArgumentParser(description="Astra ML Lane Intelligence server")
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--grpc-port", type=int, default=DEFAULT_GRPC_PORT)
    parser.add_argument("--grpc-workers", type=int, default=DEFAULT_GRPC_WORKERS)
    parser.add_argument("--model", type=Path, default=DEFAULT_MODEL_PATH)
    args = parser.parse_args(argv)

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )

    _serve(
        model_path=args.model,
        host=args.host,
        port=args.port,
        grpc_port=args.grpc_port,
        grpc_workers=args.grpc_workers,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(cli())
