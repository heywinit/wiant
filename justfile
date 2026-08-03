# Global settings
set dotenv-load
set shell := ["bash", "-cu"]

test_go:
  @echo "Running Go tests..."
  (cd agent && go test ./...)
  (cd server && go test ./...)

build_go:
  @echo "Building Go binaries..."
  (cd agent && go build -o ../../bin/agent ./...)
  (cd server && go build -o ../../bin/server ./...)

docker_up:
  @echo "Starting Docker services..."
  docker compose up -d db mailpit

docker_down:
  @echo "Stopping Docker services..."
  docker compose down

dev_agent:
  @echo "Running agent..."
  (cd agent && go run ./...)

dev_server:
  @echo "Running server..."
  (cd server && go run ./...)

web_install:
  @echo "Installing web deps..."
  (cd web && bun install)

dev_web:
  @echo "Starting web dev server..."
  (cd web && bun run dev)

build_web:
  @echo "Building web..."
  (cd web && bun run build)

generate:
  @echo "Generating database and API clients..."
  (cd server && sqlc generate)
  (cd web && bun run api:generate)

migrate_up:
  (cd server && goose -dir db/migrations postgres "$DATABASE_URL" up)

migrate_down:
  (cd server && goose -dir db/migrations postgres "$DATABASE_URL" down)

# Full local dev stack: agent + server + web
dev:
  @echo "Starting full dev stack (agent, server, web)..."
  # Run in background; adjust to your preference (tmux, systemd, etc.)
  (cd agent && go run ./...) &
  (cd server && go run ./...) &
  (cd web && bun run dev)

# Run only backend (agent + server)
dev_backend:
  @echo "Starting backend (agent + server)..."
  (cd agent && go run ./...) &
  (cd server && go run ./...)

# Full test suite
test: test_go

# Full build
build: build_go build_web
  @echo "Build complete. Binaries in ./bin, web in web/dist"

fmt:
  @echo "Formatting Go code..."
  (cd agent && go fmt ./...)
  (cd server && go fmt ./...)
  @echo "Formatting/linting web..."
  (cd web && bun run lint)
  (cd web && bun run format)

clean:
  @echo "Cleaning Go binaries..."
  rm -rf bin
  @echo "Cleaning web build..."
  (cd web && bun run clean || true)

default:
  @just --list --justfile {{justfile()}}
