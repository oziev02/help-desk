.PHONY: up down up-db migrate test test-race test-integration build run lint lint-fix fmt vet tidy vuln

DB_URL ?= postgres://helpdesk:helpdesk@127.0.0.1:5433/helpdesk?sslmode=disable
GO_PACKAGES ?= ./...
GOFLAGS ?= -buildvcs=false
export GOFLAGS

DOCKER_GO = docker run --rm --network=host \
	-v "$(CURDIR):/src" \
	-v go-mod-cache:/go/pkg/mod \
	-v go-build-cache:/root/.cache/go-build \
	-e GOFLAGS=$(GOFLAGS) \
	-w /src golang:1.25-alpine

DOCKER_GO_RACE = docker run --rm --network=host \
	-v "$(CURDIR):/src" \
	-v go-mod-cache:/go/pkg/mod \
	-v go-build-cache:/root/.cache/go-build \
	-e GOFLAGS=$(GOFLAGS) \
	-e CGO_ENABLED=1 \
	-w /src golang:1.25

DOCKER_LINT = docker run --rm \
	-v "$(CURDIR):/src" \
	-w /src \
	golangci/golangci-lint:v2.6.0

# host migrate binary (static) - works with PG on host network :5433
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
	$(DOCKER_GO) sh -c 'go test $(GO_PACKAGES) -count=1'

test-race:
	$(DOCKER_GO_RACE) sh -c 'go test $(GO_PACKAGES) -count=1 -race -shuffle=on'

build:
	$(DOCKER_GO) go build -o /tmp/help-desk ./cmd/server

run: up-db migrate
	DATABASE_URL=$(DB_URL) JWT_SECRET=dev-secret SEED_DEMO=true APP_ENV=dev HTTP_ADDR=:8080 \
	$(DOCKER_GO) sh -c 'go run ./cmd/server'

# Local: starts compose postgres + migrate. CI: set DATABASE_URL and skip docker deps.
test-integration:
ifndef CI
	@$(MAKE) up-db migrate
endif
	DATABASE_URL="$(or $(DATABASE_URL),$(DB_URL))" JWT_SECRET="$(or $(JWT_SECRET),dev-secret)" \
		SEED_DEMO=true APP_ENV=dev \
		$(DOCKER_GO_RACE) sh -c 'go test $(GO_PACKAGES) -count=1 -tags=integration -race'

lint:
	$(DOCKER_LINT) golangci-lint run $(GO_PACKAGES)

lint-fix:
	$(DOCKER_LINT) golangci-lint run --fix $(GO_PACKAGES)

fmt:
	$(DOCKER_LINT) golangci-lint fmt $(GO_PACKAGES)

vet:
	$(DOCKER_GO) go vet $(GO_PACKAGES)

tidy:
	$(DOCKER_GO) sh -c 'go mod tidy && go mod verify'

vuln:
	$(DOCKER_GO) sh -c 'go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck $(GO_PACKAGES)'
