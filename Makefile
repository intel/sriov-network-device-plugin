#
# Credit:
#   This makefile was adapted from: https://github.com/vincentbernat/hellogopher/blob/feature/glide/Makefile
#
# Go environment
GOPATH=$(CURDIR)/.gopath
GOBIN=$(CURDIR)/bin
# Go tools
GOLINT = $(GOBIN)/golint
GOCOVMERGE = $(GOBIN)/gocovmerge
GOCOV = $(GOBIN)/gocov
GOCOVXML = $(GOBIN)/gocov-xml
GO2XUNIT = $(GOBIN)/go2xunit
# Package info
BINARY_NAME=sriovdp
PACKAGE=sriov-network-device-plugin
ORG_PATH=github.com/k8snetworkplumbingwg
# Build info
BUILDDIR=$(CURDIR)/build
REPO_PATH=$(ORG_PATH)/$(PACKAGE)
BASE=$(GOPATH)/src/$(REPO_PATH)
PKGS = $(or $(PKG),$(shell cd $(BASE) && env GOPATH=$(GOPATH) go list ./... | grep -v "^$(PACKAGE)/vendor/"))
GOFILES = $(shell find . -name *.go | grep -vE "(\/vendor\/)|(_test.go)")
# Test artifacts and settings
TESTPKGS = $(shell env GOPATH=$(GOPATH) go list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS))
TIMEOUT = 15
COVERAGE_MODE = atomic
COVERAGE_PROFILE = $(COVERAGE_DIR)/profile.out
COVERAGE_XML = $(COVERAGE_DIR)/coverage.xml
COVERAGE_HTML = $(COVERAGE_DIR)/index.html
# Docker image
DOCKERFILE?=$(CURDIR)/images/Dockerfile
TAG=ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin
# Docker arguments - To pass proxy for Docker invoke it as 'make image HTTP_POXY=http://192.168.0.1:8080'
DOCKERARGS=
ifdef HTTP_PROXY
	DOCKERARGS += --build-arg http_proxy=$(HTTP_PROXY)
endif
ifdef HTTPS_PROXY
	DOCKERARGS += --build-arg https_proxy=$(HTTPS_PROXY)
endif

LDFLAGS=
ifdef STATIC
	export CGO_ENABLED=0
	LDFLAGS=-a -ldflags '-extldflags \"-static\"'
endif

export GOPATH
export GOBIN

V = 0
Q = $(if $(filter 1,$V),,@)

.PHONY: all
all: fmt lint build

$(BASE): ; $(info  Setting GOPATH...)
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

$(GOBIN):
	@mkdir -p $@

$(BUILDDIR): | $(BASE) ; $(info Creating build directory...)
	@cd $(BASE) && mkdir -p $@

build: $(BUILDDIR)/$(BINARY_NAME) | ; $(info Building $(BINARY_NAME)...) @ ## Build SR-IOV Network device plugin
	$(info Done!)

$(BUILDDIR)/$(BINARY_NAME): $(GOFILES) | $(BUILDDIR)
	@cd $(BASE)/cmd/$(BINARY_NAME) && CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILDDIR)/$(BINARY_NAME) -tags no_openssl -v

$(GOLINT): | $(BASE) ; $(info  building golint...)
	$Q go get -u golang.org/x/lint/golint

$(GOCOVMERGE): | $(BASE) ; $(info  building gocovmerge...)
	$Q go get github.com/wadey/gocovmerge

$(GOCOV): | $(BASE) ; $(info  building gocov...)
	$Q go get github.com/axw/gocov/...

$(GOCOVXML): | $(BASE) ; $(info  building gocov-xml...)
	$Q go get github.com/AlekSi/gocov-xml

$(GO2XUNIT): | $(BASE) ; $(info  building go2xunit...)
	$Q go get github.com/tebeka/go2xunit

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test-xml check test tests
test-bench:   ARGS=-run=__absolutelynothing__ -bench=. ## Run benchmarks
test-short:   ARGS=-short        ## Run only short tests
test-verbose: ARGS=-v            ## Run tests in verbose mode with coverage reporting
test-race:    ARGS=-race         ## Run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test
check test tests: fmt lint | $(BASE) ; $(info  running $(NAME:%=% )tests...) @ ## Run tests
	$Q cd $(BASE) && go test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

test-xml: fmt lint | $(BASE) $(GO2XUNIT) ; $(info  running $(NAME:%=% )tests...) @ ## Run tests with xUnit output
	$Q cd $(BASE) && 2>&1 go test -timeout 20s -v $(TESTPKGS) | tee test/tests.output
	$(GO2XUNIT) -fail -input test/tests.output -output test/tests.xml

.PHONY: test-coverage test-coverage-tools
test-coverage-tools: | $(GOCOVMERGE) $(GOCOV) $(GOCOVXML)
test-coverage: COVERAGE_DIR := $(CURDIR)/test/coverage.$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
test-coverage: fmt lint test-coverage-tools | $(BASE) ; $(info  Running coverage tests...) @ ## Run coverage tests
	$Q mkdir -p $(COVERAGE_DIR)/coverage
	$Q cd $(BASE) && for pkg in $(TESTPKGS); do \
		go test \
			-coverpkg=$$(go list -f '{{ join .Deps "\n" }}' $$pkg | \
					grep '^$(PACKAGE)/' | grep -v '^$(PACKAGE)/vendor/' | \
					tr '\n' ',')$$pkg \
			-covermode=$(COVERAGE_MODE) \
			-coverprofile="$(COVERAGE_DIR)/coverage/`echo $$pkg | tr "/" "-"`.cover" $$pkg ;\
	 done
	$Q $(GOCOVMERGE) $(COVERAGE_DIR)/coverage/*.cover > $(COVERAGE_PROFILE)
	$Q go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	$Q $(GOCOV) convert $(COVERAGE_PROFILE) | $(GOCOVXML) > $(COVERAGE_XML)

.PHONY: lint
lint: | $(BASE) $(GOLINT) ; $(info  Running golint...) @ ## Run golint on all source files
	$Q cd $(BASE) && ret=0 && for pkg in $(PKGS); do \
		test -z "$$($(GOLINT) $$pkg | tee /dev/stderr)" || ret=1 ; \
	 done ; exit $$ret

.PHONY: fmt
fmt: ; $(info  running gofmt...) @ ## Run gofmt on all source files
	@ret=0 && for d in $$(go list -f '{{.Dir}}' ./... | grep -v /vendor/); do \
		gofmt -l -w $$d/*.go || ret=$$? ; \
	 done ; exit $$ret

.PHONY: deps-update
deps-update: ; $(info  Updating dependencies...) @ ## Update dependencies
	@go mod tidy && go mod vendor

.PHONY: image
image: | $(BASE) ; $(info Building Docker image...) @ ## Build SR-IOV Network device plugin docker image
	@docker build -t $(TAG) -f $(DOCKERFILE)  $(CURDIR) $(DOCKERARGS)

.PHONY: clean
clean: ; $(info  Cleaning...) @ ## Cleanup everything
	@go clean --modcache --cache --testcache
	@rm -rf $(GOPATH)
	@rm -rf $(BUILDDIR)
	@rm -rf $(GOBIN)
	@rm -rf test/

.PHONY: help
help: ; @ ## Display this help message
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
