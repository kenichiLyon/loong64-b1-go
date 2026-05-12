from __future__ import annotations

from fastapi import FastAPI

from .models import (
    BuildRetrievalIndexRequest,
    BuildRetrievalIndexResponse,
    EvaluateSubmissionRequest,
    EvaluateSubmissionResponse,
    HealthResponse,
    ParseArtifactRequest,
    ParseArtifactResponse,
    QueryRetrievalRequest,
    QueryRetrievalResponse,
)


app = FastAPI(title="python-ai-gateway", version="0.1.0")


@app.get("/health/live", response_model=HealthResponse)
def live() -> HealthResponse:
    return HealthResponse()


@app.get("/health/ready", response_model=HealthResponse)
def ready() -> HealthResponse:
    return HealthResponse()


@app.post("/internal/parse-artifact", response_model=ParseArtifactResponse)
def parse_artifact(request: ParseArtifactRequest) -> ParseArtifactResponse:
    return ParseArtifactResponse(
        text_excerpt="stub parse result",
        metadata={
            "artifact_id": request.artifact_id,
            "artifact_kind": request.artifact_kind,
            "mode": "stub",
        },
        sections=[],
        evidence=[],
    )


@app.post("/internal/evaluate-submission", response_model=EvaluateSubmissionResponse)
def evaluate_submission(request: EvaluateSubmissionRequest) -> EvaluateSubmissionResponse:
    return EvaluateSubmissionResponse(
        summary="stub evaluation result",
        findings=[],
        metric_scores=[],
        confidence=0.0,
        raw_model_meta={"mode": request.mode, "engine": "stub"},
    )


@app.post("/internal/build-retrieval-index", response_model=BuildRetrievalIndexResponse)
def build_retrieval_index(request: BuildRetrievalIndexRequest) -> BuildRetrievalIndexResponse:
    return BuildRetrievalIndexResponse(
        index_ref=f"stub-index:{request.submission_id}",
        chunk_count=len(request.chunks),
    )


@app.post("/internal/query-retrieval", response_model=QueryRetrievalResponse)
def query_retrieval(request: QueryRetrievalRequest) -> QueryRetrievalResponse:
    return QueryRetrievalResponse(
        matches=[],
        citations=[],
    )
