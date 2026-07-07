"""Tests for FastAPI health/readiness/liveness endpoints."""

from __future__ import annotations

from pathlib import Path

import pytest
from fastapi.testclient import TestClient

import main


@pytest.fixture
def client(model_path: Path) -> TestClient:
    """Create a TestClient with the stub model loaded at startup."""
    original_model = main.state.model_path
    main.state.model_path = model_path
    main.state.estimator = None

    with TestClient(main.app) as test_client:
        yield test_client

    main.state.model_path = original_model
    main.state.estimator = None


class TestHealth:
    def test_health_ok(self, client: TestClient) -> None:
        response = client.get("/health")
        assert response.status_code == 200
        body = response.json()
        assert body["status"] == "ok"
        assert body["model_loaded"] is True
        assert body["model_path"] is not None


class TestReady:
    def test_ready_when_model_loaded(self, client: TestClient) -> None:
        response = client.get("/ready")
        assert response.status_code == 200
        body = response.json()
        assert body["ready"] is True
        assert body["model_loaded"] is True

    def test_ready_when_model_missing(self, client: TestClient) -> None:
        main.state.estimator = None
        response = client.get("/ready")
        assert response.status_code == 503
        body = response.json()
        assert body["ready"] is False
        assert body["model_loaded"] is False


class TestLive:
    def test_live(self, client: TestClient) -> None:
        response = client.get("/live")
        assert response.status_code == 200
        body = response.json()
        assert body["alive"] is True
