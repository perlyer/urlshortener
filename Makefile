DATABASE_URL ?= postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable

.PHONY: db-up db-down migrate-up migrate-down sqlc test tidy run-api run-redirector

## поднять Postgres в docker
db-up:
	docker compose -f deploy/docker-compose.yml up -d postgres redis kafka prometheus grafana

## остановить и удалить контейнеры
db-down:
	docker compose -f deploy/docker-compose.yml down

## применить миграции
migrate-up:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

## откатить последнюю миграцию
migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

## сгенерировать типобезопасный код из SQL
sqlc:
	go tool sqlc generate

## прогнать все тесты
test:
	go test ./...

## подчистить go.mod / go.sum
tidy:
	go mod tidy

## запустить api-сервис
run-api:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

## запустить redirector-сервис
run-redirector:
	DATABASE_URL="$(DATABASE_URL)" PORT=8081 go run ./cmd/redirector

## запустить analytics-consumer
run-analytics:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/analytics

## скачать/обновить GeoIP-базу (DB-IP Lite, без регистрации)
geoip-fetch:
	mkdir -p deploy/geoip
	curl -sfL -o deploy/geoip/dbip-country-lite.mmdb.gz "https://download.db-ip.com/free/dbip-country-lite-$(shell date +%Y-%m).mmdb.gz"
	gunzip -f deploy/geoip/dbip-country-lite.mmdb.gz

## создать топик clicks заранее (не полагаемся на auto-create)
kafka-topic:
	docker exec shortener-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic clicks --partitions 1 --replication-factor 1
