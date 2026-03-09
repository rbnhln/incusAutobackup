# Include variables from the .envrc file
-include .envrc

# Detect operating system
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
        DETECTED_ARCH = AMD64
    endif
    ifeq ($(PROCESSOR_ARCHITECTURE),x86)
        DETECTED_ARCH = IA32
    endif
else
    # Use 'uname -s' to get the system name (Linux, Darwin, BSD, etc.)
    # The 'sh -c' part handles cases where uname might not be directly available
    DETECTED_OS := $(shell sh -c 'uname -s 2>/dev/null || echo Unknown')
    UNAME_P := $(shell uname -p)
    ifeq ($(UNAME_P),x86_64)
        DETECTED_ARCH = AMD64
    endif
    ifneq ($(filter %86,$(UNAME_P)),)
        DETECTED_ARCH = AMD64
    endif
    ifneq ($(filter arm%,$(UNAME_P)),)
        DETECTED_ARCH = ARM
    endif
endif

# Convert to lowercase for easier comparisons (optional)
DETECTED_OS := $(shell echo $(DETECTED_OS) | tr A-Z a-z)
DETECTED_ARCH := $(shell echo $(DETECTED_ARCH) | tr A-Z a-z)

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: run the application
.PHONY: run
run:
	go run ./cmd 

## nfrun: run without flags
.PHONY: nfrun
nfrun:
	go run ./cmd

## onboard: use onboarding function with predefined values
.PHONY: onboard
onboard:
	go run ./cmd onboard --sourceURL ${SOURCEURL} --sourceToken ${SOURCETOKEN} --targetURL ${TARGETURL} --targetToken ${TARGETTOKEN} --iabCredDir ${CREDDIR}

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #
## tidy: tidy module dependencies and format all .go files
.PHONY: tidy
tidy:
	@echo 'Tidying module dependencies...'
	go mod tidy
	@echo 'Verifying and vendoring module dependencies...'
	go mod verify
	go mod vendor
	@echo 'Formatting .go files...'
	go fmt ./...

## audit: run quality control checks
.PHONY: audit
audit:
	@echo 'Checking module dependencies...'
	go mod tidy -diff
	go mod verify
	@echo 'Vetting code...'
	go vet ./...
	go tool staticcheck ./...
	go tool govulncheck
	@echo 'Running tests...'
	go test -race -vet=off ./...
	@echo 'Running golangci-lint...'
	golangci-lint run ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #
## build/: build the cmd/ application
.PHONY: build
build:
	@echo 'Building ....'
	GOARCH=$(DETECTED_ARCH) GOOS=$(DETECTED_OS) CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags='-s -w' -o=./iab ./cmd/iab

.PHONY: build debug
build_debug:
	@echo 'Building ....'
	go build -ldflags='-s' -o=./bin/iab ./cmd

.PHONY: build_x86_64
build_x86_64:
	@echo 'Building for x86_64 ....'
	GOARCH=amd64 GOOS=$(DETECTED_OS) CGO_ENABLED=0 go build -ldflags='-s' -trimpath -o=./bin/iab_x86 ./cmd

.PHONY: build_docker_tar
build_docker_tar:
	@echo 'Building for x86_64 in Docker ....'
	docker buildx build --platform linux/amd64 -t iab:test . --output type=docker,dest=./bin/iab.tar
