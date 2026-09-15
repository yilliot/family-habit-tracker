# Family Habit Tracker — build and run targets.
#
# The templ CLI is pinned in go.mod as a tool dependency, so `go tool templ`
# always runs the same version that generated the committed code. No Node, no
# package manager, no build step beyond Go itself.

BINARY  := habit-tracker
DB      ?= habits.sqlite
ADDR    ?= 127.0.0.1:8080
TZ      ?= Asia/Singapore
LDFLAGS := -s -w

.PHONY: help
help: ## Show the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: generate
generate: ## Regenerate Go code from the .templ templates
	go tool templ generate

.PHONY: build
build: generate ## Build the binary for this machine
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/server

.PHONY: run
run: generate ## Run the server against a local database
	go run ./cmd/server -db $(DB) -addr $(ADDR) -tz $(TZ)

.PHONY: seed
seed: ## Seed a fresh database with the four family members
	go run ./cmd/server -db $(DB) -seed

.PHONY: test
test: generate ## Run the whole test suite
	go test ./... -count=1

.PHONY: cover
cover: generate ## Report test coverage per package
	go test ./... -count=1 -cover

.PHONY: check
check: generate ## Format, vet, and test
	gofmt -l -w . | tee /dev/stderr
	go vet ./...
	go test ./... -count=1

.PHONY: dist
dist: generate ## Cross-compile static binaries for the Linux server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 ./cmd/server
	@ls -lh dist/

.PHONY: clean
clean: ## Remove build output
	rm -rf dist $(BINARY)

# Deploy to the Axon EC2. deploy/install.sh is the single source of truth for the
# port, timezone and site; these targets read the site from it.

SERVER  ?= ubuntu@18.141.199.217
SSH_KEY ?= ~/.ssh/openclaw-key.pem
SSH     := ssh -i $(SSH_KEY) $(SERVER)
SITE    := $(shell sed -n 's/^SITE=//p' deploy/install.sh)

.PHONY: deploy
deploy: check dist ## Test, build, then snapshot + install + restart on the Axon EC2
	@tmp=$$($(SSH) mktemp -d) && \
	scp -q -i $(SSH_KEY) dist/$(BINARY)-linux-amd64 $(SERVER):$$tmp/$(BINARY) && \
	scp -q -i $(SSH_KEY) deploy/install.sh deploy/habit-tracker.service deploy/nginx.conf $(SERVER):$$tmp/ && \
	$(SSH) "sudo sh $$tmp/install.sh; status=\$$?; rm -rf $$tmp; exit \$$status"

.PHONY: status
status: ## Service state, public health check, and newest snapshots on the Axon EC2
	@$(SSH) 'systemctl status habit-tracker --no-pager -n 5; sudo ls -1t /var/lib/habit-tracker/backups /var/lib/habit-tracker/pre-deploy | head -8'
	@printf 'https://$(SITE)/healthz -> '; curl -fsS https://$(SITE)/healthz; echo
