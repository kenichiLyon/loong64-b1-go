# python-ai-gateway

Internal AI middleware for the teaching evaluation system.

This service is intentionally narrow:

- it does not own business state
- it does not publish grades
- it does not expose public user-facing APIs

It only serves internal AI-heavy capabilities for the Go application.
