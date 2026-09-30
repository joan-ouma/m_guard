.PHONY: all run stop clean db-restore

DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/motoguard?sslmode=disable
JWT_SECRET ?= motoguard-production-enterprise-secret-key-32bytes
PORT ?= 8080
CORS_ORIGINS ?= http://localhost:4200
MQTT_BROKER_URL ?= tcp://localhost:1883

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
	@docker compose stop db mqtt
	@pkill -f "./server" || true
	@pkill -f "ng serve" || true
	@echo "==> Stopped."

clean: stop
	@rm -f backend/server
	@rm -rf frontend/dist
