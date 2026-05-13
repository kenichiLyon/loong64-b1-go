# python-ai-gateway

Internal AI middleware for the teaching evaluation system.

This service is intentionally narrow:

- it does not own business state
- it does not publish grades
- it does not expose public user-facing APIs

It only serves internal AI-heavy capabilities for the Go application.

Current real capabilities:

- artifact parsing for text, docx, pdf, images, and zip archives
- evaluation through an OpenAI-compatible model endpoint when `AI_GATEWAY_LLM_*` is configured
- in-memory retrieval index building and query endpoints for artifact evidence
- retrieval-augmented evaluation prompts built from artifact excerpts

Current stub capabilities:

- none for the current middleware endpoints

Evaluation environment variables:

- `AI_GATEWAY_LLM_BASE_URL`
- `AI_GATEWAY_LLM_API_KEY`
- `AI_GATEWAY_LLM_MODEL`
- `AI_GATEWAY_LLM_TIMEOUT` (seconds, optional, default `30`)

If the evaluator env vars are missing, `/internal/evaluate-submission` returns an internal error and the Go service can fall back to its native evaluator path.
