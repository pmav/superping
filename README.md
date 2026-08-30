# ⚡ Superping

Superping is a self-hosted service to monitor network latency using simple pings. It stores your targets in a local SQLite database, checks them on a schedule, and exposes both a lightweight HTTP API and a browser UI for quick visibility into latency and uptime.

## 🌐 What it does

- Monitors network endpoints with the system `ping` command
- Tracks latency in milliseconds
- Stores results locally in SQLite
- Gives you a simple web dashboard for your targets
- Runs as a lightweight self-hosted service on your own machine or server

## 🚀 Run locally

Use the project Make targets to build and run the app locally:

```sh
make build
make run
```

This builds the Docker image and starts the service on `http://localhost:8080`.

You can override the image name or port if needed:

```sh
make build IMAGE_NAME=superping TAG=latest
make run PORT=8080 IMAGE_NAME=superping TAG=latest
```

The container mounts the local `data/` directory to `/data`, so the database persists between runs.

Then open `http://localhost:8080` in your browser to use the built-in web app.

## 🧭 How it works

- Each ping target is stored as a record in the local database
- The background monitor checks each active ping once per second
- Results are written to the `ping_result` table
- Failed checks include the error message in `message`
- Latency is measured in milliseconds
- The web app shows recent checks and target status

Each API request is logged with its method, path, response status, and duration.

## 📡 API

Create a ping:

```sh
curl -X POST http://localhost:8080/pings \
  -H 'Content-Type: application/json' \
  -d '{"name":"router","ip":"192.168.1.1","status":"on"}'
```

Notes:

- `status` accepts `on` or `off`
- it defaults to `off` when omitted
- only pings with status `on` are monitored
- responses include UTC `created_at` and `updated_at` timestamps

Available endpoints:

- `GET /pings` — list all ping targets
- `POST /pings` — create a ping target
- `GET /pings/{uuid}/results` — read recent results for a ping
- `PUT /pings/{uuid}` — update one or more ping fields
- `DELETE /pings/{uuid}` — remove a ping target

The web UI displays the most recent results for each ping, up to the latest 1000 checks.

## 🧪 Tests

Run the Go test suite with:

```sh
go test ./...
```

## 🐳 Docker

Build and run the service with a persistent local database volume:

```sh
docker build -t superping .
docker run --rm -p 8080:8080 -v superping-data:/data superping
```

This keeps the database data outside the container so your checks remain available between restarts.

## 🛠️ Why Superping?

Because sometimes the simplest monitoring is the most reliable: a tiny self-hosted service that checks the basics that matter most — whether your network endpoints respond, and how fast they do it. ✨