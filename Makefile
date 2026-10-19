COMPOSE := docker compose -f site/compose.yaml
.PHONY: up down build logs ps

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

build:
	$(COMPOSE) build

logs:
	$(COMPOSE) logs -f --tail 50 $(s)

ps:
	$(COMPOSE) ps
