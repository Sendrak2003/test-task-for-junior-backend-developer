SWAG_VERSION := v1.16.4
SWAG_OUTPUT  := internal/transport/http/docs/openapi.json

.PHONY: swag build run docker-up docker-down

swag:
	@which swag > /dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	swag init \
		--generalInfo cmd/api/main.go \
		--output $(dir $(SWAG_OUTPUT)) \
		--outputTypes json \
		--parseDependency \
		--parseInternal
	@mv $(dir $(SWAG_OUTPUT))swagger.json $(SWAG_OUTPUT) 2>/dev/null || true

build: swag
	CGO_ENABLED=0 go build -o bin/taskservice ./cmd/api

run: swag
	go run ./cmd/api

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
