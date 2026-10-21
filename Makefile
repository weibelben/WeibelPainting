COMPOSE := docker compose -f site/compose.yaml
API_DIR := site/api
WEB_DIR := site/web

.PHONY: up down build deploy dev logs ps test psql

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

build:
	$(COMPOSE) build

deploy:
	$(COMPOSE) up -d --build

dev:
	cd $(WEB_DIR) && npm run dev -- --host

logs:
	$(COMPOSE) logs -f --tail 50 $(s)

ps:
	$(COMPOSE) ps

test:
	cd $(API_DIR) && go vet ./... && go test ./...

psql:
	$(COMPOSE) exec db psql -U benw weibelpainting
