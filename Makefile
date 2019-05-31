################################################################################

# This Makefile generated by GoMakeGen 1.0.0 using next command:
# gomakegen .
#
# More info: https://kaos.sh/gomakegen

################################################################################

.DEFAULT_GOAL := help
.PHONY = fmt all clean git-config deps help

################################################################################

all: rbinstall rbinstall-clone rbinstall-gen ## Build all binaries

rbinstall: ## Build rbinstall binary
	go build rbinstall.go

rbinstall-clone: ## Build rbinstall-clone binary
	go build rbinstall-clone.go

rbinstall-gen: ## Build rbinstall-gen binary
	go build rbinstall-gen.go

install: ## Install binaries
	cp rbinstall /usr/bin/rbinstall
	cp rbinstall-clone /usr/bin/rbinstall-clone
	cp rbinstall-gen /usr/bin/rbinstall-gen

uninstall: ## Uninstall binaries
	rm -f /usr/bin/rbinstall
	rm -f /usr/bin/rbinstall-clone
	rm -f /usr/bin/rbinstall-gen

git-config: ## Configure git redirects for stable import path services
	git config --global http.https://pkg.re.followRedirects true

deps: git-config ## Download dependencies
	go get -d -v pkg.re/cheggaaa/pb.v1
	go get -d -v pkg.re/essentialkaos/ek.v10
	go get -d -v pkg.re/essentialkaos/z7.v8

fmt: ## Format source code with gofmt
	find . -name "*.go" -exec gofmt -s -w {} \;

clean: ## Remove generated files
	rm -f rbinstall
	rm -f rbinstall-clone
	rm -f rbinstall-gen

help: ## Show this info
	@echo -e '\nSupported targets:\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[33m%-12s\033[0m %s\n", $$1, $$2}'
	@echo -e ''

################################################################################
