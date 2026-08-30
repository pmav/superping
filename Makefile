.PHONY: build run

IMAGE_NAME ?= superping
TAG ?= latest
PORT ?= 8080

build:
	docker build -t $(IMAGE_NAME):$(TAG) .

run:
	mkdir -p data
	docker run --rm -p $(PORT):8080 -v "$$(pwd)/data:/data" $(IMAGE_NAME):$(TAG)
