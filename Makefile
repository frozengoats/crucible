VERSION = $(shell cat ./VERSION)
MAJOR_VERSION ?= $(shell echo $(VERSION) | cut -d . -f1)
GO_IMAGE := frozengoats/golang:1
GO_TEST_IMAGE := frozengoats/crucible-test:latest
GO_RUN := docker run --rm -e CGO_ENABLED=0 -e HOME=$$HOME -v $$HOME:$$HOME -u $(shell id -u):$(shell id -g) -v $(shell pwd):/build -w /build $(GO_IMAGE) go
GO_RUN_TEST := docker run  --network host --rm -e CGO_ENABLED=0 -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock -v $(shell pwd):/home/test/build -w /home/test/build $(GO_TEST_IMAGE) go test
GO_FILES := $(shell find . -type f -path **/*.go -not -path "./vendor/*")
PACKAGES := $(shell go list ./...)
DOCKER_GID := $(shell getent group docker | cut -d: -f3)
DOCKER_REPOSITORY := frozengoats/crucible

.PHONY: test
test: testcontainer
	$(GO_RUN_TEST) -p 1 --timeout 10m $(PACKAGES)

.PHONY: lint-check
lint-check:
	docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.8.0 golangci-lint run

.PHONY: build
build: build-docker

.PHONY: build-binary
build-binary: bin/crucible

.PHONY: build-docker
build-docker: build-binary
	docker build -t $(DOCKER_REPOSITORY):$(VERSION) -t $(DOCKER_REPOSITORY):$(MAJOR_VERSION) .

.PHONY: publish
publish:
	docker push $(DOCKER_REPOSITORY):$(VERSION)
	docker push $(DOCKER_REPOSITORY):$(MAJOR_VERSION)

.PHONY: run
run: bin/crucible
	./bin/crucible abc defg

bin/crucible: $(GO_FILES)
	$(GO_RUN) build -trimpath -ldflags="-s -w -X 'main.Version=$(VERSION)'" -mod=vendor -o ./bin/crucible main.go

.PHONY: install
install: bin/crucible
	sudo cp ./bin/crucible /usr/local/bin/crucible

.PHONY: clean
clean:
	rm -rf bin

.PHONY: testcontainer
testcontainer:
	cd testcontainer && docker build --build-arg GID=$(DOCKER_GID) -t $(GO_TEST_IMAGE) .
