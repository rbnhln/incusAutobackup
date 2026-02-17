# Include variables from the .envrc file
-include .envrc

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

# ==================================================================================== #
# BUILD
# ==================================================================================== #
## build/: build the cmd/ application
.PHONY: build
build:
	@echo 'Building ....'
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags='-s -w' -o=./iab ./cmd

.PHONY: build debug
build_debug:
	@echo 'Building ....'
	go build -ldflags='-s' -o=./bin/iab ./cmd

.PHONY: build_x86_64
build_x86_64:
	@echo 'Building for x86_64 ....'
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags='-s' -trimpath -o=./bin/iab_x86 ./cmd

.PHONY: build_docker_tar
build_docker_tar:
	@echo 'Building for x86_64 in Docker ....'
	docker buildx build --platform linux/amd64 -t iab:test . --output type=docker,dest=./bin/iab.tar
