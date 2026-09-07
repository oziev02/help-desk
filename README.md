# Help Desk Backend

[![CI](https://github.com/oziev02/help-desk/actions/workflows/ci.yml/badge.svg)](https://github.com/oziev02/help-desk/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/oziev02/help-desk)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/oziev02/help-desk)](https://goreportcard.com/report/github.com/oziev02/help-desk)
[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/oziev02/help-desk/pulls)

Go REST API для help desk: JWT, RBAC, PostgreSQL, OpenAPI.

API version: **v1** (`/api/v1`) - текущий контракт буткемпа.

## About

Этот репозиторий появился в контексте буткемпа, где я участвовал как ментор. Проект реализован как учебный ориентир: участники могут разобрать рабочий backend end-to-end и увидеть практичные Go-подходы - слоистую архитектуру, доменные правила и RBAC, миграции, Docker, тесты с race detection, линтеры и CI.

Задача не в «идеальном продакшене», а в понятном и воспроизводимом примере, от которого удобно отталкиваться.

## Stack

- Go 1.25, chi, pgx, PostgreSQL, JWT, slog

## Architecture

Слоистая структура (`cmd` + `internal`), зависимости направлены внутрь:

```
cmd/server       → точка входа
internal/app     → wiring, router, lifecycle
internal/handler → HTTP + DTO
internal/service → use-cases
internal/domain  → сущности, RBAC, переходы статусов, sentinel errors
internal/repository → Postgres (интерфейсы Store + pgx)
```

DI вручную в `app.New`. Контракт API: [api/openapi.yaml](api/openapi.yaml).

## Quick start

Одной командой (нужен только Docker) - Postgres, миграции и API:

```bash
docker compose up --build
```

Эквивалент: `make up` (в фоне: `docker compose up --build -d`).

При старте API-контейнер сам применяет миграции, затем поднимает сервер. Локальный Go и `migrate` на хосте не нужны.

API: http://localhost:8080 · Postgres на хосте: `localhost:5433`.

Остановка: `docker compose down` или `make down`.

### Dev-режим (опционально)

Postgres из compose, API через `go run` в golang-контейнере (без сборки API-образа):

```bash
make up-db
make migrate
make run
```

Server listens on `:8080`.

## Demo users

| Email | Password | Role |
|---|---|---|
| admin@helpdesk.local | admin123 | admin |
| user@helpdesk.local | user123 | user (заявитель) |
| dispatcher@helpdesk.local | dispatcher123 | dispatcher |
| executor@helpdesk.local | executor123 | executor |
| manager@helpdesk.local | manager123 | manager (руководитель) |

## Example flow

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@helpdesk.local","password":"user123"}' | jq -r .access_token)

# Create ticket (due_at ставится автоматически по SLA_HOURS)
curl -s -X POST http://localhost:8080/api/v1/tickets \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Broken lamp","description":"Room 5"}'

# Dispatcher assigns → executor works → applicant completes with rating
# POST /tickets/{id}/complete {"rating":5,"comment":"ok"}
# Executor may refuse: POST /tickets/{id}/refuse {"reason":"..."}
```

## Extra features (допы ТЗ)

- **SLA**: `SLA_HOURS` (default 48), авто-`due_at`, флаг `overdue`, фильтр `?overdue=true`
- **Уведомления**: `LogNotifier` пишет события assign/status/refuse/complete в slog
- **Помещения**: `GET /rooms`, admin CRUD `/admin/rooms`
- **Связывание заявок**: link/unlink только dispatcher/admin
- **Отчёты**: `GET /reports/tickets?format=csv` (manager/dispatcher/admin)
- **Справочники**: categories + deactivate `PATCH /admin/categories/{id}`

## API

OpenAPI spec: [api/openapi.yaml](api/openapi.yaml)

Main endpoints:

- `POST /api/v1/auth/login`, `POST /api/v1/auth/register`
- `GET/POST /api/v1/tickets`, actions: assign, transitions, complete, refuse, reopen, cancel, comments, links
- `GET /api/v1/rooms`, `GET /api/v1/categories`
- `GET /api/v1/reports/tickets?format=csv`
- Admin: roles, categories, rooms under `/api/v1/admin/...`

## Tests and quality

```bash
make test
make test-race
make test-integration   # requires running postgres + migrations
make lint
make lint-fix            # автофикс, где возможно
make fmt                # gofumpt / goimports через golangci-lint
make vet
make vuln
```

CI (GitHub Actions): unit tests с `-race`, integration tests против Postgres, `go vet`, `golangci-lint` (включая gosec), `govulncheck`, CodeQL. Dependabot еженедельно предлагает обновления Go-модулей, Actions и Docker-образов (см. `.github/dependabot.yml`).

## Environment

| Variable | Default |
|---|---|
| HTTP_ADDR | :8080 |
| DATABASE_URL | postgres://helpdesk:helpdesk@localhost:5433/helpdesk?sslmode=disable |
| JWT_SECRET | dev-secret-change-me (только APP_ENV=dev) |
| SEED_DEMO | false (в compose/make run: true) |
| SLA_HOURS | 48 |
| APP_ENV | dev |
