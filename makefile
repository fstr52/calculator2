.PHONY: build run-orchestrator run-worker run-all

build: bin/orchestrator bin/worker

bin/orchestrator: ./cmd/orchestrator/*.go
        go build -o bin/orchestrator ./cmd/orchestrator

bin/worker: ./cmd/agent/*.go
        go build -o bin/agent ./cmd/agent

run-orchestrator: build
        docker build -t myproject-orchestrator -f docker/orchestrator.Dockerfile .
        docker run -p 8000:8000 myproject-orchestrator

run-worker: build
        docker build -t myproject-agent -f docker/woragentker.Dockerfile .
        docker run -e ORCHESTRATOR_URL=http://host.docker.internal:8000 myproject-agent

run-all:
        docker-compose -f docker/docker-compose.yml up

clean:
        rm -rf bin
        docker rmi myproject-orchestrator myproject-worker || true