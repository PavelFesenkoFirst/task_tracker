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

## Available endpoints

- `GET /health`
- `GET /ping`
- `POST /tasks`
- `GET /tasks`
- `GET /tasks/{id}`
- `PATCH /tasks/{id}`
- `DELETE /tasks/{id}`

## Task API examples

Create task:

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Write integration tests",
    "description": "Cover mysql repository and HTTP handlers",
    "status": "new",
    "priority": 4,
    "due_at": "2026-03-10T12:00:00Z"
  }'
```

List tasks:

```bash
curl "http://localhost:8080/tasks?status=new&q=tests&limit=20&offset=0"
```

Get task by id:

```bash
curl http://localhost:8080/tasks/1
```

Update task:

```bash
curl -X PATCH http://localhost:8080/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Write CRUD integration tests",
    "status": "in_progress",
    "priority": 5
  }'
```

Clear due date:

```bash
curl -X PATCH http://localhost:8080/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{
    "clear_due_at": true
  }'
```

Delete task:

```bash
curl -X DELETE http://localhost:8080/tasks/1
```
