"""gRPC servicer implementation for LaneIntelService."""

from __future__ import annotations

import logging
from collections.abc import Callable, Iterator
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import grpc

from model import QueueDepthEstimator
from proto import lane_pb2, lane_pb2_grpc

logger = logging.getLogger(__name__)

FrameProvider = Callable[[str, str], bytes]

DEFAULT_SECONDS_PER_PERSON = 45
DEFAULT_STREAM_INTERVAL_SECONDS = 5.0


def _default_frame_provider(lane_id: str, store_id: str) -> bytes:  # pragma: no cover
    """Placeholder camera frame provider.

    In production this function is replaced by an integration that fetches a
    frame from the store/lane camera (e.g. RTSP snapshot or object store). The
    default implementation returns a blank JPEG so the service can start without
    a camera configured.
    """
    from io import BytesIO
    from PIL import Image

    del lane_id, store_id
    image = Image.new("RGB", (640, 480), color=(128, 128, 128))
    buffer = BytesIO()
    image.save(buffer, format="JPEG")
    return buffer.getvalue()


class LaneIntelServicer(lane_pb2_grpc.LaneIntelServiceServicer):
    """gRPC servicer that estimates checkout-lane queue depth from camera frames."""

    def __init__(
        self,
        estimator: QueueDepthEstimator,
        frame_provider: FrameProvider,
        seconds_per_person: int = DEFAULT_SECONDS_PER_PERSON,
        stream_interval_seconds: float = DEFAULT_STREAM_INTERVAL_SECONDS,
    ):
        self.estimator = estimator
        self.frame_provider = frame_provider
        self.seconds_per_person = seconds_per_person
        self.stream_interval_seconds = stream_interval_seconds

    def _estimate(self, lane_id: str, store_id: str) -> lane_pb2.QueueDepthResponse:
        """Fetch a frame and estimate the current queue depth."""
        frame_bytes = self.frame_provider(lane_id, store_id)
        result = self.estimator.estimate(frame_bytes)
        captured_at = datetime.now(timezone.utc).isoformat()
        return lane_pb2.QueueDepthResponse(
            lane_id=lane_id,
            store_id=store_id,
            queue_depth=result.queue_depth,
            estimated_wait_seconds=result.estimated_wait_seconds,
            captured_at=captured_at,
        )

    def GetQueueDepth(
        self,
        request: lane_pb2.QueueDepthRequest,
        context: grpc.ServicerContext,
    ) -> lane_pb2.QueueDepthResponse:
        try:
            return self._estimate(request.lane_id, request.store_id)
        except Exception as exc:  # pragma: no cover
            logger.exception("GetQueueDepth failed for lane %s store %s", request.lane_id, request.store_id)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            return lane_pb2.QueueDepthResponse()

    def StreamQueueDepth(
        self,
        request: lane_pb2.QueueDepthRequest,
        context: grpc.ServicerContext,
    ) -> Iterator[lane_pb2.QueueDepthResponse]:
        """Stream queue-depth estimates until the client disconnects."""
        try:
            while context.is_active():
                yield self._estimate(request.lane_id, request.store_id)
                # Simple cooperative back-off; real deployments may use a
                # configured frame sampling interval.
                from time import sleep

                sleep(self.stream_interval_seconds)
        except Exception as exc:  # pragma: no cover
            logger.exception("StreamQueueDepth failed for lane %s store %s", request.lane_id, request.store_id)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))


def create_servicer(
    model_path: str | Path,
    frame_provider: FrameProvider | None = None,
    seconds_per_person: int = DEFAULT_SECONDS_PER_PERSON,
) -> LaneIntelServicer:
    """Create a LaneIntelServicer with the given model and frame provider."""
    estimator = QueueDepthEstimator(
        model_path=model_path,
        seconds_per_person=seconds_per_person,
    )
    return LaneIntelServicer(
        estimator=estimator,
        frame_provider=frame_provider or _default_frame_provider,
        seconds_per_person=seconds_per_person,
    )
