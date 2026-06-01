BINARY    := odak
ARCHER    := archer
MARKO     := marko
REMOTE_USER := cagdas
REMOTE_DIR  := /home/$(REMOTE_USER)/odak
SERVICE     := odak
INSTALL_DIR := $(HOME)/.local/bin

_TAG   := $(shell git describe --tags --abbrev=0 2>/dev/null)
_REV   := $(shell git rev-list $(_TAG)..HEAD --count 2>/dev/null || git rev-list HEAD --count 2>/dev/null || echo 0)
_HASH  := $(shell git rev-parse --short HEAD 2>/dev/null || echo pre)
_DIRTY := $(shell git diff --quiet && git diff --cached --quiet || echo +dirty)
VERSION := $(if $(_TAG),$(_TAG)+$(_REV)-$(_HASH)$(_DIRTY),v0+$(_REV)-$(_HASH)$(_DIRTY))
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

.PHONY: build web-build build-archer deploy install-service logs status restart clean

build: web-build
	go build $(LDFLAGS) -o bin/$(BINARY) .
	@mkdir -p $(INSTALL_DIR)
	@cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "installed $(INSTALL_DIR)/$(BINARY)  [$(VERSION)]"

web-build:
	cd web && npm install --silent && npm run build

build-archer:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux .

deploy: web-build build-archer
	go build $(LDFLAGS) -o bin/$(BINARY) .
	@mkdir -p $(INSTALL_DIR)
	@cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	ssh $(REMOTE_USER)@$(ARCHER) "mkdir -p $(REMOTE_DIR)"
	rsync -av bin/$(BINARY)-linux $(REMOTE_USER)@$(ARCHER):$(REMOTE_DIR)/$(BINARY)
	ssh $(REMOTE_USER)@$(ARCHER) "systemctl --user restart $(SERVICE)"
	rsync -av bin/$(BINARY)-linux $(REMOTE_USER)@$(MARKO):~/.local/bin/$(BINARY)
	@echo "deployed to $(ARCHER) + $(MARKO) + installed locally  [$(VERSION)]"

# First-time setup: copy and enable the systemd service.
# Fill in deploy/odak.service from the template before running this.
install-service:
	@test -f deploy/odak.service || (echo "copy deploy/odak.service.template → deploy/odak.service and fill in secrets" && exit 1)
	ssh $(REMOTE_USER)@$(ARCHER) "mkdir -p $(REMOTE_DIR) ~/.config/systemd/user"
	rsync -av deploy/odak.service $(REMOTE_USER)@$(ARCHER):~/.config/systemd/user/$(SERVICE).service
	ssh $(REMOTE_USER)@$(ARCHER) "systemctl --user daemon-reload && systemctl --user enable --now $(SERVICE)"

logs:
	ssh $(REMOTE_USER)@$(ARCHER) "journalctl --user -u $(SERVICE) -f"

status:
	ssh $(REMOTE_USER)@$(ARCHER) "systemctl --user status $(SERVICE)"

restart:
	ssh $(REMOTE_USER)@$(ARCHER) "systemctl --user restart $(SERVICE)"

clean:
	rm -rf bin/ web/dist/
