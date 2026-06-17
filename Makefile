.PHONY: test-app test-app-build test-app-run test-app-stop test-app-logs test-app-clean

TEST_HOME ?= $(CURDIR)/.tmp/test-home
TEST_PORT ?= 18110
TEST_VITE_PORT ?= 9255
TEST_GOPATH ?= $(shell go env GOPATH 2>/dev/null)
WAILS3 ?= $(shell command -v wails3 2>/dev/null || printf "%s/bin/wails3" "$$(go env GOPATH 2>/dev/null)")
WAILS3_DIR := $(dir $(WAILS3))

test-app:
	GOPATH="$(TEST_GOPATH)" WAILS_VITE_PORT="$(TEST_VITE_PORT)" ./scripts/run-test-app.sh

test-app-stop:
	TEST_HOME="$(TEST_HOME)" ./scripts/run-test-app.sh stop

test-app-logs:
	TEST_HOME="$(TEST_HOME)" ./scripts/run-test-app.sh logs

test-app-build:
	./scripts/prepare-test-home.sh
	PATH="$(WAILS3_DIR):$$PATH" GOPATH="$(TEST_GOPATH)" HOME="$(TEST_HOME)" USERPROFILE="$(TEST_HOME)" XDG_CONFIG_HOME="$(TEST_HOME)/.config" "$(WAILS3)" task build

test-app-run:
	./scripts/prepare-test-home.sh
	PATH="$(WAILS3_DIR):$$PATH" GOPATH="$(TEST_GOPATH)" HOME="$(TEST_HOME)" USERPROFILE="$(TEST_HOME)" XDG_CONFIG_HOME="$(TEST_HOME)/.config" "$(WAILS3)" task run

test-app-clean:
	-chmod -R u+w "$(TEST_HOME)" 2>/dev/null || true
	rm -rf "$(TEST_HOME)"
