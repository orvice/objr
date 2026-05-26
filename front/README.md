# objr Frontend

A small React + shadcn-style SPA for uploading images to objr.

## Features

- Upload images to `POST /v1/image`
- Sends the required `Token` request header
- Shows image preview and returned CDN URL
- Copy/open uploaded URL

## Development

```bash
cd front
npm install
npm run dev
```

Open the printed Vite URL in your browser.

## Configuration

The API base URL input defaults to `VITE_API_BASE_URL` when set, otherwise the current browser origin.

```bash
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

The backend endpoint requires:

- Header: `Token: <auth_token>`
- Form field: `image=@file`

## Build

```bash
npm run build
```

The static files will be emitted to `front/dist`.

## Docker

Build the standalone frontend image:

```bash
docker build -f front/Dockerfile -t objr-front:local front
```

The image serves the static Vite build with nginx on port 80 and includes `/healthz`.
