.PHONY: up down up-db migrate test test-race test-integration build run lint vuln

DB_URL = postgres://helpdesk:helpdesk@127.0.0.1:5433/helpdesk?sslmode=disable

DOCKER_GO = docker run --rm --network=host \
	-v "$(CURDIR):/src" \
	-v go-mod-cache:/go/pkg/mod \
	-v go-build-cache:/root/.cache/go-build \
	-w /src golang:1.25-alpine

# host migrate binary (static) — работает с PG на host network :5433
DOCKER_MIGRATE = docker run --rm --network=host \
	-v /usr/local/bin/migrate:/migrate:ro \
	-v "$(CURDIR)/migrations:/migrations:ro" \
	alpine:latest \
	/migrate -path /migrations -database "$(DB_URL)"

up-db:
	docker compose up -d postgres

up:
	docker compose up -d postgres

down:
	docker compose down

migrate: up-db
	@echo "waiting for postgres..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do \
		docker compose exec -T postgres pg_isready -U helpdesk -d helpdesk -p 5433 >/dev/null 2>&1 && break; \
		sleep 1; \
	done
	$(DOCKER_MIGRATE) up

test:
	$(DOCKER_GO) sh -c 'go test $$(go list ./... | grep -v /self-educate/) -count=1'

test-race:
	docker run --rm --network=host \
		-v "$(CURDIR):/src" \
		-v go-mod-cache:/go/pkg/mod \
		-v go-build-cache:/root/.cache/go-build \
		-w /src -e CGO_ENABLED=1 golang:1.25 \
		sh -c 'go test $$(go list ./... | grep -v /self-educate/) -count=1 -race'

build:
	$(DOCKER_GO) go build -o /tmp/help-desk ./cmd/server

run: up-db migrate
	DATABASE_URL=$(DB_URL) JWT_SECRET=dev-secret SEED_DEMO=true APP_ENV=dev HTTP_ADDR=:8080 \
	$(DOCKER_GO) sh -c 'go run ./cmd/server'

test-integration: up-db migrate
	DATABASE_URL=$(DB_URL) JWT_SECRET=dev-secret SEED_DEMO=true APP_ENV=dev $(DOCKER_GO) sh -c 'go test $$(go list ./... | grep -v /self-educate/) -count=1 -tags=integration -race'

lint:
	$(DOCKER_GO) sh -c 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 && golangci-lint run $$(go list ./... | grep -v /self-educate/)'

vuln:
	$(DOCKER_GO) sh -c 'go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck $$(go list ./... | grep -v /self-educate/)'
