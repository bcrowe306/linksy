.PHONY: all build test dev-backend dev-frontend clean

all: test build

build: build-frontend build-backend

build-frontend:
	@echo "==> Building Vue 3 frontend..."
	cd web && npm run build

build-backend:
	@echo "==> Building Go binary..."
	go build -o bin/linksy cmd/api/main.go

test:
	@echo "==> Running backend tests..."
	go test -v -race ./...

dev-backend:
	@echo "==> Starting Go Fiber dev server..."
	go run cmd/api/main.go

dev-frontend:
	@echo "==> Starting Vite dev server..."
	cd web && npm run dev

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf bin/ web/dist/ storage/
