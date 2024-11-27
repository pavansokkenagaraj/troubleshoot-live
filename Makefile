GORELEASER_PARALLELISM ?= $(shell nproc --ignore=1)
GORELEASER_DEBUG ?= false
export DOCKER_REGISTRY ?= ghcr.io
export DOCKERHUB_ORG ?= mhrabovcin
export GIT_TREE_STATE ?=

.PHONY: gogenerate
gogenerate:
	go generate ./...

.PHONY: test
test: tools.gotestsum gogenerate
	gotestsum --format pkgname --junitfile unit-tests.xml --jsonfile test.json -- -coverprofile=cover.out ./... && \
		go tool cover -func=cover.out

.PHONY: lint
lint:
	helm lint --strict ./charts/troubleshoot-live
	golangci-lint run --fix

ifndef GORELEASER_CURRENT_TAG
export GORELEASER_CURRENT_TAG=$(GIT_TAG)
endif

.PHONY: build-snapshot
build-snapshot:
	goreleaser  \
		build \
		--snapshot \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		$(if $(BUILD_ALL),,--single-target) \


.PHONY: release
release:
	goreleaser  \
		release \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--timeout=60m \
		$(GORELEASER_FLAGS)

.PHONY: release-snapshot
release-snapshot:
	goreleaser  \
		release \
		--snapshot \
		--skip-publish \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--timeout=60m

.PHONY: tools.gotestsum
tools.gotestsum:
	go install gotest.tools/gotestsum@v1.10.0
