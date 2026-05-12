from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field


class HealthResponse(BaseModel):
    status: Literal["ok", "fail"] = "ok"
    service: str = "python-ai-gateway"


class ParseArtifactRequest(BaseModel):
    artifact_id: str
    artifact_kind: str
    storage_path_or_url: str
    content_type: str = ""
    parse_options: dict[str, Any] = Field(default_factory=dict)


class ParseArtifactResponse(BaseModel):
    status: Literal["succeeded", "failed"] = "succeeded"
    text_excerpt: str = ""
    metadata: dict[str, Any] = Field(default_factory=dict)
    sections: list[dict[str, Any]] = Field(default_factory=list)
    evidence: list[dict[str, Any]] = Field(default_factory=list)
    error: str = ""


class EvaluateSubmissionRequest(BaseModel):
    submission_id: str
    rubric: dict[str, Any] = Field(default_factory=dict)
    submission_spec: dict[str, Any] = Field(default_factory=dict)
    evidence_bundle: dict[str, Any] = Field(default_factory=dict)
    mode: str = "rule_and_llm"


class EvaluateSubmissionResponse(BaseModel):
    summary: str = ""
    findings: list[dict[str, Any]] = Field(default_factory=list)
    metric_scores: list[dict[str, Any]] = Field(default_factory=list)
    confidence: float = 0.0
    raw_model_meta: dict[str, Any] = Field(default_factory=dict)
    error: str = ""


class BuildRetrievalIndexRequest(BaseModel):
    submission_id: str
    artifact_ids: list[str] = Field(default_factory=list)
    chunks: list[dict[str, Any]] = Field(default_factory=list)


class BuildRetrievalIndexResponse(BaseModel):
    index_ref: str
    chunk_count: int
    error: str = ""


class QueryRetrievalRequest(BaseModel):
    index_ref: str
    query: str
    top_k: int = 5


class QueryRetrievalResponse(BaseModel):
    matches: list[dict[str, Any]] = Field(default_factory=list)
    citations: list[dict[str, Any]] = Field(default_factory=list)
    error: str = ""
