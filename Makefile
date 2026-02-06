.PHONY: help dev dev-backend dev-frontend build build-backend build-frontend clean test lint batch-cleanup batch-fetch batch-live install

# デフォルトターゲット
help:
	@echo "Pixicast Development Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make dev              - Start both backend and frontend dev servers"
	@echo "  make dev-backend      - Start backend server only (port 8080)"
	@echo "  make dev-frontend     - Start frontend dev server only (port 3000)"
	@echo ""
	@echo "Build:"
	@echo "  make build            - Build both backend and frontend"
	@echo "  make build-backend    - Build backend only"
	@echo "  make build-frontend   - Build frontend only"
	@echo ""
	@echo "Batch Jobs:"
	@echo "  make batch-cleanup    - Run cleanup anonymous users job"
	@echo "  make batch-fetch      - Run fetch videos job"
	@echo "  make batch-live       - Run update live status job"
	@echo ""
	@echo "Testing & Linting:"
	@echo "  make test             - Run all tests"
	@echo "  make test-backend     - Run backend tests"
	@echo "  make lint             - Run linters"
	@echo "  make lint-backend     - Run Go linter"
	@echo "  make lint-frontend    - Run frontend linter"
	@echo ""
	@echo "Utilities:"
	@echo "  make install          - Install all dependencies"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make migrate          - Run database migrations"

# Development
dev:
	@echo "Starting development servers..."
	@make -j2 dev-backend dev-frontend

dev-backend:
	@echo "Starting backend server on port 8080..."
	@cd backend && go run cmd/server/main.go

dev-frontend:
	@echo "Starting frontend dev server on port 3000..."
	@cd frontend && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	@echo "Building backend..."
	@cd backend && go build -o bin/server cmd/server/main.go
	@cd backend && go build -o bin/cleanup_anonymous cmd/batch/cleanup_anonymous/cleanup_anonymous.go
	@cd backend && go build -o bin/fetch_videos cmd/batch/fetch_videos/fetch_videos.go
	@cd backend && go build -o bin/update_live_status cmd/batch/update_live_status/update_live_status.go
	@echo "Backend binaries created in backend/bin/"

build-frontend:
	@echo "Building frontend..."
	@cd frontend && npm run build
	@echo "Frontend build complete"

# Batch Jobs
batch-cleanup:
	@echo "Running cleanup anonymous users job..."
	@cd backend && go run cmd/batch/cleanup_anonymous/cleanup_anonymous.go

batch-fetch:
	@echo "Running fetch videos job..."
	@cd backend && go run cmd/batch/fetch_videos/fetch_videos.go

batch-live:
	@echo "Running update live status job..."
	@cd backend && go run cmd/batch/update_live_status/update_live_status.go

# Testing
test: test-backend
	@echo "All tests complete"

test-backend:
	@echo "Running backend tests..."
	@cd backend && go test ./...

# Linting
lint: lint-backend lint-frontend

lint-backend:
	@echo "Running Go linter..."
	@cd backend && go vet ./...
	@cd backend && go fmt ./...

lint-frontend:
	@echo "Running frontend linter..."
	@cd frontend && npm run lint

# Utilities
install:
	@echo "Installing backend dependencies..."
	@cd backend && go mod download
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "All dependencies installed"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf backend/bin
	@rm -rf frontend/.next
	@rm -rf frontend/out
	@echo "Clean complete"

migrate:
	@echo "Running database migrations..."
	@cd backend && go run cmd/migrate/main.go
	@echo "Migrations complete"

# Run production server
prod-backend:
	@echo "Starting production backend server..."
	@cd backend && ./bin/server

prod-frontend:
	@echo "Starting production frontend server..."
	@cd frontend && npm run start
