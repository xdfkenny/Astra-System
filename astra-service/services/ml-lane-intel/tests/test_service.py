"""Unit tests for the LaneIntelService gRPC servicer."""

from __future__ import annotations

from pathlib import Path

import grpc
import pytest
from concurrent import futures

from model import QueueDepthEstimator
from proto import lane_pb2, lane_pb2_grpc
from service import LaneIntelServicer


@pytest.fixture
def grpc_server(model_path: Path, fake_image_bytes: bytes):
    """Start the gRPC server on a free port and yield a channel to it."""
    estimator = QueueDepthEstimator(model_path=model_path)

    def frame_provider(lane_id: str, store_id: str) -> bytes:
        del lane_id, store_id
        return fake_image_bytes

    servicer = LaneIntelServicer(
        estimator=estimator,
        frame_provider=frame_provider,
        stream_interval_seconds=0.1,
    )
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    lane_pb2_grpc.add_LaneIntelServiceServicer_to_server(servicer, server)
    bound_port = server.add_insecure_port("localhost:0")
    server.start()

    channel = grpc.insecure_channel(f"localhost:{bound_port}")
    yield channel

    channel.close()
    server.stop(grace=None)


class TestGetQueueDepth:
    def test_returns_queue_depth(self, grpc_server: grpc.Channel) -> None:
        stub = lane_pb2_grpc.LaneIntelServiceStub(grpc_server)
        request = lane_pb2.QueueDepthRequest(lane_id="lane-1", store_id="store-42")
        response = stub.GetQueueDepth(request)
        assert response.lane_id == "lane-1"
        assert response.store_id == "store-42"
        assert response.queue_depth == 3
        assert response.estimated_wait_seconds == 135
        assert response.captured_at


class TestStreamQueueDepth:
    def test_streams_multiple_responses(self, grpc_server: grpc.Channel) -> None:
        stub = lane_pb2_grpc.LaneIntelServiceStub(grpc_server)
        request = lane_pb2.QueueDepthRequest(lane_id="lane-2", store_id="store-42")
        responses = []
        for response in stub.StreamQueueDepth(request):
            responses.append(response)
            if len(responses) >= 2:
                break
        assert len(responses) == 2
        assert all(r.lane_id == "lane-2" for r in responses)
        assert all(r.store_id == "store-42" for r in responses)
        assert all(r.queue_depth == 3 for r in responses)


class TestLaneIntelServicerDirectly:
    def test_estimate_populates_all_fields(self, model_path: Path, fake_image_bytes: bytes) -> None:
        estimator = QueueDepthEstimator(model_path=model_path)
        servicer = LaneIntelServicer(
            estimator=estimator,
            frame_provider=lambda _l, _s: fake_image_bytes,
        )
        response = servicer._estimate("lane-a", "store-b")
        assert response.lane_id == "lane-a"
        assert response.store_id == "store-b"
        assert response.queue_depth == 3
        assert response.estimated_wait_seconds == 135
        assert response.captured_at
