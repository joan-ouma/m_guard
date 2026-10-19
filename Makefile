.PHONY: all run stop clean db-restore

-include .env
-include backend/.env

DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/motoguard?sslmode=disable
JWT_SECRET ?= dev-secret-key-change-in-production
PORT ?= 8080
MQTT_BROKER_URL ?= tcp://localhost:1883
MQTT_USERNAME ?=
MQTT_PASSWORD ?=
FRONTEND_URL ?= http://localhost:4200
MAILJET_API_KEY ?=
MAILJET_SECRET_KEY ?=
MAILJET_SENDER_EMAIL ?=
MAILJET_REPLY_TO_EMAIL ?=
MAILJET_SENDER_NAME ?= MGuard
LOG_FILE ?= ./logs/app.log

all: run

run:
	@echo "==> Starting database, MQTT broker, Backend & Frontend..."
	@docker compose up --build -d db mqtt backend frontend
	@until docker exec motoguard-db pg_isready -U postgres -d motoguard > /dev/null 2>&1; do sleep 1; done
	@if [ -f motoguard_backup.sql ] && ! docker exec motoguard-db psql -U postgres -d motoguard -c "SELECT 1 FROM bikes LIMIT 1;" > /dev/null 2>&1; then \
		echo "==> Initializing fresh database from motoguard_backup.sql..."; \
		docker exec -i motoguard-db psql -U postgres -d motoguard < motoguard_backup.sql > /dev/null 2>&1; \
	fi
	@echo "==> Application is running! Frontend is accessible at http://localhost:4200"

db-restore:
	@echo "==> Restoring database from motoguard_backup.sql..."
	@docker exec -i motoguard-db psql -U postgres -d motoguard < motoguard_backup.sql
	@echo "==> Database restore complete."

stop:
	@echo "==> Stopping all services..."
	@docker compose stop db mqtt backend frontend
	@pkill -f "./server" || true
	@pkill -f "ng serve" || true
	@echo "==> Stopped."

clean: stop
	@rm -f backend/server
	@rm -rf frontend/dist

DB_USER ?= postgres
DB_PASSWORD ?= postgres

migrate:
	@echo "==> Running Liquibase migrations..."
	@docker run --rm --network="host" -v $$(pwd)/backend/liquibase:/liquibase/changelog liquibase/liquibase --changelog-file=changelog-master.xml --url="jdbc:postgresql://localhost:5433/motoguard" --username=$(DB_USER) --password=$(DB_PASSWORD) update

migrate-status:
	@echo "==> Checking Liquibase migration status..."
	@docker run --rm --network="host" -v $$(pwd)/backend/liquibase:/liquibase/changelog liquibase/liquibase --changelog-file=changelog-master.xml --url="jdbc:postgresql://localhost:5433/motoguard" --username=$(DB_USER) --password=$(DB_PASSWORD) status
