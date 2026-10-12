.PHONY: all run stop clean db-restore

DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/motoguard?sslmode=disable
JWT_SECRET ?= motoguard-production-enterprise-secret-key-32bytes
PORT ?= 8080
MQTT_BROKER_URL ?= tcp://localhost:1883
MQTT_USERNAME ?= tracker
MQTT_PASSWORD ?= ~:!Nu2>#6TDe5Y+
FRONTEND_URL ?= https://mxguard.jullie.tech
MAILJET_API_KEY ?= b0db02dc22bbde39a369687dce33bc1c
MAILJET_SECRET_KEY ?= b5cd5867d7906029a7fd2debad0cc8a4
MAILJET_SENDER_EMAIL ?= no-reply@jullie.tech
MAILJET_REPLY_TO_EMAIL ?= hello@jullie.tech
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
