# Help Desk Backend

Go REST API for help desk ticket management with JWT authentication and RBAC.

API version: **v1** (`/api/v1`) — текущий контракт буткемпа.

## Stack

- Go 1.25, chi, pgx, PostgreSQL, JWT, slog

## Quick start

```bash
make up-db    # поднять Postgres
make migrate  # миграции
make run      # API на :8080
```

`make up` поднимает только Postgres (сборка API-образа не нужна для локального демо и часто падает на Docker Hub).

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

## Tests

```bash
make test
make test-race
make test-integration   # requires running postgres + migrations
```

## Environment

| Variable | Default |
|---|---|
| HTTP_ADDR | :8080 |
| DATABASE_URL | postgres://helpdesk:helpdesk@localhost:5433/helpdesk?sslmode=disable |
| JWT_SECRET | dev-secret-change-me (только APP_ENV=dev) |
| SEED_DEMO | false (в compose/make run: true) |
| SLA_HOURS | 48 |
| APP_ENV | dev |
