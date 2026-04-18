# objr API Documentation

This document describes the HTTP endpoints currently provided by objr.

## Basics

- Default response format: JSON
- Authentication: all `/v1/*` endpoints and `/mcp` require the `Token` request header
- Upload content type: `multipart/form-data`
- Download URL: generated from `s3.cdn_base_url` + object key after a successful upload

### Authentication Header

```http
Token: <auth_token>
```

The `Token` value must match `auth_token` in the service configuration. If authentication fails, the response is:

```http
401 Unauthorized
```

## GET /ping

Health check endpoint.

### Authentication

Not required.

### Request Example

```bash
curl "$BASE_URL/ping"
```

### Success Response

```http
200 OK
```

```json
{
  "message": "pong"
}
```

## /mcp

Streamable HTTP MCP endpoint for MCP-capable clients. The endpoint is served on the same HTTP server as the REST API and uses `github.com/modelcontextprotocol/go-sdk/mcp`.

### Authentication

Requires the `Token` request header on every MCP HTTP request.

### Supported Methods

The streamable HTTP handler is mounted for:

- `GET /mcp`
- `POST /mcp`
- `DELETE /mcp`

MCP clients should send normal streamable HTTP JSON-RPC traffic to `/mcp`. Upload tools accept file content as structured JSON arguments, not `multipart/form-data`.

### MCP Client Configuration

Configure the MCP client with endpoint:

```text
{BASE_URL}/mcp
```

and request header:

```http
Token: <auth_token>
```

### Tool: `ping`

Reports service liveness.

Input:

```json
{}
```

Output:

```json
{
  "message": "pong"
}
```

### Tool: `upload_image`

Uploads an image to object storage and returns the generated CDN URL.

Input fields:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `file_path` | string | Conditional | Server-readable local image path |
| `content_base64` | string | Conditional | Base64-encoded image content |
| `filename` | string | Conditional | Required with `content_base64`; optional override for `file_path` basename |

Exactly one of `file_path` or `content_base64` must be provided.

Input example with base64 content:

```json
{
  "content_base64": "<base64 image bytes>",
  "filename": "demo.png"
}
```

Output:

```json
{
  "message": "success",
  "url": "https://cdn.example.com/images/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.png",
  "object_key": "images/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.png",
  "content_type": "image/png"
}
```

### Tool: `upload_app_package`

Uploads an Android APK or AAB and returns historical plus stable channel download metadata.

Input fields:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `app_name` | string | Yes | Application name, sanitized for object key paths |
| `version` | string | No | App version. Defaults to `nightly` when missing or blank |
| `file_path` | string | Conditional | Server-readable local APK or AAB path |
| `content_base64` | string | Conditional | Base64-encoded APK or AAB content |
| `filename` | string | Conditional | Required with `content_base64`; optional override for `file_path` basename |

Exactly one of `file_path` or `content_base64` must be provided. Filename must end with `.apk` or `.aab` case-insensitively.

Input example with base64 content:

```json
{
  "app_name": "demo-app",
  "version": "1.2.3",
  "content_base64": "<base64 package bytes>",
  "filename": "demo.aab"
}
```

Output for a nightly upload includes the fixed nightly URL:

```json
{
  "message": "success",
  "download_url": "https://cdn.example.com/apps/demo-app/nightly/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.apk",
  "app_name": "demo-app",
  "version": "nightly",
  "object_key": "apps/demo-app/nightly/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.apk",
  "nightly_download_url": "https://cdn.example.com/apps/demo-app/nightly/app.apk",
  "nightly_object_key": "apps/demo-app/nightly/app.apk"
}
```

Output for a non-nightly upload includes the fixed latest URL:

```json
{
  "message": "success",
  "download_url": "https://cdn.example.com/apps/demo-app/1.2.3/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.aab",
  "app_name": "demo-app",
  "version": "1.2.3",
  "object_key": "apps/demo-app/1.2.3/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.aab",
  "latest_download_url": "https://cdn.example.com/apps/demo-app/latest/app.aab",
  "latest_object_key": "apps/demo-app/latest/app.aab"
}
```

### MCP Tool Errors

Tool calls return MCP tool errors for invalid inputs, including:

- Missing upload source
- Both `file_path` and `content_base64` provided
- Unreadable `file_path`
- Invalid base64 content
- Missing `filename` when using `content_base64`
- Blank `app_name`
- Unsupported app package filename extension

## POST /v1/image

Upload an image to object storage and return a CDN URL.

### Authentication

Requires the `Token` request header.

### Request Parameters

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `image` | file | Yes | Image file to upload |

### Request Example

