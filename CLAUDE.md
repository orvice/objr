# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is objr

objr is a lightweight Go HTTP service for uploading images to S3-compatible storage (MinIO). It exposes a token-authenticated REST API that accepts image uploads and stores them in a MinIO bucket, returning a CDN URL.

## Build and Test

```bash
make build        # builds bin/objr (CGO_ENABLED=0)
make test         # runs go test -v ./...
go test -v -run TestFunctionName ./internal/...  # run a single test
```

## Architecture

The app uses the `butterfly.orx.me/core` framework, which wraps Gin and provides config loading, logging, and OpenTelemetry integration.

- **`cmd/objr/main.go`** — Entry point. Wires config, routes, and init functions into the core framework via `core.New()`.
- **`internal/conf/`** — Config struct populated by the framework from YAML. Key fields: `auth_token`, `cors_header`, and `s3` (endpoint, credentials, bucket, CDN base URL).
- **`internal/apis/`** — Gin HTTP handlers and middleware. Routes are grouped under `/v1` with token auth middleware (`Token` header). Currently has one endpoint: `POST /v1/image`.
- **`internal/object/`** — MinIO client wrapper. `Init()` creates the client; `Upload()` puts an object and returns a CDN URL constructed from `cdn_base_url + object key`.

## Key Details

- Authentication is via a `Token` HTTP header checked against `conf.Conf.AuthToken`.
- Uploaded files are saved to `/tmp/` temporarily, then streamed to MinIO, then deleted.
- Object names follow the pattern `images/{year}/{month}/{day}/{uuid}-{filename}`.
- Content type is auto-detected using `gabriel-vasile/mimetype`.
- Config is loaded via the butterfly core framework (YAML-based); see `conf.Config` for the schema.
