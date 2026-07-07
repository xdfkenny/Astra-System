"""FastAPI health/readiness/liveness endpoints for the lane intelligence service."""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, status
from pydantic import BaseModel, Field

router = APIRouter(tags=["health"])


class HealthResponse(BaseModel):
    """Service health status."""

    status: str = Field(description="Overall service status")
    model_loaded: bool = Field(description="Whether the ONNX model is loaded")
    model_path: str | None = Field(None, description="Path to the loaded model")


class ReadyResponse(BaseModel):
    """Readiness probe response."""

    ready: bool = Field(description="True when the service is ready to serve traffic")
    model_loaded: bool = Field(description="Whether the ONNX model is loaded")


class LiveResponse(BaseModel):
    """Liveness probe response."""

    alive: bool = Field(description="True when the service process is alive")


def _health_state() -> dict[str, Any]:
    """Return the current health state from the shared app state.

    This function is defined lazily to avoid importing the FastAPI app at
    module-load time, which keeps the router importable from tests.
    """
    from main import state

    return {
        "status": "ok" if state.estimator is not None else "degraded",
        "model_loaded": state.estimator is not None,
        "model_path": str(state.model_path) if state.model_path else None,
    }


@router.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    """Service health check."""
    state = _health_state()
    return HealthResponse(
        status=state["status"],
        model_loaded=state["model_loaded"],
        model_path=state["model_path"],
    )


@router.get("/ready", response_model=ReadyResponse)
async def ready() -> ReadyResponse:
    """Readiness probe. Returns 503 until the model is loaded."""
    state = _health_state()
    ready_flag = state["model_loaded"]
    response = ReadyResponse(
        ready=ready_flag,
        model_loaded=state["model_loaded"],
    )
    if not ready_flag:
        from fastapi import Response

        return Response(
            content=response.model_dump_json(),
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            media_type="application/json",
        )
    return response


@router.get("/live", response_model=LiveResponse)
async def live() -> LiveResponse:
    """Liveness probe. Always returns 200 while the process is running."""
    return LiveResponse(alive=True)
