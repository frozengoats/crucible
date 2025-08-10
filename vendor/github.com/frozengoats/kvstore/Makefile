GO_IMAGE := golang:1.24-alpine
GO_RUN := docker run --rm -e HOME=$$HOME -v $$HOME:$$HOME -u $(shell id -u):$(shell id -g) -v $(shell pwd):/build -w /build $(GO_IMAGE) go

.PHONY: test
test:
	$(GO_RUN) test

.PHONY: lint-check
lint-check:
	docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.1.1 golangci-lint run