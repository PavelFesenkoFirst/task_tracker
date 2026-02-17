# task_tracker

Task Tracker API with background worker processes.

Current architecture:

```text
cmd/
  api/main.go
  worker/main.go

internal/
  config/
  platform/
    logger/
    mysql/
  task/
  job/
```

## Run (dev)

```bash
docker compose -f docker-compose.dev.yml up --build
```

## Run (prod-like)

```bash
docker compose up --build
```

## Available endpoints (current bootstrap)

- `GET /health`
- `GET /ping`
