TUNNEL ?= none
BASE_CMD = docker compose -f docker-compose.yml

ifeq ($(TUNNEL), none)
  COMPOSE_CMD = $(BASE_CMD)
else
  COMPOSE_CMD = $(BASE_CMD) -f tunnels/$(TUNNEL).yml
endif

.PHONY: up down restart logs status up-cloudflare up-tailscale up-wireguard up-local setup

up:
	$(COMPOSE_CMD) up -d

down:
	$(COMPOSE_CMD) down

restart:
	$(COMPOSE_CMD) restart

logs:
	$(COMPOSE_CMD) logs -f

status:
	$(COMPOSE_CMD) ps

up-cloudflare:
	$(MAKE) up TUNNEL=cloudflare

up-tailscale:
	$(MAKE) up TUNNEL=tailscale

up-wireguard:
	$(MAKE) up TUNNEL=wireguard

up-local:
	$(MAKE) up TUNNEL=none

setup:
	@mkdir -p secrets
	@if [ ! -f secrets/grafana_password.txt ]; then \
		echo "admin" > secrets/grafana_password.txt; \
		echo "Created secrets/grafana_password.txt with default password. Change it!"; \
	fi
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env from .env.example. Fill in your values."; \
	fi
