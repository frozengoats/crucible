GO_IMAGE := golang:1.24-alpine
GO_RUN := docker run --rm -e APP_TEST_NAME=hello -e CGO_ENABLED=0 -e HOME=$$HOME -v $$HOME:$$HOME -u $(shell id -u):$(shell id -g) -v $(shell pwd):/build -w /build $(GO_IMAGE) go
GO_RUN_TEST := docker run  --network host --rm -e APP_TEST_NAME=hello -e CGO_ENABLED=0 -e HOME=$$HOME -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock -v $$HOME:$$HOME -v $(shell pwd):/build -w /build $(GO_IMAGE) go test
GO_FILES := $(shell find . -type f -path **/*.go -not -path "./vendor/*")
PACKAGES := $(shell go list ./...)

.PHONY: test
test:
	$(GO_RUN_TEST) $(PACKAGES)

.PHONY: lint-check
lint-check:
	docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.1.1 golangci-lint run

.PHONY: build
build: bin/crucible

.PHONY: run
run: bin/crucible
	./bin/crucible abc defg

bin/crucible: $(GO_FILES)
	$(GO_RUN) build -trimpath -ldflags="-s -w" -mod=vendor -o ./bin/crucible main.go

.PHONY: clean
clean:
	rm -rf bin