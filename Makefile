.PHONY: up down logs lint test swag k6 mocks

up:
	docker compose up --build -d
 
down:
	docker compose down

logs:
	docker compose logs -f app

lint:
	golangci-lint run

test:
	go test ./... -coverprofile=coverage.out && \
	grep -vE "mock|docs|swag|_mock.go" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out && \
	rm coverage.out coverage.filtered.out

swag:
	swag init -g cmd/wishlist/main.go -o docs

k6:
	K6_WEB_DASHBOARD=true k6 run k6/race_cond_test.js

mocks:
	mockery --name=WishlistRepository --dir=internal/service --output=internal/service/mocks --outpkg=mocks
	mockery --name=ItemRepository --dir=internal/service --output=internal/service/mocks --outpkg=mocks
	mockery --name=UserRepository --dir=internal/service --output=internal/service/mocks --outpkg=mocks
