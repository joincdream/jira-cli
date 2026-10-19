.PHONY: build install clean test test-unit test-verbose test-pretty test-web test-report test-cover complexity complexity-top complexity-cyclo complexity-cognit

BINARY_NAME=jira
BIN_DIR=../../bin

# Complexity thresholds (fail if exceeded)
CYCLO_MAX ?= 40
COGNIT_MAX ?= 70

build:
	@mkdir -p $(BIN_DIR)
	@echo "🔨 Building Jira CLI..."
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd
	@echo "✅ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

install: build
	@echo "📦 Installing to ~/.local/bin/$(BINARY_NAME)..."
	install -m 755 $(BIN_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	@echo "✅ Installed: $(HOME)/.local/bin/$(BINARY_NAME)"

clean:
	rm -rf $(BIN_DIR)/$(BINARY_NAME) coverage.out coverage.html test-report.html

test: test-unit complexity

test-unit:
	@echo "🧪 Running unit tests (clean failure-focused mode)..."
	go test ./...

test-verbose:
	@echo "🧪 Running unit tests (verbose mode)..."
	go test -v ./...

test-pretty:
	@echo "🧪 Running unit tests with gotestsum..."
	@go run gotest.tools/gotestsum@latest --format testdox

test-cover:
	@echo "📊 Coverage summary by package:"
	go test -cover ./...

test-report:
	@echo "📑 Generating QA Test Execution Dashboard (test-report.html)..."
	@go test -json ./... | go run github.com/vakenbolt/go-test-report@latest -o test-report.html
	@echo "✅ QA Dashboard generated: test-report.html"
	@if command -v xdg-open > /dev/null 2>&1; then \
		xdg-open test-report.html > /dev/null 2>&1 || true; \
	elif command -v open > /dev/null 2>&1; then \
		open test-report.html > /dev/null 2>&1 || true; \
	fi

test-web: test-report

complexity: complexity-cyclo complexity-cognit
	@echo "✅ All complexity checks passed!"

complexity-cyclo:
	@echo "🔍 Checking cyclomatic complexity (gocyclo <= $(CYCLO_MAX))..."
	@go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -ignore "_test\.go" -over $(CYCLO_MAX) .

complexity-cognit:
	@echo "🔍 Checking cognitive complexity (gocognit <= $(COGNIT_MAX))..."
	@go run github.com/uudashr/gocognit/cmd/gocognit@latest -ignore "_test\.go" -over $(COGNIT_MAX) .

complexity-top:
	@echo "📊 Top 10 Cyclomatic Complexity Functions (gocyclo):"
	@go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -ignore "_test\.go" -top 10 .
	@echo ""
	@echo "📊 Top 10 Cognitive Complexity Functions (gocognit):"
	@go run github.com/uudashr/gocognit/cmd/gocognit@latest -ignore "_test\.go" -top 10 .
