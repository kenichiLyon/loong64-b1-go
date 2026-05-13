# ADR 0001: Python AI Middleware Without LangChain

## Status

Accepted

## Context

The project is a Go-first teaching evaluation system targeting LoongArch and Kylin Linux.
The business core is stable in Go:

- HTTP API and middleware
- database and storage
- audit and report workflows
- user, teacher, and student interaction flows

However, several AI-heavy capabilities are expected to evolve quickly and have better ecosystem support in Python:

- document parsing
- OCR
- embeddings and retrieval
- local model integration
- structured AI evaluation pipelines

The current concern is not whether AI should exist, but where those AI-heavy capabilities should live.

## Decision

We introduce a separate Python HTTP service as an internal AI middleware.

Go remains the primary application service and source of business truth.
Python handles AI-heavy capabilities only.

We do not introduce LangChain in the first phase.

Instead, the Python middleware uses:

- FastAPI for HTTP serving
- Pydantic for schemas
- direct model / parser integrations behind plain service functions

## Scope Split

### Go keeps

- external REST API
- auth, authorization, sessions, CSRF
- business workflows
- persistence and audit logs
- report generation and delivery
- orchestration of AI tasks

### Python handles

- artifact parsing
- OCR
- retrieval index building
- retrieval query execution
- local / remote model invocation
- structured extraction and advisory evaluation payload generation

## Why Not LangChain Now

- The current need is capability isolation, not complex agent orchestration.
- Adding LangChain would add an extra abstraction layer before the actual parsing and retrieval strategy is even stable.
- The Go service needs a simple internal HTTP contract, not framework-specific chain semantics.
- LoongArch delivery and deployment complexity should stay as low as possible.

## Consequences

### Positive

- AI-heavy code can evolve independently of the Go business core.
- Python ecosystem tools can be adopted where they have clear advantages.
- Go stays small, auditable, and deployment-oriented.
- Future migration to LangChain or LangGraph remains possible inside the Python middleware without changing the Go contract.

### Negative

- The system becomes multi-service.
- Internal API contracts must be versioned and tested.
- Operational monitoring and retries become more important.

## Next Steps

1. Add Python AI middleware skeleton with stable internal endpoints.
2. Add Go-side internal client and readiness integration.
3. Route future parsing / retrieval / local-model work through the Python middleware.
4. Re-evaluate LangChain only after real orchestration complexity appears.
