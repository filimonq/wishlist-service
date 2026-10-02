.PHONY: build release up scale migrate down logs lint test swag k6 mocks check-version check-release
.DEFAULT_GOAL := up

VERSION ?=
REPLICAS ?= 2
IMAGE := filimonq/wishlist-service
RELEASE_ENV := .env.$(VERSION)
COMPOSE := env -u HTTP_PORT -u HOST_PORT -u POSTGRES_DB -u POSTGRES_USER \
	-u POSTGRES_PASSWORD -u DB_CONN -u JWT_SECRET -u MIGRATIONS_PATH \
	VERSION="$(VERSION)" RELEASE_ENV="$(RELEASE_ENV)" \
	docker compose --env-file "$(RELEASE_ENV)"

check-version:
	@printf '%s\n' "$(VERSION)" | grep -Eq '^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$$' || \
		{ echo 'Укажите версию: VERSION=hw1-v1'; exit 1; }
	@test "$(VERSION)" != latest || { echo 'Используйте отдельную версию вместо latest'; exit 1; }

build: check-version
	@if docker image inspect "$(IMAGE):$(VERSION)" >/dev/null 2>&1; then \
		echo 'Образ с этой версией уже существует. Укажите новую версию.'; exit 1; fi
	docker build -t "$(IMAGE):$(VERSION)" .

release: check-version
	docker image inspect "$(IMAGE):$(VERSION)" >/dev/null
	@test ! -e "$(RELEASE_ENV)" || { echo 'Этот релиз уже существует. Укажите новую версию.'; exit 1; }
	@umask 077; cp .env "$(RELEASE_ENV)"

MOCKERY_VERSION := v2.53.6
SWAG_VERSION := v1.16.6
GOLANGCI_LINT_VERSION := v2.10.1
K6_VERSION := 1.4.2

MOCKERY := go run github.com/vektra/mockery/v2@$(MOCKERY_VERSION)
SWAG := go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

check-release: check-version
	@test -f "$(RELEASE_ENV)" || { echo 'Сначала выполните make release с этой версией'; exit 1; }

up: check-release
	docker image inspect "$(IMAGE):$(VERSION)" >/dev/null
	$(COMPOSE) up --no-build -d

scale: check-release
	@printf '%s\n' "$(REPLICAS)" | grep -Eq '^[1-9][0-9]*$$' || \
		{ echo 'REPLICAS должно быть положительным целым числом'; exit 1; }
	docker image inspect "$(IMAGE):$(VERSION)" >/dev/null
	$(COMPOSE) up --no-build --scale wishlist-service=$(REPLICAS) -d
	$(COMPOSE) ps wishlist-service

migrate: check-release
	docker image inspect "$(IMAGE):$(VERSION)" >/dev/null
	$(COMPOSE) run --rm migrate
 
down: check-release
	$(COMPOSE) down

logs: check-release
	$(COMPOSE) logs -f wishlist-service

lint:
	$(GOLANGCI_LINT) run

test: mocks
	go test ./... -coverprofile=coverage.out && \
	grep -vE "mock|docs|swag|_mock.go" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out && \
	rm coverage.out coverage.filtered.out

swag:
	$(SWAG) init -g cmd/wishlist/main.go -o docs

k6:
	docker run --rm --network host -e K6_WEB_DASHBOARD=true \
		-e BASE_URL -v "$(CURDIR)/k6:/scripts:ro" \
		grafana/k6:$(K6_VERSION) run /scripts/race_cond_test.js

mocks:
	$(MOCKERY) --name=WishlistRepository --dir=internal/service --output=internal/service/mocks --outpkg=mocks
	$(MOCKERY) --name=ItemRepository --dir=internal/service --output=internal/service/mocks --outpkg=mocks
	$(MOCKERY) --name=UserRepository --dir=internal/service --output=internal/service/mocks --outpkg=mocks
