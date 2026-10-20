COMPOSE := docker compose -f site/compose.yaml
API_DIR := site/api

.PHONY: up down build deploy logs ps test psql

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

build:
	$(COMPOSE) build

deploy: check
	$(COMPOSE) up -d --build

logs:
	$(COMPOSE) logs -f --tail 50 $(s)

ps:
	$(COMPOSE) ps

test:
	cd $(API_DIR) && go vet ./... && go test ./...

psql:
	$(COMPOSE) exec db psql -U benw weibelpainting
