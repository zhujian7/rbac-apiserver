.PHONY: build build-image push-image deploy clean

IMAGE_REGISTRY ?= quay.io/stolostron
IMAGE_NAME ?= auth-server
IMAGE_TAG ?= latest
IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

build:
	@echo "Building auth-server binary..."
	CGO_ENABLED=0 go build -o bin/auth-server .

build-image:
	@echo "Building Docker image $(IMAGE)..."
	docker build -t $(IMAGE) .

push-image: build-image
	@echo "Pushing image $(IMAGE)..."
	docker push $(IMAGE)

run-local:
	@echo "Running auth-server locally..."
	go run . --cluster-name=local-cluster --hub-kubeconfig=${HOME}/.kube/config --v=4

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f go.sum

tidy:
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  build         - Build auth-server binary"
	@echo "  build-image   - Build Docker image"
	@echo "  push-image    - Build and push Docker image"
	@echo "  run-local     - Run auth-server locally"
	@echo "  clean         - Clean build artifacts"
	@echo "  tidy          - Run go mod tidy"
