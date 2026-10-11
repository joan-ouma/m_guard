.PHONY: all run stop clean db-restore

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/motoguard?sslmode=disable
JWT_SECRET ?= motoguard-production-enterprise-secret-key-32bytes
PORT ?= 8080
CORS_ORIGINS ?= http://localhost:4200
MQTT_BROKER_URL ?= tcp://localhost:1883
MQTT_USERNAME ?= tracker
MQTT_PASSWORD ?= ~:!Nu2>#6TDe5Y+
FRONTEND_URL ?= http://localhost:4200
MAILJET_API_KEY ?= b0db02dc22bbde39a369687dce33bc1c
MAILJET_SECRET_KEY ?= b5cd5867d7906029a7fd2debad0cc8a4
MAILJET_SENDER_EMAIL ?= no-reply@jullie.tech
MAILJET_REPLY_TO_EMAIL ?= hello@jullie.tech
MAILJET_SENDER_NAME ?= MGuard

all: run

run:
	@echo "==> Starting database & MQTT broker..."
	@docker compose up -d db mqtt
	@until docker exec motoguard-db pg_isready -U postgres -d motoguard > /dev/null 2>&1; do sleep 1; done
	@if [ -f motoguard_backup.sql ] && ! docker exec motoguard-db psql -U postgres -d motoguard -c "SELECT 1 FROM bikes LIMIT 1;" > /dev/null 2>&1; then \
		echo "==> Initializing fresh database from motoguard_backup.sql..."; \
		docker exec -i motoguard-db psql -U postgres -d motoguard < motoguard_backup.sql > /dev/null 2>&1; \
	fi
	@echo "==> Building backend..."
	@cd backend && go build -o server ./cmd/server
	@if [ ! -d frontend/node_modules ]; then \
		echo "==> Installing frontend dependencies..."; \
		cd frontend && npm install; \
	fi
	@echo "==> Running Backend (:8080) & Frontend (:4200)..."
	@bash -c 'trap "kill 0" EXIT INT TERM; \
	  (cd backend && DATABASE_URL="$(DATABASE_URL)" JWT_SECRET="$(JWT_SECRET)" PORT=$(PORT) CORS_ORIGINS="$(CORS_ORIGINS)" MQTT_BROKER_URL="$(MQTT_BROKER_URL)" MQTT_USERNAME="$(MQTT_USERNAME)" MQTT_PASSWORD="$(MQTT_PASSWORD)" FRONTEND_URL="$(FRONTEND_URL)" MAILJET_API_KEY="$(MAILJET_API_KEY)" MAILJET_SECRET_KEY="$(MAILJET_SECRET_KEY)" MAILJET_SENDER_EMAIL="$(MAILJET_SENDER_EMAIL)" MAILJET_REPLY_TO_EMAIL="$(MAILJET_REPLY_TO_EMAIL)" MAILJET_SENDER_NAME="$(MAILJET_SENDER_NAME)" ./server) & \
	  (cd frontend && npm start -- --proxy-config proxy.conf.json --host 0.0.0.0 --port 4200) & \
	  wait'

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
