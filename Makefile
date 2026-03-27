BINARY_ENGINE = bin/engine
BINARY_ADAPTER = bin/adapter-stub
DB_DSN = mysql://root:password@tcp(localhost:3306)/event_engine

.PHONY: all build build-adapter run run-adapter test lint docker-up docker-down migrate-up migrate-down clean

all: lint test build

build:
	go build -o $(BINARY_ENGINE) ./cmd/engine/main.go

build-adapter:
	go build -o $(BINARY_ADAPTER) ./cmd/adapter-stub/main.go

run:
	go run ./cmd/engine/main.go

run-adapter:
	go run ./cmd/adapter-stub/main.go

test:
	go test ./...

lint:
	go vet ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-up:
	migrate -path migrations -database "$(DB_DSN)" up

migrate-down:
	migrate -path migrations -database "$(DB_DSN)" down

clean:
	rm -rf bin/