```bash
curl -X POST "$BASE_URL/v1/image" \
  -H "Token: $TOKEN" \
  -F "image=@./demo.png"
```

### Storage Path

Image object key format:

```text
images/{year}/{month}/{day}/{uuid}-{filename}
```

Example:

```text
images/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.png
```

### Success Response

```http
200 OK
```

```json
{
  "message": "success",
  "url": "https://cdn.example.com/images/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.png",
  "file_mine": {
    "Content-Disposition": [
      "form-data; name=\"image\"; filename=\"demo.png\""
    ],
    "Content-Type": [
      "image/png"
    ]
  },
  "content_type": "image/png"
}
```

### Error Response

Returned when `image` is missing, temporary file save fails, object storage upload fails, or similar errors:

```http
500 Internal Server Error
```

```json
{
  "message": "<error message>"
}
```

## POST /v1/app-package

Upload an Android app package to object storage and return historical download URLs. Supports both `.apk` and `.aab` files.

When the effective version is `nightly`, the endpoint also overwrites a fixed nightly download object. When the effective version is not `nightly`, it also overwrites a fixed latest download object.

### Authentication

Requires the `Token` request header.

### Request Parameters

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `file` | file | Yes | Android app package to upload. Filename must end with `.apk` or `.aab` (case-insensitive) |
| `app_name` | string | Yes | Application name, used as part of the object key path |
| `version` | string | No | App version. Defaults to `nightly` if missing or blank |

### Field Normalization

- `version` is normalized to `nightly` when missing or containing only whitespace.
- `app_name`, `version`, and filename are sanitized for path safety:
  - Keep letters, numbers, `.`, `_`, `-`
  - Replace all other characters with `-`
  - Trim leading and trailing `.` and `-`

### Request Example: Upload Nightly APK

```bash
curl -X POST "$BASE_URL/v1/app-package" \
  -H "Token: $TOKEN" \
  -F "app_name=demo-app" \
  -F "file=@./demo.apk"
```

### Request Example: Upload Release AAB

```bash
curl -X POST "$BASE_URL/v1/app-package" \
  -H "Token: $TOKEN" \
  -F "app_name=demo-app" \
  -F "version=1.2.3" \
  -F "file=@./demo.aab"
```

### Historical Object Path

Every successful upload stores one historical object with date and UUID:

```text
apps/{app_name}/{version}/{year}/{month}/{day}/{uuid}-{filename}
```

Example:

```text
apps/demo-app/1.2.3/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.apk
```

### Fixed Nightly Download Path

When the effective version is `nightly`, the endpoint additionally overwrites fixed objects:

```text
apps/{app_name}/nightly/app.apk
apps/{app_name}/nightly/app.aab
```

Response includes:

- `nightly_download_url`
- `nightly_object_key`

Nightly uploads do not update fixed latest objects.

### Fixed Latest Download Path

When the effective version is not `nightly`, the endpoint additionally overwrites fixed objects:

```text
apps/{app_name}/latest/app.apk
apps/{app_name}/latest/app.aab
```

Response includes:

- `latest_download_url`
- `latest_object_key`

Release uploads do not update fixed nightly objects.

### Success Response: Nightly Upload

```http
200 OK
```

```json
{
  "message": "success",
  "download_url": "https://cdn.example.com/apps/demo-app/nightly/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.apk",
  "app_name": "demo-app",
  "version": "nightly",
  "object_key": "apps/demo-app/nightly/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.apk",
  "nightly_download_url": "https://cdn.example.com/apps/demo-app/nightly/app.apk",
  "nightly_object_key": "apps/demo-app/nightly/app.apk"
}
```

### Success Response: Release Upload

```http
200 OK
```

```json
{
  "message": "success",
  "download_url": "https://cdn.example.com/apps/demo-app/1.2.3/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.aab",
  "app_name": "demo-app",
  "version": "1.2.3",
  "object_key": "apps/demo-app/1.2.3/2026/4/18/018f89e0-1234-5678-90ab-abcdefabcdef-demo.aab",
  "latest_download_url": "https://cdn.example.com/apps/demo-app/latest/app.aab",
  "latest_object_key": "apps/demo-app/latest/app.aab"
}
```

### Error Responses

Missing file:

```http
400 Bad Request
```

```json
{
  "message": "file is required"
}
```

Missing or empty app name:

```http
400 Bad Request
```

```json
{
  "message": "app_name is required"
}
```

File extension is not `.apk` or `.aab`:

```http
400 Bad Request
```

```json
{
  "message": "file must be .apk or .aab"
}
```

Returned when temporary file upload, UUID generation, object storage upload, or similar operations fail:

```http
500 Internal Server Error
```

```json
{
  "message": "<error message>"
}
```
