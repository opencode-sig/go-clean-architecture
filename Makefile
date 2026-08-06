.PHONY: build-web build-server build dev

build-web:
	cd web && npm install && npm run build

build-server:
	go build -o bin/server ./cmd/server/

build: build-web build-server

dev-web:
	cd web && npm run dev

dev-server:
	go run ./cmd/server/
