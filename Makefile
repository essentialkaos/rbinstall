########################################################################################

DESTDIR?=
PREFIX?=/usr

########################################################################################

.PHONY = all clean install uninstall deps fmt

########################################################################################

all: rbinstall rbinstall-gen rbinstall-clone

rbinstall:
	go build rbinstall.go

rbinstall-gen:
	go build rbinstall-gen.go

rbinstall-clone:
	go build rbinstall-clone.go

deps:
	go get -v pkg.re/essentialkaos/ek.v7
	go get -v pkg.re/essentialkaos/z7.v4
	go get -v pkg.re/essentialkaos/go-linenoise.v3
	go get -v pkg.re/cheggaaa/pb.v1

fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	mkdir -p $(DESTDIR)/etc
	cp rbinstall $(DESTDIR)$(PREFIX)/bin/
	cp rbinstall-gen $(DESTDIR)$(PREFIX)/bin/
	cp rbinstall-clone $(DESTDIR)$(PREFIX)/bin/
	cp common/rbinstall.knf $(DESTDIR)/etc/

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall-gen
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall-clone
	rm -f $(DESTDIR)/etc/rbinstall.knf

clean:
	rm -f rbinstall
	rm -f rbinstall-gen
	rm -f rbinstall-clone
