########################################################################################

DESTDIR?=
PREFIX?=/usr

########################################################################################

.DEFAULT_GOAL := help
.PHONY = all clean install uninstall deps fmt

########################################################################################

all: rbinstall rbinstall-gen rbinstall-clone ## Build all binaries

rbinstall: ## Build rbinstall binary
	go build rbinstall.go

rbinstall-gen: ## Build rbinstall-gen binary
	go build rbinstall-gen.go

rbinstall-clone: ## Build rbinstall-clone binary
	go build rbinstall-clone.go

deps: ## Download dependencies
	git config --global http.https://pkg.re.followRedirects true
	go get -v pkg.re/essentialkaos/ek.v9
	go get -v pkg.re/essentialkaos/z7.v6
	go get -v pkg.re/essentialkaos/go-linenoise.v3
	go get -v pkg.re/cheggaaa/pb.v1

fmt: ## Format source code with gofmt
	find . -name "*.go" -exec gofmt -s -w {} \;

install: ## Install binaries and config
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	mkdir -p $(DESTDIR)/etc
	cp rbinstall $(DESTDIR)$(PREFIX)/bin/
	cp rbinstall-gen $(DESTDIR)$(PREFIX)/bin/
	cp rbinstall-clone $(DESTDIR)$(PREFIX)/bin/
	cp common/rbinstall.knf $(DESTDIR)/etc/

uninstall: ## Uninstall binaries and config
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall-gen
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall-clone
	rm -f $(DESTDIR)/etc/rbinstall.knf

clean: ## Remove generated files
	rm -f rbinstall
	rm -f rbinstall-gen
	rm -f rbinstall-clone

help: ## Show this info
	@echo -e '\nSupported targets:\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[33m%-16s\033[0m %s\n", $$1, $$2}'
	@echo -e ''
