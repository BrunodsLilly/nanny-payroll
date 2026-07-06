MODULES := ./domain/... ./ports/... ./app/... ./adapters/sqlite/... ./cmd/cli/...
DB ?= nannypayroll.db

.PHONY: build test test-domain test-functional lint vet fmt fmt-check tidy run-cli list-cli clean help

help:
	@echo "Targets:"
	@echo "  build           build all modules"
	@echo "  test            run domain + functional tests"
	@echo "  test-domain     run pure domain unit tests"
	@echo "  test-functional run end-to-end tests against real SQLite"
	@echo "  vet             go vet all modules"
	@echo "  lint            golangci-lint all modules"
	@echo "  fmt             gofmt -w all modules"
	@echo "  fmt-check       fail if any file is not gofmt'd"
	@echo "  tidy            go mod tidy in every module"
	@echo "  run-cli         go run ./cmd/cli (pass ARGS=\"...\")"
	@echo "  clean           remove build artifacts and the local dev DB"

build:
	go build $(MODULES)

test: test-domain test-functional

test-domain:
	go test ./domain/...

test-functional:
	go test ./functional_tests/...

vet:
	go vet $(MODULES)

lint:
	golangci-lint run $(MODULES)

fmt:
	gofmt -w -l $(shell go list -f '{{.Dir}}' $(MODULES))

fmt-check:
	@diff="$$(gofmt -l $(shell go list -f '{{.Dir}}' $(MODULES)))"; \
	if [ -n "$$diff" ]; then \
		echo "Files not gofmt'd:"; echo "$$diff"; exit 1; \
	fi

tidy:
	for mod in domain ports app adapters/sqlite cmd/cli functional_tests; do \
		(cd $$mod && go mod tidy); \
	done

run-cli:
	go run ./cmd/cli $(ARGS) -db $(DB)

list-cli:
	go run ./cmd/cli list -db $(DB)

clean:
	go clean $(MODULES)
	rm -f $(DB)
