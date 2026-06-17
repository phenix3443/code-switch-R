.PHONY: test-app test-app-build test-app-run test-app-clean

TEST_HOME ?= $(CURDIR)/.tmp/test-home
TEST_PORT ?= 18110
WAILS3 ?= $(shell command -v wails3 2>/dev/null || printf "%s/bin/wails3" "$$(go env GOPATH 2>/dev/null)")
WAILS3_DIR := $(dir $(WAILS3))

test-app:
	./scripts/run-test-app.sh

test-app-build:
	./scripts/prepare-test-home.sh
	PATH="$(WAILS3_DIR):$$PATH" HOME="$(TEST_HOME)" USERPROFILE="$(TEST_HOME)" XDG_CONFIG_HOME="$(TEST_HOME)/.config" "$(WAILS3)" task build

test-app-run:
	./scripts/prepare-test-home.sh
	PATH="$(WAILS3_DIR):$$PATH" HOME="$(TEST_HOME)" USERPROFILE="$(TEST_HOME)" XDG_CONFIG_HOME="$(TEST_HOME)/.config" "$(WAILS3)" task run

test-app-clean:
	rm -rf "$(TEST_HOME)"
